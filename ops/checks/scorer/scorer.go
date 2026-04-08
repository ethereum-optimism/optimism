package scorer

import "github.com/ethereum-optimism/optimism/ops/checks/graph"

// Score represents the scoring output for a single check.
type Score struct {
	CheckID     string
	PFail       float64 // P(fail | diff), 0.0–1.0
	RunCost     float64 // cost of running (seconds)
	Explanation string
}

// Scorer computes P(fail | diff) for each reachable check.
type Scorer interface {
	Score(g *graph.Graph, changedFiles []string, reachable []graph.CheckSignal) ([]Score, error)
}
