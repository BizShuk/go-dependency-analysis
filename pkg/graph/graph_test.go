package graph

import (
	"reflect"
	"testing"

	"github.com/bizshuk/go-dependency-analysis/pkg/discovery"
)

func TestAlgorithms(t *testing.T) {
	edges := []Edge{{"a", "b", nil}, {"b", "c", nil}, {"c", "a", nil}, {"c", "d", nil}}
	if got := Cycles(edges); !reflect.DeepEqual(got, [][]string{{"a", "b", "c"}}) {
		t.Fatalf("cycles %#v", got)
	}
	if got := Reachable(edges, "a"); !reflect.DeepEqual(got, []string{"b", "c", "d"}) {
		t.Fatalf("reachable %#v", got)
	}
	if got := ShortestPath(edges, "a", "d"); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("path %#v", got)
	}
}
func TestBuildDeduplicatesAndPreservesEvidence(t *testing.T) {
	s := discovery.Snapshot{Modules: []discovery.Module{{Path: "m1"}, {Path: "m2"}}, Packages: []discovery.Package{{ImportPath: "m1/p", Module: "m1", Imports: []string{"m2/q"}}, {ImportPath: "m1/x", Module: "m1", Imports: []string{"m2/q"}}, {ImportPath: "m2/q", Module: "m2"}}}
	a := Build(s)
	if len(a.ModuleEdges) != 1 || len(a.ModuleEdges[0].Evidence) != 2 {
		t.Fatalf("edges %#v", a.ModuleEdges)
	}
}
func TestBuildVersionSummaries(t *testing.T) {
	snapshot := discovery.Snapshot{
		Modules: []discovery.Module{
			{Path: "example.com/root", Main: true, Owner: "example.com/root"},
			{Path: "example.com/lib", Version: "v1.3.0", Owner: "example.com/root"},
			{Path: "example.com/lib/v2", Version: "v2.1.0", Owner: "example.com/root", Replacement: &discovery.Replacement{Dir: "../lib-v2"}},
		},
		Requests: []discovery.ModuleRequest{
			{Origin: "example.com/root", Requester: "example.com/a", Path: "example.com/lib", Version: "v1.1.0", Evidence: "a lib@v1.1.0"},
			{Origin: "example.com/root", Requester: "example.com/b", Path: "example.com/lib", Version: "v1.2.0", Evidence: "b lib@v1.2.0"},
			{Origin: "example.com/root", Requester: "example.com/root", Path: "example.com/lib/v2", Version: "v2.0.0", Evidence: "root lib/v2@v2.0.0"},
		},
	}

	analysis := Build(snapshot)
	if len(analysis.VersionSummaries) != 1 {
		t.Fatalf("summaries = %#v", analysis.VersionSummaries)
	}
	summary := analysis.VersionSummaries[0]
	if summary.Family != "example.com/lib" {
		t.Fatalf("family = %q", summary.Family)
	}
	if want := []string{"example.com/lib", "example.com/lib/v2"}; !reflect.DeepEqual(summary.Paths, want) {
		t.Fatalf("paths = %#v, want %#v", summary.Paths, want)
	}
	if want := []string{"v1.1.0", "v1.2.0", "v2.0.0"}; !reflect.DeepEqual(summary.Requested, want) {
		t.Fatalf("requested = %#v, want %#v", summary.Requested, want)
	}
	if want := []string{"../lib-v2"}; !reflect.DeepEqual(summary.Replacements, want) {
		t.Fatalf("replacements = %#v, want %#v", summary.Replacements, want)
	}
}

func TestModuleFamilyOnlyStripsSemanticMajor(t *testing.T) {
	tests := map[string]string{
		"example.com/lib/v2":  "example.com/lib",
		"example.com/lib/v12": "example.com/lib",
		"example.com/lib/v1":  "example.com/lib/v1",
		"example.com/2024":    "example.com/2024",
	}
	for input, want := range tests {
		if got := moduleFamily(input); got != want {
			t.Errorf("moduleFamily(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSingleRequestedVersionDoesNotCreateSummary(t *testing.T) {
	snapshot := discovery.Snapshot{
		Modules:  []discovery.Module{{Path: "example.com/lib", Version: "v1.2.0", Owner: "root"}},
		Requests: []discovery.ModuleRequest{{Origin: "root", Requester: "root", Path: "example.com/lib", Version: "v1.0.0"}},
	}
	if summaries := Build(snapshot).VersionSummaries; len(summaries) != 0 {
		t.Fatalf("summaries = %#v", summaries)
	}
}

func TestBuildDeterministic(t *testing.T) {
	a := Build(discovery.Snapshot{Modules: []discovery.Module{{Path: "b"}, {Path: "a"}}})
	if a.Modules[0].Path != "a" {
		t.Fatal(a.Modules)
	}
}
