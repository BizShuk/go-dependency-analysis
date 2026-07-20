package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseDirectRequires(t *testing.T) {
	tests := []struct {
		name, src string
		want      []Requirement
	}{
		{"single", "require example.com/a v1.0.0\n", []Requirement{{Path: "example.com/a", Version: "v1.0.0", Direct: true}}},
		{"group", "require (\n example.com/a v1.0.0\n example.com/b v1.2.0 // indirect\n)\n", []Requirement{{Path: "example.com/a", Version: "v1.0.0", Direct: true}, {Path: "example.com/b", Version: "v1.2.0", Direct: false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDirectRequires(tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestParseDirectRequiresIgnoresComments(t *testing.T) {
	if got := ParseDirectRequires("// require x v1\n"); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestCanonicalPathFromResolvesRelativePathAndSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Fatal(err)
	}
	if got, want := canonicalPathFrom(root, "link"), canonicalPath(realDir); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

type fakeRunner struct {
	workspace string
	module    map[string]string
}

func (r fakeRunner) Run(_ context.Context, dir string, _ []string, name string, args ...string) ([]byte, []byte, error) {
	command := name + " " + strings.Join(args, " ")
	switch {
	case command == "go work edit -json "+r.workspace:
		return []byte(`{"Use":[{"DiskPath":"./app"},{"DiskPath":"./tool"}]}`), nil, nil
	case command == "go list -m -f {{.Path}}":
		path, ok := r.module[canonicalPath(dir)]
		if !ok {
			return nil, nil, fmt.Errorf("unexpected module dir %s", dir)
		}
		return []byte(path + "\n"), nil, nil
	case command == "go mod edit -json":
		path := r.module[canonicalPath(dir)]
		return []byte(fmt.Sprintf(`{"Module":{"Path":%q}}`, path)), nil, nil
	case command == "go list -m -json all":
		path := r.module[canonicalPath(dir)]
		return []byte(fmt.Sprintf(`{"Path":%q,"Main":true}`, path)), nil, nil
	case command == "go mod graph":
		return nil, nil, nil
	case command == "go list -deps -json ./...":
		path := r.module[canonicalPath(dir)]
		return []byte(fmt.Sprintf(`{"ImportPath":%q,"Module":{"Path":%q}}`, path, path)), nil, nil
	default:
		return nil, nil, fmt.Errorf("unexpected command %s", command)
	}
}

func TestParseModuleGraph(t *testing.T) {
	direct := map[string]bool{"example.com/a": true}
	got, err := parseModuleGraph("example.com/root example.com/a@v1.0.0\nexample.com/a@v1.0.0 example.com/b@v1.2.0\n", "example.com/root", direct)
	if err != nil {
		t.Fatal(err)
	}
	want := []ModuleRequest{
		{Origin: "example.com/root", Requester: "example.com/root", Path: "example.com/a", Version: "v1.0.0", Direct: true, Evidence: "example.com/root example.com/a@v1.0.0"},
		{Origin: "example.com/root", Requester: "example.com/a", RequesterVersion: "v1.0.0", Path: "example.com/b", Version: "v1.2.0", Evidence: "example.com/a@v1.0.0 example.com/b@v1.2.0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestParseModuleGraphRejectsMalformedLine(t *testing.T) {
	if _, err := parseModuleGraph("example.com/root", "example.com/root", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverRejectsUnsupportedInput(t *testing.T) {
	_, err := Discover(context.Background(), fakeRunner{}, Options{Workspace: filepath.Join(t.TempDir(), "modules.json")})
	if err == nil || !strings.Contains(err.Error(), "go.work or go.mod") {
		t.Fatalf("got %v", err)
	}
}

type moduleRunner struct {
	dir      string
	goMod    string
	commands []string
}

func (r *moduleRunner) Run(_ context.Context, dir string, env []string, name string, args ...string) ([]byte, []byte, error) {
	command := name + " " + strings.Join(args, " ")
	r.commands = append(r.commands, command+"|"+strings.Join(env, "\x00"))
	switch command {
	case "go list -m -f {{.Path}}":
		return []byte("example.com/root\n"), nil, nil
	case "go mod edit -json":
		return []byte(`{"Module":{"Path":"example.com/root"},"Require":[{"Path":"example.com/a","Version":"v1.0.0"}]}`), nil, nil
	case "go list -m -json all":
		return []byte("{\"Path\":\"example.com/root\",\"Main\":true}\n{\"Path\":\"example.com/a\",\"Version\":\"v1.1.0\"}\n"), nil, nil
	case "go mod graph":
		return []byte("example.com/root example.com/a@v1.0.0\n"), nil, nil
	case "go list -deps -json ./...":
		return []byte(`{"ImportPath":"example.com/root","Module":{"Path":"example.com/root"}}`), nil, nil
	default:
		return nil, nil, fmt.Errorf("unexpected command %s in %s", command, dir)
	}
}

func TestDiscoverGoMod(t *testing.T) {
	dir := t.TempDir()
	runner := &moduleRunner{dir: dir, goMod: filepath.Join(dir, "go.mod")}
	snapshot, err := Discover(context.Background(), runner, Options{Workspace: runner.goMod, IncludeToolModule: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := snapshot.Workspace.Modules, []string{canonicalPath(dir)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("modules = %#v, want %#v", got, want)
	}
	if len(snapshot.Requests) != 1 || snapshot.Requests[0].Path != "example.com/a" || !snapshot.Requests[0].Direct {
		t.Fatalf("requests = %#v", snapshot.Requests)
	}
	for _, command := range runner.commands {
		if strings.HasPrefix(command, "go mod ") || strings.HasPrefix(command, "go list ") {
			if !strings.Contains(command, "GOWORK=off") {
				t.Fatalf("command missing GOWORK=off: %s", command)
			}
		}
	}
}

func TestDiscoverExcludesSelfByModuleIdentity(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "go.work")
	appDir := filepath.Join(root, "app")
	toolDir := filepath.Join(root, "tool")
	for _, dir := range []string{appDir, toolDir} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runner := fakeRunner{workspace: workspace, module: map[string]string{
		canonicalPath(appDir):  "example.com/app",
		canonicalPath(toolDir): SELF_MODULE_PATH,
	}}

	snapshot, err := Discover(context.Background(), runner, Options{Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := snapshot.Workspace.Modules, []string{canonicalPath(appDir)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}

	snapshot, err = Discover(context.Background(), runner, Options{Workspace: workspace, IncludeToolModule: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Workspace.Modules) != 2 {
		t.Fatalf("got modules %#v", snapshot.Workspace.Modules)
	}
}
