package selector

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

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

	// Fixpoint: each invalidated node may be consumed by checks; each
	// such check is selected and its outputs become invalidated,
	// propagating further.
	selected := make(map[string]bool)
	queue := make([]string, 0, len(invalidated))
	for id := range invalidated {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
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
