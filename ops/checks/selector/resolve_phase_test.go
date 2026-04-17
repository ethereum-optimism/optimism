package selector

import (
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
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

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())
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

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())
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
	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())
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

// TestResolve_StaleCoverageDownweighted — a coverage edge whose
// source_sha no longer matches the current file contributes only
// stale_multiplier × raw_signal, and the provenance reflects that.
func TestResolve_StaleCoverageDownweighted(t *testing.T) {
	g, cat := testGraph(t)

	// Stamp the coverage edge with a sha that definitely doesn't match
	// anything on disk.
	for _, e := range g.EdgesTo("sol:src/L1/OptimismPortal.sol") {
		if e.Source == graph.SourceCoverage {
			e.Properties["source_sha"] = "deadbeef0000000000000000000000000000dead"
		}
	}

	diffs := []diff.FileDiff{{
		Path: "packages/contracts-bedrock/src/L1/OptimismPortal.sol",
		Hunks: []diff.Hunk{{NewStart: 45, NewCount: 3}},
	}}

	// Real freshness.Checker rooted at tempdir: the stamped sha won't
	// match (no file at the expected path), so it's stale.
	root := t.TempDir()
	pol := testPolicy(t)
	fresh := freshness.New(root, pol, g)

	cands := Resolve(g, diffs, cat, pol, fresh)
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	c := cands[0]

	// Compare against fresh baseline to confirm downweighting actually
	// occurred, rather than asserting an absolute number that would
	// drift if the signal formula changes.
	freshCands := Resolve(g, diffs, cat, pol, freshness.Nop())
	if len(freshCands) != 1 {
		t.Fatalf("fresh baseline: expected 1 candidate, got %d", len(freshCands))
	}
	if c.Signal >= freshCands[0].Signal {
		t.Errorf("stale signal (%f) should be below fresh signal (%f)", c.Signal, freshCands[0].Signal)
	}
	if _, ok := c.Provenance[0].Raw["freshness"]; !ok {
		t.Errorf("expected freshness in provenance.Raw, got %+v", c.Provenance[0].Raw)
	}
}

// TestResolve_TestHelperChangeFindsConsumers — changing a Solidity
// test helper must surface every test file that imports it (directly
// or transitively) as a scope candidate. This is the regression test
// for the reverse-walk fix: the old forward-walk returned junk scopes
// (./test/helpers/*) that ran no tests.
func TestResolve_TestHelperChangeFindsConsumers(t *testing.T) {
	g := graph.NewGraph()

	// Graph topology:
	//   test/L1/OptimismPortal.t.sol ─imports─> test/helpers/Helper.sol
	//   test/L2/Bridge.t.sol         ─imports─> test/helpers/Helper.sol
	//   test/L1/Other.t.sol          (no import of Helper; should not fire)
	nodes := []string{
		"sol:test/helpers/Helper.sol",
		"sol:test/L1/OptimismPortal.t.sol",
		"sol:test/L2/Bridge.t.sol",
		"sol:test/L1/Other.t.sol",
	}
	for _, id := range nodes {
		if err := g.AddNode(&graph.Node{ID: id, Kind: graph.KindSource}); err != nil {
			t.Fatalf("add %s: %v", id, err)
		}
	}
	if err := g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck}); err != nil {
		t.Fatalf("add check: %v", err)
	}

	// importer → imported (Solidity adapter's convention)
	imports := [][2]string{
		{"sol:test/L1/OptimismPortal.t.sol", "sol:test/helpers/Helper.sol"},
		{"sol:test/L2/Bridge.t.sol", "sol:test/helpers/Helper.sol"},
	}
	for _, ie := range imports {
		if err := g.AddEdge(&graph.Edge{
			From: ie[0], To: ie[1],
			Kind:       graph.EdgeImports,
			Source:     graph.SourceStatic,
			Strength:   0.9, Confidence: 1.0,
		}); err != nil {
			t.Fatalf("add edge: %v", err)
		}
	}

	// tested_by from every source node to forge-test (like the builder wires it)
	for _, id := range nodes {
		if err := g.AddEdge(&graph.Edge{
			From: id, To: "check:forge-test",
			Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
			Strength: 0.9, Confidence: 0.8,
		}); err != nil {
			t.Fatalf("add tested_by: %v", err)
		}
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
    per_unit_duration: 60
`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	// Diff changes the helper. No hunks → no line-level info, and no
	// coverage edges point at the helper, so Phase 1 must use the
	// reverse-walk import fallback.
	diffs := []diff.FileDiff{{
		Path: "packages/contracts-bedrock/test/helpers/Helper.sol",
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	scopes := make(map[string]bool)
	for _, c := range cands {
		if c.CheckID != "forge-test" {
			continue
		}
		scopes[c.Scope] = true
	}

	if !scopes["./test/L1/OptimismPortal.t.sol"] {
		t.Errorf("expected OptimismPortal.t.sol as scope; got %v", scopes)
	}
	if !scopes["./test/L2/Bridge.t.sol"] {
		t.Errorf("expected Bridge.t.sol as scope; got %v", scopes)
	}
	if scopes["./test/L1/Other.t.sol"] {
		t.Errorf("Other.t.sol does not import Helper; should not be a scope")
	}
	if scopes["./test/helpers/*"] || scopes["./test/helpers/Helper.sol"] {
		t.Errorf("the helper file itself should not be a scope; got %v", scopes)
	}
}

// TestResolve_TransitiveHelperChain — H1 imports H2; tests import H1.
// Changing H2 must still reach the tests via the two-hop chain.
func TestResolve_TransitiveHelperChain(t *testing.T) {
	g := graph.NewGraph()

	for _, id := range []string{
		"sol:test/helpers/Base.sol",
		"sol:test/helpers/Derived.sol",
		"sol:test/L1/X.t.sol",
	} {
		_ = g.AddNode(&graph.Node{ID: id, Kind: graph.KindSource})
	}
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck})

	// Derived imports Base; X.t.sol imports Derived.
	_ = g.AddEdge(&graph.Edge{
		From: "sol:test/helpers/Derived.sol", To: "sol:test/helpers/Base.sol",
		Kind: graph.EdgeImports, Strength: 0.9, Confidence: 1.0,
	})
	_ = g.AddEdge(&graph.Edge{
		From: "sol:test/L1/X.t.sol", To: "sol:test/helpers/Derived.sol",
		Kind: graph.EdgeImports, Strength: 0.9, Confidence: 1.0,
	})
	for _, id := range []string{
		"sol:test/helpers/Base.sol",
		"sol:test/helpers/Derived.sol",
		"sol:test/L1/X.t.sol",
	} {
		_ = g.AddEdge(&graph.Edge{
			From: id, To: "check:forge-test",
			Kind: graph.EdgeTestedBy, Strength: 0.9, Confidence: 0.8,
		})
	}

	cat, _ := catalog.Parse([]byte(`
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
`))

	// Change Base — two hops away from X.t.sol.
	diffs := []diff.FileDiff{{Path: "packages/contracts-bedrock/test/helpers/Base.sol"}}
	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	found := false
	for _, c := range cands {
		if c.Scope == "./test/L1/X.t.sol" {
			found = true
		}
	}
	if !found {
		t.Errorf("transitive walk should reach X.t.sol via Derived → Base; got candidates: %+v", cands)
	}
}

// TestResolve_GoModAffectsImportingPackages — a go.mod version bump
// for module M produces candidates only for the Go packages that
// import M, not every package in the repo. End-to-end smoke of the
// graph-based blast-radius replacement.
func TestResolve_GoModAffectsImportingPackages(t *testing.T) {
	g := graph.NewGraph()

	// Three packages: uses-M, doesnt-use-M, and a bystander.
	pkgUses := "go:github.com/op/repo/opnode/rollup"
	pkgDoesnt := "go:github.com/op/repo/opbatcher/batcher"
	bystander := "go:github.com/op/repo/opprogram/client"
	modM := "mod:github.com/ethereum/go-ethereum"
	modN := "mod:github.com/stretchr/testify"

	for _, id := range []string{pkgUses, pkgDoesnt, bystander} {
		_ = g.AddNode(&graph.Node{ID: id, Kind: graph.KindSource, Granularity: "package"})
	}
	for _, id := range []string{modM, modN} {
		_ = g.AddNode(&graph.Node{ID: id, Kind: graph.KindModule, Granularity: "module"})
	}
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck, Name: "go-test"})

	// pkgUses imports M; pkgDoesnt imports N only; bystander imports nothing external.
	_ = g.AddEdge(&graph.Edge{From: pkgUses, To: modM, Kind: graph.EdgeImports, Strength: 0.8, Confidence: 1.0})
	_ = g.AddEdge(&graph.Edge{From: pkgDoesnt, To: modN, Kind: graph.EdgeImports, Strength: 0.8, Confidence: 1.0})

	// Every Go package has tested_by → go-test (builder convention).
	for _, id := range []string{pkgUses, pkgDoesnt, bystander} {
		_ = g.AddEdge(&graph.Edge{
			From: id, To: "check:go-test",
			Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
			Strength: 0.9, Confidence: 0.8,
		})
	}

	cat, _ := catalog.Parse([]byte(`
check_types:
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_flag: ""
    scope_type: packages
    avg_duration: 7200
    per_unit_duration: 60
`))

	// Diff: go.mod bumps go-ethereum version.
	diffs := []diff.FileDiff{{
		Path: "go.mod",
		Hunks: []diff.Hunk{{
			Removed: []string{"	github.com/ethereum/go-ethereum v1.14.7"},
			Added:   []string{"	github.com/ethereum/go-ethereum v1.14.8"},
		}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	scopes := make(map[string]bool)
	for _, c := range cands {
		if c.CheckID == "go-test" {
			scopes[c.Scope] = true
		}
	}

	if !scopes["./opnode/rollup/..."] {
		t.Errorf("expected opnode/rollup/... scope (importer of affected module); got %v", scopes)
	}
	if scopes["./opbatcher/batcher/..."] {
		t.Errorf("opbatcher/batcher imports testify, not go-ethereum — should not be a scope")
	}
	if scopes["./opprogram/client/..."] {
		t.Errorf("bystander package imports nothing external — should not be a scope")
	}
}

// TestResolve_CargoTomlAffectsConsumers — a Cargo.toml version bump
// for an internal crate produces candidates only for workspace
// members that import it, plus the owning crate.
func TestResolve_CargoTomlAffectsConsumers(t *testing.T) {
	g := graph.NewGraph()

	// kona-derive depends on kona-primitives. A bump to kona-derive's
	// Cargo.toml (changing a dep version) should affect kona-derive
	// itself but not unrelated crates.
	_ = g.AddNode(&graph.Node{ID: "rs:kona-derive", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": "rust/crates/derive"}})
	_ = g.AddNode(&graph.Node{ID: "rs:kona-primitives", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": "rust/crates/primitives"}})
	_ = g.AddNode(&graph.Node{ID: "rs:bystander", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": "rust/crates/bystander"}})
	_ = g.AddNode(&graph.Node{ID: "mod:alloy-primitives", Kind: graph.KindModule})
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck, Name: "go-test"})

	// kona-derive imports alloy-primitives (external)
	_ = g.AddEdge(&graph.Edge{
		From: "rs:kona-derive", To: "mod:alloy-primitives",
		Kind: graph.EdgeImports, Strength: 0.7, Confidence: 1.0,
	})

	cat, _ := catalog.Parse([]byte(`
check_types:
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_type: packages
    avg_duration: 7200
`))

	// Diff: kona-derive bumps alloy-primitives.
	diffs := []diff.FileDiff{{
		Path: "rust/crates/derive/Cargo.toml",
		Hunks: []diff.Hunk{{
			Context: []string{"[dependencies]"},
			Removed: []string{`alloy-primitives = "0.7"`},
			Added:   []string{`alloy-primitives = "0.8"`},
		}},
	}}

	// This test doesn't care which specific candidates emerge — it
	// only needs to verify the Cargo.toml change doesn't blast-radius
	// onto every check type.
	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())
	allUnscoped := true
	for _, c := range cands {
		if c.Scope != "" {
			allUnscoped = false
		}
	}
	if allUnscoped && len(cands) > 0 {
		// Every candidate unscoped = blast-radius behavior. Cargo.toml
		// with just a dep version bump must not trigger that.
		t.Errorf("expected scoped candidates (graph-walk mode), got all-unscoped blast-radius: %+v", cands)
	}
}

// TestResolve_CargoTomlForceBlastOnFeatures — [features] changes
// flip compile-time behavior across consumers; must force blast.
func TestResolve_CargoTomlForceBlastOnFeatures(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"})
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck, Name: "go-test"})

	cat, _ := catalog.Parse([]byte(`
check_types:
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    scope_type: paths
    avg_duration: 3600
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_type: packages
    avg_duration: 7200
`))

	diffs := []diff.FileDiff{{
		Path: "rust/crates/derive/Cargo.toml",
		Hunks: []diff.Hunk{{
			Context: []string{"[features]"},
			Added:   []string{`default = ["async"]`},
		}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())
	got := make(map[string]bool)
	for _, c := range cands {
		got[c.CheckID] = true
		if c.Signal != 1.0 {
			t.Errorf("blast candidate %q has signal %f, want 1.0", c.CheckID, c.Signal)
		}
	}
	if !got["forge-test"] || !got["go-test"] {
		t.Errorf("expected blast across checks; got %v", got)
	}
}

// TestResolve_GoModForceBlastOnGoVersion — bumping the `go` directive
// forces blast-radius (every check, not scoped).
func TestResolve_GoModForceBlastOnGoVersion(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck, Name: "go-test"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"})

	cat, _ := catalog.Parse([]byte(`
check_types:
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_type: packages
    avg_duration: 7200
  - id: forge-test
    name: forge-test
    kind: test
    language: solidity
    command: forge test
    scopeable: true
    scope_type: paths
    avg_duration: 3600
`))

	diffs := []diff.FileDiff{{
		Path: "go.mod",
		Hunks: []diff.Hunk{{
			Removed: []string{"go 1.23.0"},
			Added:   []string{"go 1.24.0"},
		}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	// Blast-radius: every check type in the catalog gets a candidate,
	// all with signal=1.0 and Scope="" (unscoped "run everything").
	checkIDs := make(map[string]bool)
	for _, c := range cands {
		if c.Signal != 1.0 {
			t.Errorf("blast-radius candidate %q has signal %f, want 1.0", c.CheckID, c.Signal)
		}
		if c.Scope != "" {
			t.Errorf("blast-radius candidate %q should be unscoped, got %q", c.CheckID, c.Scope)
		}
		checkIDs[c.CheckID] = true
	}
	if !checkIDs["go-test"] || !checkIDs["forge-test"] {
		t.Errorf("expected both checks in blast radius; got %v", checkIDs)
	}
}

// TestResolve_SolidityCoverageDoesNotTriggerGoTest — a Solidity
// coverage edge (sol:test → sol:src) must never produce a go-test
// candidate. Pre-fix, coverageCandidates iterated every coverage
// edge regardless of check language and scopeForCandidate happily
// used the sol: test path as the go-test scope, emitting commands
// like `go test ./test/L1/X.t.sol` that don't parse.
func TestResolve_SolidityCoverageDoesNotTriggerGoTest(t *testing.T) {
	g := graph.NewGraph()

	// Solidity source + test with a coverage edge.
	_ = g.AddNode(&graph.Node{ID: "sol:src/L1/X.sol", Kind: graph.KindSource})
	_ = g.AddNode(&graph.Node{ID: "sol:test/L1/X.t.sol", Kind: graph.KindSource})
	_ = g.AddEdge(&graph.Edge{
		From:     "sol:test/L1/X.t.sol",
		To:       "sol:src/L1/X.sol",
		Kind:     graph.EdgeTestedBy,
		Source:   graph.SourceCoverage,
		Strength: 0.9, Confidence: 1.0,
		Properties: map[string]any{"line_ranges": [][2]int{{40, 50}}},
	})
	// Both forge-test and go-test exist as check nodes, and both
	// get tested_by edges (mimicking what the builder does).
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck})
	_ = g.AddNode(&graph.Node{ID: "check:go-test", Kind: graph.KindCheck})
	for _, check := range []string{"check:forge-test", "check:go-test"} {
		_ = g.AddEdge(&graph.Edge{
			From: "sol:test/L1/X.t.sol", To: check,
			Kind: graph.EdgeTestedBy, Source: graph.SourceStatic,
			Strength: 0.9, Confidence: 0.8,
		})
	}

	cat, _ := catalog.Parse([]byte(`
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
  - id: go-test
    name: go-test
    kind: test
    language: go
    command: go test
    scopeable: true
    scope_type: packages
    avg_duration: 7200
`))

	diffs := []diff.FileDiff{{
		Path:  "packages/contracts-bedrock/src/L1/X.sol",
		Hunks: []diff.Hunk{{NewStart: 45, NewCount: 3}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	for _, c := range cands {
		if c.CheckID == "go-test" {
			t.Errorf("go-test should not receive a Solidity-coverage-derived candidate; got scope=%q", c.Scope)
		}
		if c.CheckID == "forge-test" && c.Scope != "./test/L1/X.t.sol" {
			t.Errorf("forge-test scope = %q, want ./test/L1/X.t.sol", c.Scope)
		}
	}
}

// TestResolve_CorrelationEdgeProducesBinaryCandidate — a CI-history
// correlation edge on a binary check surfaces as a Candidate carrying
// SourceCIHistory provenance, independent of coverage.
func TestResolve_CorrelationEdgeProducesBinaryCandidate(t *testing.T) {
	g, _ := testGraph(t)

	// Add a binary check and a correlation edge to it.
	if err := g.AddNode(&graph.Node{ID: "check:snapshots-check", Kind: graph.KindCheck, Name: "snapshots-check"}); err != nil {
		t.Fatalf("add check: %v", err)
	}
	if err := g.AddEdge(&graph.Edge{
		From:       "sol:src/L1/OptimismPortal.sol",
		To:         "check:snapshots-check",
		Kind:       graph.EdgeObservedCorrelation,
		Source:     graph.SourceCIHistory,
		Strength:   0.8, // precision
		Confidence: 1.0, // enough samples
		Properties: map[string]any{
			"observations": 25,
			"failures":     20,
			"precision":    0.8,
		},
	}); err != nil {
		t.Fatalf("add correlation: %v", err)
	}

	// Extend the catalog with the binary check.
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
  - id: snapshots-check
    name: snapshots-check
    kind: check
    language: solidity
    command: just snapshots-check
    scopeable: false
    avg_duration: 60
profiles:
  - name: main
`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	diffs := []diff.FileDiff{{
		Path:  "packages/contracts-bedrock/src/L1/OptimismPortal.sol",
		Hunks: []diff.Hunk{{NewStart: 45, NewCount: 1}},
	}}

	cands := Resolve(g, diffs, cat, testPolicy(t), freshness.Nop())

	var snap *Candidate
	for i := range cands {
		if cands[i].CheckID == "snapshots-check" {
			snap = &cands[i]
		}
	}
	if snap == nil {
		t.Fatalf("expected snapshots-check candidate, got: %+v", cands)
	}
	if snap.Signal != 0.8 {
		t.Errorf("signal = %f, want 0.8 (strength*confidence)", snap.Signal)
	}
	foundCI := false
	for _, p := range snap.Provenance {
		if p.Source == graph.SourceCIHistory {
			foundCI = true
		}
	}
	if !foundCI {
		t.Errorf("expected SourceCIHistory in provenance, got: %+v", snap.Provenance)
	}
}

// TestOptimize_PreservesProvenance — provenance attached to Candidates
// flows onto the emitted ExecutionItems so downstream consumers
// (explain, JSON output) can see *why* each item is in the plan.
func TestOptimize_PreservesProvenance(t *testing.T) {
	_, cat := testGraph(t)

	cands := []Candidate{{
		CheckID: "forge-test",
		Scope:   "./test/L1/A.t.sol",
		Profile: "main",
		Signal:  0.95,
		Provenance: []SignalContribution{{
			Source:       graph.SourceCoverage,
			EdgeKind:     graph.EdgeTestedBy,
			Contribution: 0.95,
			Raw:          map[string]any{"hit_lines": 4, "total_changed": 5},
		}},
	}, {
		CheckID: "forge-test",
		Scope:   "./test/L1/B.t.sol",
		Profile: "main",
		Signal:  0.85,
		Provenance: []SignalContribution{{
			Source:       graph.SourceCoverage,
			EdgeKind:     graph.EdgeTestedBy,
			Contribution: 0.85,
			Raw:          map[string]any{"hit_lines": 3, "total_changed": 5},
		}},
	}}

	pol := testPolicy(t)
	prStage, _ := pol.Stage("pr")
	res, err := NewSimpleOptimizer(pol).Optimize(cands, prStage, cat)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if len(res.Items) == 0 {
		t.Fatal("expected at least one item")
	}

	// Both candidates were in the very_high tier; their provenance
	// should be unioned onto the emitted item.
	item := res.Items[0]
	if len(item.Provenance) != 2 {
		t.Errorf("expected 2 provenance entries (one per contributing candidate), got %d: %+v",
			len(item.Provenance), item.Provenance)
	}
	for _, p := range item.Provenance {
		if p.Source != graph.SourceCoverage {
			t.Errorf("provenance Source=%q, want coverage", p.Source)
		}
		if p.Raw["hit_lines"] == nil {
			t.Errorf("provenance missing hit_lines: %+v", p.Raw)
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
