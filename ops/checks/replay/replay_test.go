package replay

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/cihistory"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// testFixture builds a minimal graph + catalog + policy that's
// enough to exercise the replay classification logic. The graph
// wires one Solidity source file through coverage to a specific
// test, so a diff touching that source produces a forge-test
// candidate, and separately wires foundry.toml as a blast-radius
// trigger for snapshots-check.
func testFixture(t *testing.T) (*graph.Graph, *catalog.Catalog, *policy.Policy) {
	t.Helper()
	g := graph.NewGraph()

	_ = g.AddNode(&graph.Node{ID: "sol:src/L1/X.sol", Kind: graph.KindSource})
	_ = g.AddNode(&graph.Node{ID: "sol:test/L1/X.t.sol", Kind: graph.KindSource})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck})
	_ = g.AddNode(&graph.Node{ID: "check:snapshots-check", Kind: graph.KindCheck})
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck})

	// Coverage edge: X.t.sol covers X.sol
	_ = g.AddEdge(&graph.Edge{
		From: "sol:test/L1/X.t.sol", To: "sol:src/L1/X.sol",
		Kind: graph.EdgeTestedBy, Source: graph.SourceCoverage,
		Strength: 0.9, Confidence: 1.0,
		Properties: map[string]any{},
	})
	// tested_by edges from test file to checks (builder convention)
	for _, check := range []string{"check:forge-test"} {
		_ = g.AddEdge(&graph.Edge{
			From: "sol:test/L1/X.t.sol", To: check,
			Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
			Strength: 0.9, Confidence: 0.8,
		})
	}
	// Also from the src file, so binary-check paths can reach it
	for _, check := range []string{"check:forge-test", "check:snapshots-check"} {
		_ = g.AddEdge(&graph.Edge{
			From: "sol:src/L1/X.sol", To: check,
			Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
			Strength: 0.9, Confidence: 0.8,
		})
	}

	cat, err := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    scope_flag: "--match-path"
    scope_type: paths
    avg_duration: 3600
  - id: snapshots-check
    name: snapshots-check
    kind: check
    language: solidity
    command: just snapshots
    scopeable: false
    triggers: ["packages/contracts-bedrock/src/**"]
    avg_duration: 60
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_type: packages
    avg_duration: 7200
`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	pol, err := policy.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	return g, cat, pol
}

// TestRun_CaughtFailureCountsRight — a failure that the selector
// would have run is counted as "caught."
func TestRun_CaughtFailureCountsRight(t *testing.T) {
	g, cat, pol := testFixture(t)

	events := []cihistory.Event{{
		PR:    123,
		Files: []string{"packages/contracts-bedrock/src/L1/X.sol"},
		Checks: []cihistory.CheckRun{
			{ID: "forge-test", Failed: true},       // should be caught
			{ID: "snapshots-check", Failed: false}, // ran + passed, selector picks → over-run
			{ID: "go-test", Failed: false},         // ran + passed, selector shouldn't pick it for a sol diff
		},
	}}

	results, summary, err := Run(events, g, cat, pol, "pr")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	r := results[0]
	if len(r.MissedFailures) != 0 {
		t.Errorf("MissedFailures = %v, want empty (forge-test should have been selected)", r.MissedFailures)
	}
	if len(r.CaughtFailures) != 1 || r.CaughtFailures[0] != "forge-test" {
		t.Errorf("CaughtFailures = %v, want [forge-test]", r.CaughtFailures)
	}
	if summary.FailureRecall != 1.0 {
		t.Errorf("FailureRecall = %f, want 1.0", summary.FailureRecall)
	}
}

// TestRun_MissedFailureCountsRight — a failure the selector would
// NOT have selected is counted as "missed" and hits the recall
// metric. This is THE test for the tool's safety claim.
func TestRun_MissedFailureCountsRight(t *testing.T) {
	g, cat, pol := testFixture(t)

	// Go-only diff; forge-test failed in CI (unrelated to the diff,
	// flaky, or catalog/selector disagreement). Selector wouldn't
	// have picked forge-test for a Go-only change.
	events := []cihistory.Event{{
		PR:    456,
		Files: []string{"op-node/rollup/derive/batch.go"},
		Checks: []cihistory.CheckRun{
			{ID: "forge-test", Failed: true}, // failed in CI, selector wouldn't run → miss
		},
	}}

	_, summary, err := Run(events, g, cat, pol, "pr")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.MissedFailures != 1 {
		t.Errorf("MissedFailures = %d, want 1", summary.MissedFailures)
	}
	if summary.PerCheckMissed["forge-test"] != 1 {
		t.Errorf("PerCheckMissed[forge-test] = %d, want 1", summary.PerCheckMissed["forge-test"])
	}
	if summary.FailureRecall >= 1.0 {
		t.Errorf("FailureRecall = %f, should be < 1.0 given a miss", summary.FailureRecall)
	}
}

// TestRun_OverRunCountsRight — a check the selector picks but that
// passed in CI counts as an over-run.
func TestRun_OverRunCountsRight(t *testing.T) {
	g, cat, pol := testFixture(t)

	events := []cihistory.Event{{
		PR:    789,
		Files: []string{"packages/contracts-bedrock/src/L1/X.sol"},
		Checks: []cihistory.CheckRun{
			{ID: "snapshots-check", Failed: false}, // selector picks (trigger), CI passed → over-run
			{ID: "forge-test", Failed: false},      // selector picks (coverage), CI passed → over-run
		},
	}}

	_, summary, err := Run(events, g, cat, pol, "pr")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.OverRuns == 0 {
		t.Errorf("OverRuns = %d, want at least 1", summary.OverRuns)
	}
	// No failures, no misses.
	if summary.MissedFailures != 0 {
		t.Errorf("MissedFailures = %d, want 0 (no failures in event)", summary.MissedFailures)
	}
}

// TestRun_FailureRecallAcrossMultipleEvents — aggregated metrics
// across several events.
func TestRun_FailureRecallAcrossMultipleEvents(t *testing.T) {
	g, cat, pol := testFixture(t)

	events := []cihistory.Event{
		{
			PR:    1,
			Files: []string{"packages/contracts-bedrock/src/L1/X.sol"},
			Checks: []cihistory.CheckRun{
				{ID: "forge-test", Failed: true}, // caught
			},
		},
		{
			PR:    2,
			Files: []string{"packages/contracts-bedrock/src/L1/X.sol"},
			Checks: []cihistory.CheckRun{
				{ID: "forge-test", Failed: true},       // caught
				{ID: "snapshots-check", Failed: true},  // caught (trigger)
			},
		},
		{
			PR:    3,
			Files: []string{"op-node/rollup.go"},
			Checks: []cihistory.CheckRun{
				{ID: "forge-test", Failed: true}, // missed (selector wouldn't run forge for go change)
			},
		},
	}

	_, summary, err := Run(events, g, cat, pol, "pr")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.TotalFailures != 4 {
		t.Errorf("TotalFailures = %d, want 4", summary.TotalFailures)
	}
	if summary.CaughtFailures != 3 {
		t.Errorf("CaughtFailures = %d, want 3", summary.CaughtFailures)
	}
	if summary.MissedFailures != 1 {
		t.Errorf("MissedFailures = %d, want 1", summary.MissedFailures)
	}
	// 3/4 = 0.75
	if summary.FailureRecall < 0.74 || summary.FailureRecall > 0.76 {
		t.Errorf("FailureRecall = %f, want ~0.75", summary.FailureRecall)
	}
}
