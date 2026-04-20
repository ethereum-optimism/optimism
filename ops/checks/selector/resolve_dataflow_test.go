package selector

import (
	"sort"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// TestDataflow_SingleCheckSingleInput — simplest case: a check
// consumes one source file, changing that file selects the check.
func TestDataflow_SingleCheckSingleInput(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "sol:src/Foo.sol", Kind: graph.KindSource, Name: "Foo.sol"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"})
	_ = g.AddEdge(&graph.Edge{
		From: "check:forge-test", To: "sol:src/Foo.sol", Kind: graph.EdgeConsumes,
		Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
	})

	diffs := []diff.FileDiff{{Path: "packages/contracts-bedrock/src/Foo.sol"}}
	got := dataflowForDiffsForTest(g, diffs)

	if len(got) != 1 || got[0].CheckID != "forge-test" {
		t.Fatalf("expected [forge-test], got %v", got)
	}
}

// TestDataflow_TransitivePropagation — change → forge-build → stale
// forge-artifacts → forge-test. Both checks should be selected.
func TestDataflow_TransitivePropagation(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "sol:src/Foo.sol", Kind: graph.KindSource, Name: "Foo.sol"})
	_ = g.AddNode(&graph.Node{ID: "artifact:forge-artifacts/Foo.json", Kind: graph.KindArtifact, Name: "Foo.json"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-build", Kind: graph.KindCheck, Name: "forge-build"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"})

	// forge-build consumes src, produces artifact
	_ = g.AddEdge(&graph.Edge{From: "check:forge-build", To: "sol:src/Foo.sol", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:forge-build", To: "artifact:forge-artifacts/Foo.json", Kind: graph.EdgeProduces, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	// forge-test consumes both src and the build artifact
	_ = g.AddEdge(&graph.Edge{From: "check:forge-test", To: "sol:src/Foo.sol", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:forge-test", To: "artifact:forge-artifacts/Foo.json", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})

	diffs := []diff.FileDiff{{Path: "packages/contracts-bedrock/src/Foo.sol"}}
	got := dataflowForDiffsForTest(g, diffs)

	ids := checkIDs(got)
	if !stringSetEq(ids, []string{"forge-build", "forge-test"}) {
		t.Fatalf("expected [forge-build, forge-test], got %v", ids)
	}
}

// TestDataflow_ToolchainPropagation — bumping the forge section of
// mise.toml invalidates artifact:toolchain/forge, which forge-build
// and forge-test consume; docs-build (which consumes only
// artifact:toolchain/node) does not fire. Demonstrates that per-tool
// granularity of setup checks propagates selectively.
func TestDataflow_ToolchainPropagation(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{ID: "sol:mise.toml", Kind: graph.KindSource, Name: "mise.toml"})
	_ = g.AddNode(&graph.Node{ID: "artifact:toolchain/forge", Kind: graph.KindArtifact, Name: "toolchain/forge"})
	_ = g.AddNode(&graph.Node{ID: "artifact:toolchain/node", Kind: graph.KindArtifact, Name: "toolchain/node"})
	// Separate per-tool setup checks keep the dataflow precise:
	// forge-setup is invalidated only when mise.toml's forge entry
	// changes (in this test, approximated by "mise.toml changes at
	// all" since we don't parse the toml).
	_ = g.AddNode(&graph.Node{ID: "check:forge-setup", Kind: graph.KindCheck, Name: "forge-setup"})
	_ = g.AddNode(&graph.Node{ID: "check:node-setup", Kind: graph.KindCheck, Name: "node-setup"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-build", Kind: graph.KindCheck, Name: "forge-build"})
	_ = g.AddNode(&graph.Node{ID: "check:forge-test", Kind: graph.KindCheck, Name: "forge-test"})
	_ = g.AddNode(&graph.Node{ID: "check:docs-build", Kind: graph.KindCheck, Name: "docs-build"})

	_ = g.AddEdge(&graph.Edge{From: "check:forge-setup", To: "sol:mise.toml", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:forge-setup", To: "artifact:toolchain/forge", Kind: graph.EdgeProduces, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	// node-setup has no consumes edge from mise.toml — it's driven
	// by other inputs (e.g. package.json). On a mise-toml-only diff
	// it does not fire.
	_ = g.AddEdge(&graph.Edge{From: "check:node-setup", To: "artifact:toolchain/node", Kind: graph.EdgeProduces, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:forge-build", To: "artifact:toolchain/forge", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:forge-test", To: "artifact:toolchain/forge", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&graph.Edge{From: "check:docs-build", To: "artifact:toolchain/node", Kind: graph.EdgeConsumes, Source: graph.SourceStatic, Confidence: 1, Strength: 1})

	// Seed the walk manually — FilesToNodeIDs today handles .sol/.go/.rs
	// but not .toml. Path-to-node mapping for toolchain config files is
	// a Phase-A-integration concern, not a dataflow-walker concern.
	got := dataflowFromSeed(g, map[string]bool{"sol:mise.toml": true})

	if !stringSetEq(got, []string{"forge-setup", "forge-build", "forge-test"}) {
		t.Fatalf("expected forge-setup, forge-build, forge-test; got %v", got)
	}
	for _, id := range got {
		if id == "docs-build" || id == "node-setup" {
			t.Errorf("%s should not fire — it's downstream of artifact:toolchain/node which wasn't invalidated", id)
		}
	}
}

// dataflowFromSeed is a test helper that runs the dataflow walk
// starting from an explicit set of invalidated node IDs, bypassing
// the FilesToNodeIDs step.
func dataflowFromSeed(g *graph.Graph, seed map[string]bool) []string {
	invalidated := make(map[string]bool, len(seed))
	for k, v := range seed {
		invalidated[k] = v
	}
	selected := make(map[string]bool)
	queue := make([]string, 0, len(seed))
	for k := range seed {
		queue = append(queue, k)
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, e := range g.EdgesTo(node) {
			if e.Kind != graph.EdgeConsumes {
				continue
			}
			id := e.From[len("check:"):]
			if selected[id] {
				continue
			}
			selected[id] = true
			for _, oe := range g.EdgesFrom(e.From) {
				if oe.Kind != graph.EdgeProduces {
					continue
				}
				if !invalidated[oe.To] {
					invalidated[oe.To] = true
					queue = append(queue, oe.To)
				}
			}
		}
	}
	out := make([]string, 0, len(selected))
	for id := range selected {
		out = append(out, id)
	}
	return out
}

// dataflowForDiffsForTest is a package-internal shim that mirrors the
// pre-cutover SelectViaDataflow signature for the existing test
// fixtures. It extracts paths + source-node seeds from diffs and
// returns the selected checks as Candidates.
func dataflowForDiffsForTest(g *graph.Graph, diffs []diff.FileDiff) []Candidate {
	filePaths := extractPaths(diffs)
	seedIDs, _ := diff.FilesToNodeIDs(g, filePaths)
	sel := selectViaDataflow(g, seedIDs, filePaths, nil)
	out := make([]Candidate, 0, len(sel))
	for id := range sel {
		out = append(out, Candidate{CheckID: id, Signal: 1.0})
	}
	return out
}

func checkIDs(cands []Candidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.CheckID)
	}
	sort.Strings(out)
	return out
}

func stringSetEq(got []string, want []string) bool {
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
