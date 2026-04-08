package scorer

import (
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func makeGraph(checks []struct {
	id        string
	kind      string
	flakiness float64
	duration  float64
}) *graph.Graph {
	g := graph.NewGraph()
	for _, c := range checks {
		props := map[string]any{
			"kind":         c.kind,
			"avg_duration": c.duration,
		}
		if c.flakiness > 0 {
			props["flakiness_score"] = c.flakiness
		}
		_ = g.AddNode(&graph.Node{
			ID: c.id, Kind: graph.KindCheck, Name: c.id, Properties: props,
		})
	}
	return g
}

func TestSimpleScorer_DirectDependency(t *testing.T) {
	g := makeGraph([]struct {
		id        string
		kind      string
		flakiness float64
		duration  float64
	}{
		{"check:test-a", "test", 0, 600},
	})

	s := NewSimple()
	scores, err := s.Score(g, nil, []graph.CheckSignal{
		{CheckID: "check:test-a", Signal: 1.0},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}
	// P(fail) = 1.0 × 0.3 × 1.0 = 0.3
	if math.Abs(scores[0].PFail-0.3) > 0.001 {
		t.Errorf("expected P(fail)=0.3, got %f", scores[0].PFail)
	}
}

func TestSimpleScorer_LintPrior(t *testing.T) {
	g := makeGraph([]struct {
		id        string
		kind      string
		flakiness float64
		duration  float64
	}{
		{"check:lint", "lint", 0, 300},
		{"check:test", "test", 0, 600},
	})

	s := NewSimple()
	scores, _ := s.Score(g, nil, []graph.CheckSignal{
		{CheckID: "check:lint", Signal: 1.0},
		{CheckID: "check:test", Signal: 1.0},
	})

	var lintScore, testScore Score
	for _, sc := range scores {
		if sc.CheckID == "check:lint" {
			lintScore = sc
		}
		if sc.CheckID == "check:test" {
			testScore = sc
		}
	}

	if lintScore.PFail <= testScore.PFail {
		t.Errorf("lint P(fail)=%f should be > test P(fail)=%f", lintScore.PFail, testScore.PFail)
	}
}

func TestSimpleScorer_FlakyDiscount(t *testing.T) {
	g := makeGraph([]struct {
		id        string
		kind      string
		flakiness float64
		duration  float64
	}{
		{"check:stable", "test", 0, 600},
		{"check:flaky", "test", 0.5, 600},
	})

	s := NewSimple()
	scores, _ := s.Score(g, nil, []graph.CheckSignal{
		{CheckID: "check:stable", Signal: 1.0},
		{CheckID: "check:flaky", Signal: 1.0},
	})

	var stable, flaky Score
	for _, sc := range scores {
		if sc.CheckID == "check:stable" {
			stable = sc
		}
		if sc.CheckID == "check:flaky" {
			flaky = sc
		}
	}

	// Flaky should be half of stable
	expected := stable.PFail * 0.5
	if math.Abs(flaky.PFail-expected) > 0.001 {
		t.Errorf("flaky P(fail)=%f, expected %f (half of stable %f)", flaky.PFail, expected, stable.PFail)
	}
}

func TestSimpleScorer_TransitiveDependency(t *testing.T) {
	g := makeGraph([]struct {
		id        string
		kind      string
		flakiness float64
		duration  float64
	}{
		{"check:test-a", "test", 0, 600},
	})

	s := NewSimple()
	scores, _ := s.Score(g, nil, []graph.CheckSignal{
		{CheckID: "check:test-a", Signal: 0.5}, // attenuated signal
	})

	// P(fail) = 0.5 × 0.3 = 0.15
	if math.Abs(scores[0].PFail-0.15) > 0.001 {
		t.Errorf("expected P(fail)=0.15, got %f", scores[0].PFail)
	}
}
