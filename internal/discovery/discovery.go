package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner interface {
	Run(context.Context, string, []string, string, ...string) ([]byte, []byte, error)
}
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, []byte, error) {
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	c.Env = env
	var o, e bytes.Buffer
	c.Stdout = &o
	c.Stderr = &e
	err := c.Run()
	if err != nil {
		return o.Bytes(), e.Bytes(), fmt.Errorf("run %s: %w: %s", name, err, strings.TrimSpace(e.String()))
	}
	return o.Bytes(), e.Bytes(), nil
}

type Options struct {
	Workspace         string
	IncludeTests      bool
	IncludeToolModule bool
	Exclude           []string
}
type Workspace struct {
	Path    string
	Modules []string
}
type Module struct {
	Path, Dir, Version, GoVersion string
	Main                          bool
	Requires                      []Requirement
	Replacement                   *Replacement
	Owner                         string
}
type Requirement struct {
	Path, Version, Owner string
	Direct               bool
}
type Replacement struct {
	Path, Version, Dir string
}
type Package struct {
	ImportPath, Module string
	Standard, Test     bool
	Imports            []string
}
type Snapshot struct {
	Workspace Workspace
	Modules   []Module
	Packages  []Package
}
type workJSON struct {
	Use []struct {
		DiskPath string `json:"DiskPath"`
	} `json:"Use"`
}
type listModule struct {
	Path, Dir, Version, GoVersion string
	Main                          bool
	Replace                       *struct{ Path, Version, Dir string } `json:"Replace"`
	Require                       []struct {
		Path, Version string
		Indirect      bool
	} `json:"Require"`
}
type modEditJSON struct {
	Module struct {
		Path string `json:"Path"`
	} `json:"Module"`
	Go      string `json:"Go"`
	Require []struct {
		Path, Version string
		Indirect      bool
	} `json:"Require"`
}

type listPackage struct {
	ImportPath, Dir      string
	Standard             bool
	Module               *struct{ Path string }
	Imports, TestImports []string
}

func Discover(ctx context.Context, r Runner, opt Options) (Snapshot, error) {
	wp := opt.Workspace
	if wp == "" {
		b, _, e := r.Run(ctx, "", os.Environ(), "go", "env", "GOWORK")
		if e != nil {
			return Snapshot{}, e
		}
		wp = strings.TrimSpace(string(b))
	}
	if wp == "" || wp == "off" {
		return Snapshot{}, fmt.Errorf("go.work not found")
	}
	wp, _ = filepath.Abs(wp)
	root := filepath.Dir(wp)
	workspaceEnv := append(os.Environ(), "GOWORK="+wp)
	b, _, e := r.Run(ctx, root, workspaceEnv, "go", "work", "edit", "-json", wp)
	if e != nil {
		return Snapshot{}, e
	}
	var w workJSON
	if e = json.Unmarshal(b, &w); e != nil {
		return Snapshot{}, fmt.Errorf("decode workspace: %w", e)
	}
	s := Snapshot{Workspace: Workspace{Path: wp}}
	for _, u := range w.Use {
		p := u.DiskPath
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		p = canonicalPath(p)
		excluded := false
		for _, x := range opt.Exclude {
			excluded = excluded || p == canonicalPathFrom(root, x)
		}
		if excluded {
			continue
		}
		if !opt.IncludeToolModule {
			modulePath, moduleErr := modulePath(ctx, r, p)
			if moduleErr != nil {
				return Snapshot{}, moduleErr
			}
			if modulePath == SELF_MODULE_PATH {
				continue
			}
		}
		s.Workspace.Modules = append(s.Workspace.Modules, p)
	}
	for _, dir := range s.Workspace.Modules {
		moduleEnv := append(os.Environ(), "GOWORK=off")
		b, _, e = r.Run(ctx, dir, moduleEnv, "go", "mod", "edit", "-json")
		if e != nil {
			return Snapshot{}, e
		}
		var modFile modEditJSON
		if e = json.Unmarshal(b, &modFile); e != nil {
			return Snapshot{}, fmt.Errorf("decode go.mod in %s: %w", dir, e)
		}
		b, _, e = r.Run(ctx, dir, moduleEnv, "go", "list", "-m", "-json", "all")
		if e != nil {
			return Snapshot{}, e
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		for {
			var m listModule
			if e = dec.Decode(&m); e != nil {
				if e.Error() == "EOF" {
					break
				}
				return Snapshot{}, e
			}
			if m.Path == "" {
				continue
			}
			requires := []Requirement(nil)
			isWorkspaceModule := m.Main && m.Path == modFile.Module.Path
			if isWorkspaceModule {
				requires = reqs(modFile.Require)
				for i := range requires {
					requires[i].Owner = m.Path
				}
			}
			var replacement *Replacement
			if m.Replace != nil {
				replacement = &Replacement{Path: m.Replace.Path, Version: m.Replace.Version, Dir: m.Replace.Dir}
			}
			s.Modules = append(s.Modules, Module{Path: m.Path, Dir: dir, Version: m.Version, GoVersion: m.GoVersion, Main: m.Main, Requires: requires, Replacement: replacement, Owner: m.Path})
		}
		b, _, e = r.Run(ctx, dir, workspaceEnv, "go", "list", "-deps", "-json", "./...")
		if e != nil {
			return Snapshot{}, e
		}
		dec = json.NewDecoder(bytes.NewReader(b))
		for {
			var p listPackage
			if e = dec.Decode(&p); e != nil {
				if e.Error() == "EOF" {
					break
				}
				return Snapshot{}, e
			}
			if p.ImportPath == "" {
				continue
			}
			modulePath := "stdlib"
			if p.Module != nil {
				modulePath = p.Module.Path
			}
			im := append([]string{}, p.Imports...)
			if opt.IncludeTests {
				im = append(im, p.TestImports...)
			}
			s.Packages = append(s.Packages, Package{p.ImportPath, modulePath, p.Standard, false, im})
		}
	}
	return s, nil
}

const SELF_MODULE_PATH = "github.com/bizshuk/go-dependency-analysis"

func canonicalPath(path string) string {
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}

func canonicalPathFrom(root, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return canonicalPath(path)
}

func modulePath(ctx context.Context, r Runner, dir string) (string, error) {
	env := append(os.Environ(), "GOWORK=off")
	output, _, err := r.Run(ctx, dir, env, "go", "list", "-m", "-f", "{{.Path}}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func reqs(in []struct {
	Path, Version string
	Indirect      bool
}) []Requirement {
	var out []Requirement
	for _, r := range in {
		out = append(out, Requirement{Path: r.Path, Version: r.Version, Direct: !r.Indirect})
	}
	return out
}
func ParseDirectRequires(src string) []Requirement {
	var out []Requirement
	group := false
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "require (") {
			group = true
			continue
		}
		if group && t == ")" {
			group = false
			continue
		}
		if strings.HasPrefix(t, "require ") {
			t = strings.TrimSpace(strings.TrimPrefix(t, "require "))
		}
		f := strings.Fields(t)
		if len(f) >= 2 && (!strings.HasPrefix(f[0], "//")) && (group || strings.HasPrefix(strings.TrimSpace(line), "require ")) {
			out = append(out, Requirement{Path: f[0], Version: f[1], Direct: !strings.Contains(t, "// indirect")})
		}
	}
	return out
}
