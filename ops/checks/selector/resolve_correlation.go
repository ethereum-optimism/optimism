package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// correlationCandidates emits a Candidate whose signal comes from
// EdgeObservedCorrelation edges written by CI-history ingestion. For
// each changed node, the strongest outgoing correlation edge to this
// check's node becomes the candidate's signal; freshness is applied
// like it is for coverage.
func correlationCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string, fresh freshness.Checker) []Candidate {
	checkNodeID := "check:" + ct.ID
	var bestSignal float64
	var bestProps map[string]any
	var bestFreshness float64

	for _, nodeID := range changedIDs {
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind != graph.EdgeObservedCorrelation || edge.To != checkNodeID {
				continue
			}
			fr := fresh.Assess(edge)
			signal := edge.Strength * edge.Confidence * fr
			if signal > bestSignal {
				bestSignal = signal
				bestProps = edge.Properties
				bestFreshness = fr
			}
		}
	}

	if bestSignal == 0 {
		return nil
	}

	raw := map[string]any{}
	if bestProps != nil {
		if v, ok := bestProps["observations"]; ok {
			raw["observations"] = v
		}
		if v, ok := bestProps["failures"]; ok {
			raw["failures"] = v
		}
		if v, ok := bestProps["precision"]; ok {
			raw["precision"] = v
		}
	}
	if bestFreshness < 1.0 {
		raw["freshness"] = bestFreshness
	}

	return []Candidate{{
		CheckID: ct.ID,
		Signal:  bestSignal,
		Provenance: []SignalContribution{{
			Source:       graph.SourceCIHistory,
			EdgeKind:     graph.EdgeObservedCorrelation,
			Contribution: bestSignal,
			Raw:          raw,
		}},
	}}
}
