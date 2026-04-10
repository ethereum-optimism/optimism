package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Optimizer computes an execution plan from candidate nodes.
// Phase 1 (FindCandidates) produces the candidates.
// Phase 2 (this interface) reads candidate node properties and
// inter-candidate edges from the graph to produce the plan.
type Optimizer interface {
	Optimize(
		g *graph.Graph,
		candidates []NodeWithSignal,
		stage Stage,
		cat *catalog.Catalog,
	) (*Result, error)
}
