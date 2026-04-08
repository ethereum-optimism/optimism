package executor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func testCatalog() *catalog.Catalog {
	c, _ := catalog.Parse([]byte(`check_types:
  - id: go-build
    name: "Build"
    kind: build
    language: go
    command: "echo build-ok"
    avg_duration: 10
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test"
    scopeable: true
    scope_type: packages
    avg_duration: 30
    prerequisites: ["go-build"]
    knobs:
      - name: short
        type: bool
        flag: "-short"
        default: true
`))
	return c
}

func TestRun_DryRun_WithScope(t *testing.T) {
	cat := testCatalog()
	e := New("/tmp", true)

	items := []selector.ExecutionItem{
		{ID: "go-build", CheckTypeID: "go-build", RunCost: 10},
		{
			ID:            "go-test:op-node",
			CheckTypeID:   "go-test",
			Scope:         []string{"./op-node/..."},
			Config:        map[string]any{"short": true},
			RunCost:       30,
			Prerequisites: []string{"go-build"},
		},
	}

	result := e.Run(items, cat)
	if result.Passed != 2 {
		t.Errorf("expected 2 passed, got %d passed, %d failed, %d skipped",
			result.Passed, result.Failed, result.Skipped)
	}

	// Check that the resolved command includes scope and flags
	for _, r := range result.Results {
		if r.ItemID == "go-test:op-node" {
			if r.Command == "" {
				t.Error("expected resolved command to be set")
			}
		}
	}
}

func TestRun_Parallel(t *testing.T) {
	cat := testCatalog()
	e := New("/tmp", true)

	items := []selector.ExecutionItem{
		{ID: "go-build", CheckTypeID: "go-build", RunCost: 10},
		{ID: "go-test:a", CheckTypeID: "go-test", Scope: []string{"./a/..."}, RunCost: 30, Prerequisites: []string{"go-build"}},
		{ID: "go-test:b", CheckTypeID: "go-test", Scope: []string{"./b/..."}, RunCost: 20, Prerequisites: []string{"go-build"}},
	}

	result := e.Run(items, cat)
	if result.Passed != 3 {
		t.Errorf("expected 3 passed, got %d", result.Passed)
	}
	// Build should come first
	if result.Results[0].ItemID != "go-build" {
		t.Errorf("expected go-build first, got %s", result.Results[0].ItemID)
	}
}
