package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// FindCandidates performs Phase 1: reachability.
// Finds ALL check/test nodes that could possibly fail given this diff.
// Uses two complementary mechanisms:
//   - Graph walk: changed files → source nodes → follow edges → check nodes
//   - Trigger matching: catalog trigger patterns against changed file paths
//
// Blast radius: if blast radius files changed, ALL check types become candidates.
// No pruning, no cost reasoning. This is a completeness question.
func FindCandidates(
	g *graph.Graph,
	diffs []diff.FileDiff,
	cat *catalog.Catalog,
) []NodeWithSignal {
	filePaths := extractPaths(diffs)
	if len(filePaths) == 0 {
		return nil
	}

	// Blast radius: everything is a candidate
	if isBlast, _ := diff.BlastRadiusFiles(filePaths); isBlast {
		return allCheckTypesAsCandidates(cat)
	}

	// Mechanism 1: Graph walk
	changedIDs, _ := diff.FilesToNodeIDs(g, filePaths)
	graphCandidates := graphWalkCandidates(g, changedIDs)

	// Mechanism 2: Trigger matching (for binary checks with trigger patterns)
	triggerCandidates := matchTriggers(cat, filePaths)

	return mergeCandidates(graphCandidates, triggerCandidates)
}

func extractPaths(diffs []diff.FileDiff) []string {
	var paths []string
	for _, d := range diffs {
		if d.Path != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths
}

func allCheckTypesAsCandidates(cat *catalog.Catalog) []NodeWithSignal {
	var candidates []NodeWithSignal
	for _, ct := range cat.CheckTypes {
		candidates = append(candidates, NodeWithSignal{
			NodeID: "check:" + ct.ID,
			Signal: 1.0,
		})
	}
	return candidates
}

func graphWalkCandidates(g *graph.Graph, changedIDs []string) []NodeWithSignal {
	reachable := graph.ReachableChecks(g, changedIDs, 0.01)
	candidates := make([]NodeWithSignal, len(reachable))
	for i, r := range reachable {
		candidates[i] = NodeWithSignal{
			NodeID: r.CheckID,
			Signal: r.Signal,
		}
	}
	return candidates
}

func matchTriggers(cat *catalog.Catalog, filePaths []string) []NodeWithSignal {
	var candidates []NodeWithSignal
	for _, ct := range cat.CheckTypes {
		if len(ct.Triggers) > 0 && ct.MatchesTriggers(filePaths) {
			candidates = append(candidates, NodeWithSignal{
				NodeID: "check:" + ct.ID,
				Signal: 1.0, // trigger match = direct relevance
			})
		}
	}
	return candidates
}

func mergeCandidates(a, b []NodeWithSignal) []NodeWithSignal {
	best := make(map[string]float64)
	for _, c := range a {
		if c.Signal > best[c.NodeID] {
			best[c.NodeID] = c.Signal
		}
	}
	for _, c := range b {
		if c.Signal > best[c.NodeID] {
			best[c.NodeID] = c.Signal
		}
	}

	result := make([]NodeWithSignal, 0, len(best))
	for id, signal := range best {
		result = append(result, NodeWithSignal{NodeID: id, Signal: signal})
	}
	return result
}
