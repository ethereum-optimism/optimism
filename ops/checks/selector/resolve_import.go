package selector

import (
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// importScopeCandidates is the fallback for scopeable checks when no
// coverage edges point at the changed files. Reverse-BFS walks
// *incoming* import edges from the changed nodes to find everything
// that transitively imports them — the set of files whose compilation
// or behavior could break when the changed file changes.
//
// This catches cases coverage misses:
//   - Test helper changes (test/helpers/Foo.sol has no tested_by edge
//     pointing at it, so coverage finds nothing; but test files that
//     import it must re-run because their compilation can break).
//   - Source changes in modules where coverage hasn't been collected.
//
// Edge direction: the Solidity adapter writes `importer → imported`,
// so to find consumers of a changed node we walk `EdgesTo(cur)` and
// propagate from edge.From (the importer).
//
// Only consumers that are themselves test files, and that have a
// tested_by edge to this check, become scope candidates; the scope is
// the specific test file path (e.g. `./test/L1/X.t.sol`), not a
// directory glob.
func importScopeCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string) []Candidate {
	checkNodeID := "check:" + ct.ID

	consumers := transitiveConsumers(g, changedIDs, 0.01)
	if len(consumers) == 0 {
		return nil
	}

	type emit struct {
		nodeID string
		signal float64
	}
	var hits []emit
	for nodeID, signal := range consumers {
		if !isCandidateTestNode(nodeID, ct) {
			continue
		}
		if !hasTestedByEdge(g, nodeID, checkNodeID) {
			continue
		}
		hits = append(hits, emit{nodeID, signal})
	}
	if len(hits) == 0 {
		return nil
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].signal > hits[j].signal })

	seen := make(map[string]bool)
	out := make([]Candidate, 0, len(hits))
	for _, h := range hits {
		scope := scopeForCandidate(h.nodeID, ct)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, Candidate{
			CheckID: ct.ID,
			Scope:   scope,
			Signal:  h.signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeImports,
				Contribution: h.signal,
				Raw: map[string]any{
					"via_node": h.nodeID,
					"reason":   "transitive_importer",
				},
			}},
		})
	}
	return out
}

// transitiveConsumers performs reverse-BFS on incoming `imports` edges
// from each changed node. Returns node ID → accumulated signal for
// every node that (transitively) imports a changed node. Signal
// attenuates by edge.Strength * edge.Confidence per hop.
func transitiveConsumers(g *graph.Graph, changedIDs []string, minSignal float64) map[string]float64 {
	best := make(map[string]float64)
	type item struct {
		id     string
		signal float64
	}
	queue := make([]item, 0, len(changedIDs))
	for _, id := range changedIDs {
		if g.GetNode(id) == nil {
			continue
		}
		best[id] = 1.0
		queue = append(queue, item{id, 1.0})
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.signal < best[cur.id] {
			continue
		}
		for _, edge := range g.EdgesTo(cur.id) {
			if edge.Kind != graph.EdgeImports {
				continue
			}
			s := cur.signal * edge.Strength * edge.Confidence
			if s < minSignal {
				continue
			}
			importer := edge.From
			if existing, ok := best[importer]; !ok || s > existing {
				best[importer] = s
				queue = append(queue, item{importer, s})
			}
		}
	}
	return best
}

// importBinaryCandidates emits a single Candidate for a non-scopeable
// check if it's reachable via the import graph.
func importBinaryCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string) []Candidate {
	checkNodeID := "check:" + ct.ID
	for _, r := range graph.ReachableChecks(g, changedIDs, 0.01) {
		if r.CheckID != checkNodeID {
			continue
		}
		return []Candidate{{
			CheckID: ct.ID,
			Signal:  r.Signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeTestedBy,
				Contribution: r.Signal,
				Raw: map[string]any{
					"path": r.Path,
				},
			}},
		}}
	}
	return nil
}

// mergeBinaryCandidates combines at most one import-based and one
// correlation-based candidate for the same check, taking max signal
// and unioning provenance so both sources show up in `explain`.
func mergeBinaryCandidates(imports, correlation []Candidate) []Candidate {
	switch {
	case len(imports) == 0 && len(correlation) == 0:
		return nil
	case len(imports) == 0:
		return correlation
	case len(correlation) == 0:
		return imports
	}
	primary, secondary := imports[0], correlation[0]
	if secondary.Signal > primary.Signal {
		primary, secondary = secondary, primary
	}
	primary.Provenance = append(primary.Provenance, secondary.Provenance...)
	return []Candidate{primary}
}
