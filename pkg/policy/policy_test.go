package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bizshuk/go-dependency-analysis/pkg/discovery"
	"github.com/bizshuk/go-dependency-analysis/pkg/graph"
)

func TestEvaluateDiagnostics(t *testing.T) {
	a := graph.Analysis{
		Modules:     []graph.ModuleNode{{Path: "a", Version: "v2"}, {Path: "b", Version: "v2"}},
		ModuleEdges: []graph.Edge{{From: "a", To: "b", Evidence: []string{"a/p -> b/q"}}, {From: "b", To: "a"}},
		Direct:      []discovery.Requirement{{Owner: "a", Path: "b", Version: "v1", Direct: true}, {Owner: "a", Path: "c", Version: "v1", Direct: true}},
		VersionSummaries: []graph.ModuleVersionSummary{{
			Family: "example.com/lib", Paths: []string{"example.com/lib"}, Requested: []string{"v1", "v2"}, Selected: []string{"v2"},
			Observations: []graph.VersionObservation{{Evidence: "a lib@v1"}, {Evidence: "b lib@v2"}},
		}},
	}
	diagnostics := Evaluate(a, Config{Forbidden: []Forbidden{{From: "a", To: "b"}}, Heavy: map[string]int{"b": 10}, Thresholds: map[string]int{"heavy_path": 5}})
	seen := map[string]bool{}
	for _, diagnostic := range diagnostics {
		seen[diagnostic.Code] = true
		if diagnostic.Provenance == "" {
			t.Fatal(diagnostic)
		}
	}
	for _, code := range []string{"module-cycle", "layer-forbidden", "heavy-path", "multi-version", "unused-direct-candidate"} {
		if !seen[code] {
			t.Errorf("missing %s", code)
		}
	}
}

func TestEvaluateMultiVersionUsesSeverityAndEvidence(t *testing.T) {
	a := graph.Analysis{VersionSummaries: []graph.ModuleVersionSummary{{
		Family: "example.com/lib", Paths: []string{"example.com/lib", "example.com/lib/v2"}, Requested: []string{"v1.2.0", "v2.0.0"},
		Selected: []string{"v1.3.0", "v2.1.0"}, Replacements: []string{"../lib-v2"},
		Observations: []graph.VersionObservation{{Evidence: "root lib@v1.2.0"}, {Evidence: "root lib/v2@v2.0.0"}},
	}}}
	diagnostics := Evaluate(a, Config{Severity: map[string]string{"multi-version": "error"}})
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if diagnostics[0].Severity != "error" || diagnostics[0].Code != "multi-version" {
		t.Fatalf("diagnostic = %#v", diagnostics[0])
	}
	evidence := strings.Join(diagnostics[0].Evidence, "\n")
	for _, expected := range []string{"example.com/lib/v2", "v1.2.0", "v2.0.0", "../lib-v2"} {
		if !strings.Contains(evidence, expected) {
			t.Errorf("evidence missing %q: %s", expected, evidence)
		}
	}
}

func TestEvaluateForbiddenPackageEdge(t *testing.T) {
	a := graph.Analysis{PackageEdges: []graph.Edge{{From: "example.com/project/internal/domain", To: "example.com/project/cmd/server", Evidence: []string{"domain -> cmd/server"}}}}
	c := Config{Forbidden: []Forbidden{{From: "example.com/project/internal/domain*", To: "example.com/project/cmd*"}}}

	diagnostics := Evaluate(a, c)
	if len(diagnostics) != 1 {
		t.Fatalf("len(Evaluate()) = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Code != "layer-forbidden" || diagnostics[0].Provenance != "policy-heuristic" {
		t.Fatalf("Evaluate() = %#v", diagnostics)
	}
}

func TestLoadRejectsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, []byte(`{"unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
}
