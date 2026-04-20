package selector

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/internal/glob"
)

// selectViaDataflow walks the graph's consumes/produces edges to find
// every check whose inputs have been invalidated by the diff. This is
// the single selection authority post-cutover: it answers "which
// checks run?" The scoping layer (coverage + import walks) answers
// "with what scope, if scopeable?" as pure post-processing.
//
// Seeds come in as node IDs (resolved upstream by the orchestrator).
// For paths that don't map to any graph node (.circleci/**, go.mod,
// mise.toml, etc.), filePaths are matched directly against check
// inputs — the universal_inputs + mise-setup + expandGoModDiffs
// pathways populate those nodes where possible; this fallback covers
// everything else.
//
// Returns a set of check IDs (without the "check:" prefix).
func selectViaDataflow(g *graph.Graph, seedIDs []string, filePaths []string, cat *catalog.Catalog) map[string]bool {
	invalidated := make(map[string]bool, len(seedIDs))
	for _, id := range seedIDs {
		if id != "" {
			invalidated[id] = true
		}
	}

	selected := make(map[string]bool)

	// Pre-seed selected checks by matching filePaths against
	// catalog-declared inputs. Handles files that aren't in the source
	// graph (go.mod, mise.toml, .circleci/...).
	if cat != nil {
		for i := range cat.CheckTypes {
			ct := &cat.CheckTypes[i]
			patterns := append([]string{}, ct.Inputs...)
			patterns = append(patterns, cat.UniversalInputs...)
			matched := false
			for _, in := range patterns {
				if strings.HasPrefix(in, "artifact:") {
					continue
				}
				for _, path := range filePaths {
					if glob.Match(in, path) {
						selected[ct.ID] = true
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
			// Seed this check's outputs so downstream consumers propagate.
			for _, e := range g.EdgesFrom("check:" + ct.ID) {
				if e.Kind == graph.EdgeProduces && !invalidated[e.To] {
					invalidated[e.To] = true
				}
			}
		}
	}

	// Fixpoint walk: each invalidated node may be consumed by checks;
	// each such check is selected and its outputs become invalidated,
	// propagating further.
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

	return selected
}
