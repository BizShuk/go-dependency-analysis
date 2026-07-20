package policy_test

import (
	"path/filepath"
	"testing"

	"github.com/bizshuk/go-dependency-analysis/pkg/policy"
)

func TestGenericExamplesLoad(t *testing.T) {
	t.Parallel()

	examples := []string{
		"standard-layout.json",
		"clean-architecture.json",
		"hexagonal-architecture.json",
	}
	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("..", "..", "examples", name)
			if _, err := policy.Load(path); err != nil {
				t.Fatalf("Load(%q): %v", path, err)
			}
		})
	}
}
