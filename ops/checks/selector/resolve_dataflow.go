package selector

import (
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// matchGlob mirrors the builder's glob matcher. Duplicated here to
// avoid a builder→selector import cycle; keep in sync.
func matchGlob(pattern, path string) bool {
	if i := strings.Index(pattern, "/**/"); i != -1 {
		prefix := pattern[:i]
		rest := pattern[i+len("/**/"):]
		if !(strings.HasPrefix(path, prefix+"/") || strings.Contains(path, "/"+prefix+"/")) {
			return false
		}
		if m, _ := filepath.Match(rest, filepath.Base(path)); m {
			return true
		}
		return false
	}
	if strings.HasPrefix(pattern, "**/") {
		rest := pattern[len("**/"):]
		if m, _ := filepath.Match(rest, filepath.Base(path)); m {
			return true
		}
		return false
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix)
	}
	if m, _ := filepath.Match(pattern, path); m {
		return true
	}
	return path == pattern
}

// SelectViaDataflow is the Phase A pipeline-model selector. It walks
// the graph's consumes/produces edges to find every check that must
// run for a given diff. Unlike the existing Resolve, it doesn't layer
// coverage, import, trigger, and profile logic — it's a single
// uniform dataflow walk.
//
// Behavior vs the existing selector during Phase A:
//   - A check_type participates in dataflow selection only if it
//     declares Inputs/Outputs/Tools in the catalog. Nothing declares
//     those yet, so this function returns no candidates today.
//   - As check_types migrate (Phase B), they start appearing in the
//     dataflow output. The comparison harness (DataflowEquivalence,
//     used in tests) asserts the new walk produces the same candidate
//     CheckIDs as the existing Resolve for each migrated check.
//
// Scoping is intentionally out of scope for this walk — it returns
// unscoped Candidates. The scoping layer (coverage/import walks per
// check_type) stays as-is and applies on top of the dataflow-selected
// check_types during Phase B.
func SelectViaDataflow(g *graph.Graph, diffs []diff.FileDiff) []Candidate {
	return SelectViaDataflowWithCatalog(g, diffs, nil)
}

// SelectViaDataflowWithCatalog runs the dataflow walk with optional
// catalog-aware seeding. When the catalog is supplied, changed paths
// that don't map to any graph node (e.g. go.mod, mise.toml, .circleci
// config) are matched against check_type inputs directly, so those
// checks get selected even without a seed node in the graph.
func SelectViaDataflowWithCatalog(g *graph.Graph, diffs []diff.FileDiff, cat *catalog.Catalog) []Candidate {
	filePaths := extractPaths(diffs)
	if len(filePaths) == 0 {
		return nil
	}

	// Seed: every source node matching a changed path is invalidated.
	invalidated := make(map[string]bool)
	changedNodes, _ := diff.FilesToNodeIDs(g, filePaths)
	for _, id := range changedNodes {
		invalidated[id] = true
	}

	// Pre-seed selected checks by matching paths directly against
	// catalog check inputs. This handles files that aren't in the
	// source graph (go.mod, mise.toml, .circleci/...) but appear in
	// check.inputs as path globs.
	selected := make(map[string]bool)
	if cat != nil {
		for i := range cat.CheckTypes {
			ct := &cat.CheckTypes[i]
			for _, in := range ct.Inputs {
				if strings.HasPrefix(in, "artifact:") {
					continue
				}
				for _, path := range filePaths {
					if matchGlob(in, path) {
						selected[ct.ID] = true
						// Also invalidate this check's outputs so
						// downstream consumers propagate.
						for _, e := range g.EdgesFrom("check:" + ct.ID) {
							if e.Kind == graph.EdgeProduces && !invalidated[e.To] {
								invalidated[e.To] = true
							}
						}
						break
					}
				}
			}
		}
	}

	// Fixpoint: each invalidated node may be consumed by checks; each
	// such check is selected and its outputs become invalidated,
	// propagating further.
	//
	// The walker also follows `generates` edges forward. These are a
	// legacy mechanism (src → artifact shortcut) that will be replaced
	// in Phase D by explicit producer checks (e.g. gen-go-bindings,
	// interfaces-regen) consuming src and producing the artifact. Until
	// then, following generates preserves parity with the legacy
	// selector on chains like src/Foo.sol → op-e2e/bindings/foo.go →
	// go-test-on-bindings-consumer.
	queue := make([]string, 0, len(invalidated))
	for id := range invalidated {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		// Follow generates forward — the target artifact/source is
		// stale when this node changes.
		for _, edge := range g.EdgesFrom(node) {
			if edge.Kind != graph.EdgeGenerates {
				continue
			}
			if !invalidated[edge.To] {
				invalidated[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
		for _, edge := range g.EdgesTo(node) {
			if edge.Kind != graph.EdgeConsumes {
				continue
			}
			if !strings.HasPrefix(edge.From, "check:") {
				continue
			}
			checkID := strings.TrimPrefix(edge.From, "check:")
			if selected[checkID] {
				continue
			}
			selected[checkID] = true
			// This check's outputs are now stale.
			for _, outEdge := range g.EdgesFrom(edge.From) {
				if outEdge.Kind != graph.EdgeProduces {
					continue
				}
				if !invalidated[outEdge.To] {
					invalidated[outEdge.To] = true
					queue = append(queue, outEdge.To)
				}
			}
		}
	}

	out := make([]Candidate, 0, len(selected))
	for id := range selected {
		out = append(out, Candidate{
			CheckID: id,
			Signal:  1.0,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				Contribution: 1.0,
				Raw: map[string]any{
					"reason": "dataflow",
				},
			}},
		})
	}
	return out
}
