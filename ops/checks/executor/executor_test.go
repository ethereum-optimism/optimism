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
  - id: test-a
    name: "Test A"
    kind: test
    language: go
    command: "echo test-a-ok"
    avg_duration: 30
    prerequisites: ["build"]
  - id: test-b
    name: "Test B"
    kind: test
    language: go
    command: "echo test-b-ok"
    avg_duration: 20
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
		{CheckID: "check:build", RunCost: 10},
		{CheckID: "check:test-a", RunCost: 30, Prerequisites: []string{"check:build"}},
		{CheckID: "check:test-b", RunCost: 20, Prerequisites: []string{"check:build"}},
	}

	result := e.Run(selections, cat)
	if result.Passed != 3 {
		t.Errorf("expected 3 passed (dry-run), got %d", result.Passed)
	}
}

func TestRun_Parallel(t *testing.T) {
	cat := testCatalog()
	e := New("/tmp", true)

	// test-a and test-b should be in the same layer (both depend on build)
	selections := []selector.Selection{
		{CheckID: "check:build", RunCost: 10},
		{CheckID: "check:test-a", RunCost: 30, Prerequisites: []string{"check:build"}},
		{CheckID: "check:test-b", RunCost: 20, Prerequisites: []string{"check:build"}},
	}

	result := e.Run(selections, cat)
	if result.Passed != 3 {
		t.Errorf("expected 3 passed, got %d passed, %d failed, %d skipped",
			result.Passed, result.Failed, result.Skipped)
	}

	// Build should come before tests in results
	if result.Results[0].CheckID != "check:build" {
		t.Errorf("expected build first, got %s", result.Results[0].CheckID)
	}
}
