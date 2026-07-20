package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bizshuk/go-dependency-analysis/pkg/graph"
	"github.com/bizshuk/go-dependency-analysis/pkg/policy"
)

func TestRenderDeterministic(t *testing.T) {
	a := graph.Analysis{Modules: []graph.ModuleNode{{Path: "example.com/a"}, {Path: "example.com/b"}}, ModuleEdges: []graph.Edge{{From: "example.com/a", To: "example.com/b"}}}
	var x, y bytes.Buffer
	if err := RenderJSON(&x, a, nil, "  "); err != nil {
		t.Fatal(err)
	}
	if err := RenderJSON(&y, a, nil, "  "); err != nil {
		t.Fatal(err)
	}
	if x.String() != y.String() {
		t.Fatal("JSON changed")
	}
	var m bytes.Buffer
	if err := RenderMermaid(&m, a); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.String(), `m_036cbe16b0015b0a --> m_21e68f50264534ee`) {
		t.Fatal(m.String())
	}
}
func TestRenderJSONSchemaVersion(t *testing.T) {
	var out bytes.Buffer
	if err := RenderJSONWithMetadata(&out, graph.Analysis{}, nil, "", false); err != nil {
		t.Fatal(err)
	}

	var doc Document
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 2 {
		t.Fatalf("SchemaVersion = %d, want 2", doc.SchemaVersion)
	}
}

func TestTextIncludesDiagnostics(t *testing.T) {
	var b bytes.Buffer
	RenderText(&b, graph.Analysis{}, []policy.Diagnostic{{Code: "x", Severity: "warning", Message: "msg", Provenance: "policy-heuristic", Evidence: []string{"requester asks v1"}}})
	for _, expected := range []string{"policy-heuristic", "  - requester asks v1"} {
		if !strings.Contains(b.String(), expected) {
			t.Fatal(b.String())
		}
	}
}
