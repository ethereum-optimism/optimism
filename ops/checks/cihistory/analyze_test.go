package cihistory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    scope_type: paths
    avg_duration: 3600
  - id: snapshots-check
    name: snapshots-check
    kind: check
    language: solidity
    command: just snapshots-check
    scopeable: false
    avg_duration: 60
  - id: golangci-lint
    name: golangci-lint
    kind: lint
    language: go
    command: just lint-go
    scopeable: false
    avg_duration: 300
`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	return cat
}

// TestAnalyze_BasicCorrelation — 4 events where src/L1/X.sol always
// comes with forge-test failing → precision 1.0 correlation.
func TestAnalyze_BasicCorrelation(t *testing.T) {
	now := time.Now()
	events := []Event{}
	for i := 0; i < 4; i++ {
		events = append(events, Event{
			PR:       1000 + i,
			MergedAt: now.Add(-time.Duration(i) * 24 * time.Hour),
			Files:    []string{"packages/contracts-bedrock/src/L1/X.sol"},
			Checks: []CheckRun{
				{ID: "forge-test", Failed: true},
				{ID: "snapshots-check", Failed: false},
			},
		})
	}

	a := Analyze(events, testCatalog(t), Options{MinObservations: 3, MinPrecision: 0.5})

	// Expect one correlation: X.sol → forge-test, precision=1.0
	var found *Correlation
	for i := range a.Correlations {
		c := &a.Correlations[i]
		if c.File == "packages/contracts-bedrock/src/L1/X.sol" && c.CheckID == "forge-test" {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("expected X.sol→forge-test correlation, got: %+v", a.Correlations)
	}
	if found.Precision != 1.0 {
		t.Errorf("precision = %f, want 1.0", found.Precision)
	}
	if found.Observations != 4 {
		t.Errorf("observations = %d, want 4", found.Observations)
	}

	// snapshots-check never failed → should not appear as a correlation.
	for _, c := range a.Correlations {
		if c.CheckID == "snapshots-check" {
			t.Errorf("snapshots-check should not correlate (all passes)")
		}
	}
}

// TestAnalyze_FiltersBelowThreshold — correlations with too few
// observations or too low precision are dropped.
func TestAnalyze_FiltersBelowThreshold(t *testing.T) {
	events := []Event{
		{PR: 1, Files: []string{"rare.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: true}}},
		{PR: 2, Files: []string{"common.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: true}}},
		{PR: 3, Files: []string{"common.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: false}}},
		{PR: 4, Files: []string{"common.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: false}}},
		{PR: 5, Files: []string{"common.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: false}}},
		{PR: 6, Files: []string{"common.sol"}, Checks: []CheckRun{{ID: "forge-test", Failed: false}}},
	}
	a := Analyze(events, testCatalog(t), Options{MinObservations: 3, MinPrecision: 0.5})

	// rare.sol has only 1 observation → dropped by MinObservations.
	// common.sol has 5 observations but precision 0.2 → dropped by MinPrecision.
	if len(a.Correlations) != 0 {
		t.Errorf("expected 0 correlations after filtering, got: %+v", a.Correlations)
	}
}

// TestAnalyze_LearnedPriors — per-kind prior equals aggregate base rate
// when sample size meets threshold.
func TestAnalyze_LearnedPriors(t *testing.T) {
	events := []Event{}
	// 100 events: forge-test fails 30% of runs.
	for i := 0; i < 100; i++ {
		events = append(events, Event{
			PR:    i,
			Files: []string{"packages/contracts-bedrock/src/L1/X.sol"},
			Checks: []CheckRun{{
				ID:     "forge-test",
				Failed: i < 30,
			}},
		})
	}
	a := Analyze(events, testCatalog(t), Options{MinObservationsForPrior: 20})

	if p := a.PriorsByKind["test"]; p < 0.25 || p > 0.35 {
		t.Errorf("learned test prior = %f, want ~0.30", p)
	}
	if p := a.PriorsByCheck["forge-test"]; p < 0.25 || p > 0.35 {
		t.Errorf("learned forge-test prior = %f, want ~0.30", p)
	}
}

// TestAnalyze_IgnoresUnknownChecks — events referencing checks not in
// the catalog are dropped.
func TestAnalyze_IgnoresUnknownChecks(t *testing.T) {
	events := []Event{{
		PR:    1,
		Files: []string{"x.sol"},
		Checks: []CheckRun{
			{ID: "forge-test", Failed: true},
			{ID: "nonexistent-check", Failed: true},
		},
	}}
	a := Analyze(events, testCatalog(t), Options{MinObservations: 1, MinPrecision: 0.1})

	for _, c := range a.Correlations {
		if c.CheckID == "nonexistent-check" {
			t.Errorf("unknown check should not produce correlation")
		}
	}
}

// TestWriteEdges_AddsCorrelationEdges — edges from source nodes to
// check nodes are written with expected properties.
func TestWriteEdges_AddsCorrelationEdges(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "sol:src/L1/X.sol", Kind: graph.KindSource})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck})

	a := &Analysis{
		WindowStart: time.Now().Add(-30 * 24 * time.Hour),
		WindowEnd:   time.Now(),
		Correlations: []Correlation{{
			File:         "packages/contracts-bedrock/src/L1/X.sol",
			CheckID:      "forge-test",
			Observations: 10,
			Failures:     5,
			Precision:    0.5,
		}},
	}

	n, err := WriteEdges(g, a, "")
	if err != nil {
		t.Fatalf("WriteEdges: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 edge written, got %d", n)
	}

	edges := g.EdgesFrom("sol:src/L1/X.sol")
	if len(edges) != 1 {
		t.Fatalf("expected 1 outgoing edge, got %d", len(edges))
	}
	e := edges[0]
	if e.Kind != graph.EdgeObservedCorrelation {
		t.Errorf("Kind = %q, want observed_correlation", e.Kind)
	}
	if e.Source != graph.SourceCIHistory {
		t.Errorf("Source = %q, want ci_history", e.Source)
	}
	if e.Strength != 0.5 {
		t.Errorf("Strength = %f, want 0.5 (precision)", e.Strength)
	}
	if e.Confidence != 10.0/20.0 {
		t.Errorf("Confidence = %f, want 0.5 (10/20 sample)", e.Confidence)
	}
}

// TestWriteLearnedPolicy — round-trips through the policy package as a
// layered override.
func TestWriteLearnedPolicy(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "learned.yaml")
	a := &Analysis{
		WindowStart:  time.Now().Add(-30 * 24 * time.Hour),
		WindowEnd:    time.Now(),
		PriorsByKind: map[string]float64{"test": 0.42, "lint": 0.15},
	}
	if err := WriteLearnedPolicy(tmp, a); err != nil {
		t.Fatalf("WriteLearnedPolicy: %v", err)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "priors_by_kind") || !strings.Contains(content, "test: 0.42") {
		t.Errorf("learned.yaml missing expected content:\n%s", content)
	}
}
