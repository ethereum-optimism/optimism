package scorer

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// SimpleScorer computes P(fail) from graph distance and check type priors.
//
// P(fail) = graphSignal × checkTypePrior × (1 - flakiness)
type SimpleScorer struct {
	priors map[string]float64
}

// NewSimple creates a SimpleScorer with default priors.
func NewSimple() *SimpleScorer {
	return &SimpleScorer{
		priors: map[string]float64{
			"lint":  0.8,
			"build": 0.5,
			"test":  0.3,
			"check": 0.4,
		},
	}
}

// Score computes P(fail | diff) for each reachable check.
func (s *SimpleScorer) Score(g *graph.Graph, _ []string, reachable []graph.CheckSignal) ([]Score, error) {
	var scores []Score

	for _, r := range reachable {
		node := g.GetNode(r.CheckID)
		if node == nil {
			continue
		}

		// Get check kind for prior
		kind, _ := node.Properties["kind"].(string)
		prior := s.priors[kind]
		if prior == 0 {
			prior = 0.3 // default
		}

		// Get flakiness score
		flakiness, _ := node.Properties["flakiness_score"].(float64)

		// Compute P(fail)
		pFail := r.Signal * prior * (1 - flakiness)
		if pFail > 1.0 {
			pFail = 1.0
		}

		// Get run cost
		avgDuration, _ := node.Properties["avg_duration"].(float64)
		// JSON numbers are float64; YAML loaded via catalog uses int, which becomes float64 via JSON
		if avgDuration == 0 {
			if d, ok := node.Properties["avg_duration"].(int); ok {
				avgDuration = float64(d)
			}
		}

		explanation := fmt.Sprintf("signal=%.2f × prior(%.1f) × (1-flake(%.2f)) = %.3f",
			r.Signal, prior, flakiness, pFail)

		scores = append(scores, Score{
			CheckID:     r.CheckID,
			PFail:       pFail,
			RunCost:     avgDuration,
			Explanation: explanation,
		})
	}

	return scores, nil
}
