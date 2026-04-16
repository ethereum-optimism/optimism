package selector

import (
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

func testPolicy(t *testing.T) *policy.Policy {
	t.Helper()
	p, err := policy.Load()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	return p
}

// testGraph builds a minimal graph: one source file, one test file,
// a coverage edge between them tagged with a profile, one check type
// node for forge-test.
func testGraph(t *testing.T) (*graph.Graph, *catalog.Catalog) {
	t.Helper()

	g := graph.NewGraph()

	if err := g.AddNode(&graph.Node{ID: "sol:src/L1/OptimismPortal.sol", Kind: graph.KindSource, Name: "OptimismPortal"}); err != nil {
		t.Fatalf("add source: %v", err)
	}
	if err := g.AddNode(&graph.Node{ID: "sol:test/L1/OptimismPortal.t.sol", Kind: graph.KindSource, Name: "OptimismPortal.t"}); err != nil {
		t.Fatalf("add test: %v", err)
	}
	if err := g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"}); err != nil {
		t.Fatalf("add check: %v", err)
	}

	// Coverage edge: test covers source lines 42-50 under profile "main".
	if err := g.AddEdge(&graph.Edge{
		From: "sol:test/L1/OptimismPortal.t.sol",
		To:   "sol:src/L1/OptimismPortal.sol",
		Kind: graph.EdgeTestedBy, Source: graph.SourceCoverage,
		Strength: 0.9, Confidence: 1.0,
		Properties: map[string]any{
			"line_ranges": [][2]int{{42, 50}},
			"profile":     "main",
		},
	}); err != nil {
		t.Fatalf("add coverage edge: %v", err)
	}

	// tested_by edge so import-fallback would see the test file.
	if err := g.AddEdge(&graph.Edge{
		From: "sol:test/L1/OptimismPortal.t.sol",
		To:   "check:forge-test",
		Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
		Strength: 0.9, Confidence: 0.8,
	}); err != nil {
		t.Fatalf("add tested_by: %v", err)
	}

	// Build the catalog through YAML Parse so the internal index is populated
	// — Parse is the only public constructor that calls buildIndex.
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
    per_unit_duration: 60
profiles:
  - name: main
`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	return g, cat
}

// TestResolve_CoverageBasedCandidate — a diff that changes a line the
// test covers should produce a Candidate with coverage provenance.
func TestResolve_CoverageBasedCandidate(t *testing.T) {
	g, cat := testGraph(t)

	diffs := []diff.FileDiff{{
		Path: "packages/contracts-bedrock/src/L1/OptimismPortal.sol",
		Hunks: []diff.Hunk{{
			NewStart: 45, NewCount: 3,
		}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t))
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	c := cands[0]
	if c.CheckID != "forge-test" {
		t.Errorf("CheckID=%q, want forge-test", c.CheckID)
	}
	if !strings.Contains(c.Scope, "OptimismPortal.t.sol") {
		t.Errorf("Scope=%q, want test path", c.Scope)
	}
	if c.Profile != "main" {
		t.Errorf("Profile=%q, want main", c.Profile)
	}
	if len(c.Provenance) == 0 {
		t.Fatal("expected provenance")
	}
	if c.Provenance[0].Source != graph.SourceCoverage {
		t.Errorf("provenance Source=%q, want coverage", c.Provenance[0].Source)
	}
	if hit, ok := c.Provenance[0].Raw["hit_lines"].(int); !ok || hit == 0 {
		t.Errorf("expected hit_lines in provenance.Raw, got %+v", c.Provenance[0].Raw)
	}
}

// TestResolve_NoIntersectionNoCandidate — a diff that changes lines
// outside any coverage range should produce zero candidates.
func TestResolve_NoIntersectionNoCandidate(t *testing.T) {
	g, cat := testGraph(t)

	diffs := []diff.FileDiff{{
		Path: "packages/contracts-bedrock/src/L1/OptimismPortal.sol",
		Hunks: []diff.Hunk{{
			NewStart: 200, NewCount: 1, // well outside 42-50
		}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t))
	// Coverage path returns nothing; import-fallback runs but won't find
	// a scope for the source file (no src/ → test/ derivation hits).
	for _, c := range cands {
		if c.Provenance[0].Source == graph.SourceCoverage {
			t.Errorf("unexpected coverage candidate for non-intersecting diff: %+v", c)
		}
	}
}

// TestResolve_BlastRadius — changing foundry.toml produces one
// candidate per check type with blast-radius provenance.
func TestResolve_BlastRadius(t *testing.T) {
	g, cat := testGraph(t)

	diffs := []diff.FileDiff{{Path: "foundry.toml"}}
	cands := Resolve(g, diffs, cat, testPolicy(t))
	if len(cands) != len(cat.CheckTypes) {
		t.Fatalf("expected 1 candidate per check type (%d), got %d", len(cat.CheckTypes), len(cands))
	}
	for _, c := range cands {
		if c.Signal != 1.0 {
			t.Errorf("blast-radius candidate should have Signal=1.0, got %f", c.Signal)
		}
		if reason, ok := c.Provenance[0].Raw["reason"]; !ok || reason != "blast_radius" {
			t.Errorf("expected blast_radius provenance, got %+v", c.Provenance[0].Raw)
		}
	}
}

// TestOptimize_PureFromCandidates — Optimizer should produce items
// without reaching for a graph. We pass a synthetic candidate list.
func TestOptimize_PureFromCandidates(t *testing.T) {
	_, cat := testGraph(t)

	cands := []Candidate{{
		CheckID: "forge-test",
		Scope:   "./test/L1/OptimismPortal.t.sol",
		Profile: "main",
		Signal:  0.95,
	}}

	pol := testPolicy(t)
	prStage, err := pol.Stage("pr")
	if err != nil {
		t.Fatalf("pr stage: %v", err)
	}
	opt := NewSimpleOptimizer(pol)
	res, err := opt.Optimize(cands, prStage, cat)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if len(res.Items) == 0 {
		t.Fatal("expected at least one execution item")
	}
	item := res.Items[0]
	if item.CheckTypeID != "forge-test" {
		t.Errorf("CheckTypeID=%q, want forge-test", item.CheckTypeID)
	}
	if item.Profile != "main" {
		t.Errorf("Profile=%q, want main", item.Profile)
	}
}
