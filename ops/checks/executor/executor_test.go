package executor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func testCatalog() *catalog.Catalog {
	c, _ := catalog.Parse([]byte(`checks:
  - id: build
    name: "Build"
    kind: build
    language: go
    command: "echo build-ok"
    avg_duration: 10
  - id: test
    name: "Test"
    kind: test
    language: go
    command: "echo test-ok"
    avg_duration: 30
    prerequisites: ["build"]
  - id: failing
    name: "Failing"
    kind: test
    language: go
    command: "exit 1"
    avg_duration: 5
`))
	return c
}

func TestRun_DryRun(t *testing.T) {
	cat := testCatalog()
	e := New("/tmp", true)

	selections := []selector.Selection{
		{CheckID: "check:build"},
		{CheckID: "check:test", Prerequisites: []string{"check:build"}},
	}

	result := e.Run(selections, cat)
	if result.Passed != 2 {
		t.Errorf("expected 2 passed (dry-run), got %d", result.Passed)
	}
	for _, r := range result.Results {
		if r.Status != StatusPassed {
			t.Errorf("expected passed, got %s for %s", r.Status, r.CheckID)
		}
	}
}

func TestRun_OrderRespected(t *testing.T) {
	cat := testCatalog()
	e := New("/tmp", true)

	// Pass test before build — topological sort should fix ordering
	selections := []selector.Selection{
		{CheckID: "check:test", Prerequisites: []string{"check:build"}},
		{CheckID: "check:build"},
	}

	result := e.Run(selections, cat)
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].CheckID != "check:build" {
		t.Error("expected build to run before test")
	}
}

func TestTopologicalSort(t *testing.T) {
	selections := []selector.Selection{
		{CheckID: "c", Prerequisites: []string{"b"}},
		{CheckID: "a"},
		{CheckID: "b", Prerequisites: []string{"a"}},
	}

	sorted := topologicalSort(selections)
	if len(sorted) != 3 {
		t.Fatalf("expected 3, got %d", len(sorted))
	}
	// a before b, b before c
	order := make(map[string]int)
	for i, s := range sorted {
		order[s.CheckID] = i
	}
	if order["a"] > order["b"] || order["b"] > order["c"] {
		t.Errorf("expected a < b < c, got %v", order)
	}
}
