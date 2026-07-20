package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
type ModuleRequest struct {
	Origin           string `json:"origin"`
	Requester        string `json:"requester"`
	RequesterVersion string `json:"requester_version,omitempty"`
	Path             string `json:"path"`
	Version          string `json:"version,omitempty"`
	Direct           bool   `json:"direct"`
	Evidence         string `json:"evidence,omitempty"`
}
type Snapshot struct {
	Workspace Workspace
	Modules   []Module
	Packages  []Package
	Requests  []ModuleRequest
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
	input := opt.Workspace
	if input == "" {
		b, _, err := r.Run(ctx, "", os.Environ(), "go", "env", "GOWORK")
		if err != nil {
			return Snapshot{}, err
		}
		input = strings.TrimSpace(string(b))
	}
	if input == "" || input == "off" {
		return Snapshot{}, fmt.Errorf("go.work not found")
	}

	input, err := filepath.Abs(input)
	if err != nil {
		return Snapshot{}, err
	}
	input = filepath.Clean(input)
	switch filepath.Base(input) {
	case "go.mod":
		dir := canonicalPath(filepath.Dir(input))
		moduleEnv := append(os.Environ(), "GOWORK=off")
		s := Snapshot{Workspace: Workspace{Path: input, Modules: []string{dir}}}
		if !opt.IncludeToolModule {
			path, moduleErr := modulePath(ctx, r, dir)
			if moduleErr != nil {
				return Snapshot{}, moduleErr
			}
			if path == SELF_MODULE_PATH {
				s.Workspace.Modules = nil
				return s, nil
			}
		}
		if excludedModule(dir, filepath.Dir(input), opt.Exclude) {
			s.Workspace.Modules = nil
			return s, nil
		}
		if err := discoverModule(ctx, r, dir, moduleEnv, moduleEnv, opt.IncludeTests, &s); err != nil {
			return Snapshot{}, err
		}
		sortRequests(s.Requests)
		return s, nil
	case "go.work":
		return discoverWorkspace(ctx, r, input, opt)
	default:
		return Snapshot{}, fmt.Errorf("analysis input must be go.work or go.mod: %s", input)
	}
}

func discoverWorkspace(ctx context.Context, r Runner, workspacePath string, opt Options) (Snapshot, error) {
	root := filepath.Dir(workspacePath)
	workspaceEnv := append(os.Environ(), "GOWORK="+workspacePath)
	b, _, err := r.Run(ctx, root, workspaceEnv, "go", "work", "edit", "-json", workspacePath)
	if err != nil {
		return Snapshot{}, err
	}
	var w workJSON
	if err = json.Unmarshal(b, &w); err != nil {
		return Snapshot{}, fmt.Errorf("decode workspace: %w", err)
	}
	s := Snapshot{Workspace: Workspace{Path: workspacePath}}
	for _, u := range w.Use {
		dir := u.DiskPath
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		dir = canonicalPath(dir)
		if excludedModule(dir, root, opt.Exclude) {
			continue
		}
		if !opt.IncludeToolModule {
			path, moduleErr := modulePath(ctx, r, dir)
			if moduleErr != nil {
				return Snapshot{}, moduleErr
			}
			if path == SELF_MODULE_PATH {
				continue
			}
		}
		s.Workspace.Modules = append(s.Workspace.Modules, dir)
	}
	for _, dir := range s.Workspace.Modules {
		moduleEnv := append(os.Environ(), "GOWORK=off")
		if err := discoverModule(ctx, r, dir, moduleEnv, workspaceEnv, opt.IncludeTests, &s); err != nil {
			return Snapshot{}, err
		}
	}
	sortRequests(s.Requests)
	return s, nil
}

func discoverModule(ctx context.Context, r Runner, dir string, moduleEnv, packageEnv []string, includeTests bool, s *Snapshot) error {
	b, _, err := r.Run(ctx, dir, moduleEnv, "go", "mod", "edit", "-json")
	if err != nil {
		return err
	}
	var modFile modEditJSON
	if err = json.Unmarshal(b, &modFile); err != nil {
		return fmt.Errorf("decode go.mod in %s: %w", dir, err)
	}
	direct := make(map[string]bool, len(modFile.Require))
	for _, requirement := range modFile.Require {
		direct[requirement.Path] = !requirement.Indirect
	}

	b, _, err = r.Run(ctx, dir, moduleEnv, "go", "list", "-m", "-json", "all")
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	for {
		var m listModule
		if err = dec.Decode(&m); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		if m.Path == "" {
			continue
		}
		requires := []Requirement(nil)
		if m.Main && m.Path == modFile.Module.Path {
			requires = reqs(modFile.Require)
			for i := range requires {
				requires[i].Owner = m.Path
			}
		}
		var replacement *Replacement
		if m.Replace != nil {
			replacement = &Replacement{Path: m.Replace.Path, Version: m.Replace.Version, Dir: m.Replace.Dir}
		}
		s.Modules = append(s.Modules, Module{Path: m.Path, Dir: m.Dir, Version: m.Version, GoVersion: m.GoVersion, Main: m.Main, Requires: requires, Replacement: replacement, Owner: modFile.Module.Path})
	}

	b, _, err = r.Run(ctx, dir, moduleEnv, "go", "mod", "graph")
	if err != nil {
		return err
	}
	requests, err := parseModuleGraph(string(b), modFile.Module.Path, direct)
	if err != nil {
		return err
	}
	s.Requests = append(s.Requests, requests...)

	b, _, err = r.Run(ctx, dir, packageEnv, "go", "list", "-deps", "-json", "./...")
	if err != nil {
		return err
	}
	dec = json.NewDecoder(bytes.NewReader(b))
	for {
		var p listPackage
		if err = dec.Decode(&p); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		if p.ImportPath == "" {
			continue
		}
		modulePath := "stdlib"
		if p.Module != nil {
			modulePath = p.Module.Path
		}
		imports := append([]string{}, p.Imports...)
		if includeTests {
			imports = append(imports, p.TestImports...)
		}
		s.Packages = append(s.Packages, Package{p.ImportPath, modulePath, p.Standard, false, imports})
	}
	return nil
}

func parseModuleGraph(src, origin string, direct map[string]bool) ([]ModuleRequest, error) {
	var requests []ModuleRequest
	for lineNumber, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("decode go mod graph line %d: %q", lineNumber+1, line)
		}
		requester, requesterVersion := splitModuleVersion(fields[0])
		path, version := splitModuleVersion(fields[1])
		if requester == "" || path == "" {
			return nil, fmt.Errorf("decode go mod graph line %d: %q", lineNumber+1, line)
		}
		requests = append(requests, ModuleRequest{Origin: origin, Requester: requester, RequesterVersion: requesterVersion, Path: path, Version: version, Direct: requester == origin && direct[path], Evidence: line})
	}
	return requests, nil
}

func splitModuleVersion(value string) (string, string) {
	index := strings.LastIndex(value, "@")
	if index < 0 {
		return value, ""
	}
	return value[:index], value[index+1:]
}

func sortRequests(requests []ModuleRequest) {
	sort.Slice(requests, func(i, j int) bool {
		a, b := requests[i], requests[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		if a.Origin != b.Origin {
			return a.Origin < b.Origin
		}
		if a.Requester != b.Requester {
			return a.Requester < b.Requester
		}
		return a.Evidence < b.Evidence
	})
}

func excludedModule(dir, root string, excludes []string) bool {
	for _, exclude := range excludes {
		if dir == canonicalPathFrom(root, exclude) {
			return true
		}
	}
	return false
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
