package selector

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/scorer"
)

func emptyGraph() *graph.Graph {
	return graph.NewGraph()
}

func TestSelect_HighPFailLowCost(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:lint", Kind: graph.KindCheck, Name: "lint"})

	// P(fail)=0.8, RunCost=5s, commit miss_cost=300s
	// skip_cost = 0.8 × 300 = 240 > 5 → SELECTED
	scores := []scorer.Score{
		{CheckID: "check:lint", PFail: 0.8, RunCost: 5, Explanation: "lint check"},
	}

	result := Select(scores, StageOnCommit, g)

	if len(result.Selections) != 1 {
		t.Fatalf("expected 1 selection, got %d", len(result.Selections))
	}
	if result.Selections[0].CheckID != "check:lint" {
		t.Error("expected lint to be selected")
	}
}

func TestSelect_LowPFailHighCost(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:e2e", Kind: graph.KindCheck, Name: "e2e"})

	// P(fail)=0.02, RunCost=3600s, commit miss_cost=300s
	// skip_cost = 0.02 × 300 = 6 < 3600 → SKIPPED
	scores := []scorer.Score{
		{CheckID: "check:e2e", PFail: 0.02, RunCost: 3600, Explanation: "e2e test"},
	}

	result := Select(scores, StageOnCommit, g)

	if len(result.Selections) != 0 {
		t.Errorf("expected 0 selections at commit stage, got %d", len(result.Selections))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
}

func TestSelect_SameCheckDifferentStages(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:medium", Kind: graph.KindCheck, Name: "medium"})

	// P(fail)=0.1, RunCost=600s
	scores := []scorer.Score{
		{CheckID: "check:medium", PFail: 0.1, RunCost: 600, Explanation: "medium check"},
	}

	// At commit: skip_cost = 0.1 × 300 = 30 < 600 → SKIP
	commitResult := Select(scores, StageOnCommit, g)
	if len(commitResult.Selections) != 0 {
		t.Error("expected medium check to be skipped at commit stage")
	}

	// At merge_queue: skip_cost = 0.1 × 7200 = 720 > 600 → SELECT
	mqResult := Select(scores, StageMergeQueue, g)
	if len(mqResult.Selections) != 1 {
		t.Error("expected medium check to be selected at merge_queue stage")
	}

	// At develop: skip_cost = 0.1 × 86400 = 8640 > 600 → SELECT
	devResult := Select(scores, StageDevelop, g)
	if len(devResult.Selections) != 1 {
		t.Error("expected medium check to be selected at develop stage")
	}
}

func TestSelect_DevelopRunsEverything(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:a", Kind: graph.KindCheck, Name: "a"})
	_ = g.AddNode(&graph.Node{ID: "check:b", Kind: graph.KindCheck, Name: "b"})
	_ = g.AddNode(&graph.Node{ID: "check:c", Kind: graph.KindCheck, Name: "c"})

	// At develop stage (miss_cost=86400s), even P(fail)=0.01 on a 600s check:
	// skip_cost = 0.01 × 86400 = 864 > 600 → SELECTED
	scores := []scorer.Score{
		{CheckID: "check:a", PFail: 0.01, RunCost: 600},
		{CheckID: "check:b", PFail: 0.05, RunCost: 900},
		{CheckID: "check:c", PFail: 0.001, RunCost: 60},
	}

	result := Select(scores, StageDevelop, g)
	if len(result.Selections) != 3 {
		t.Errorf("expected all 3 checks at develop stage, got %d selected, %d skipped",
			len(result.Selections), len(result.Skipped))
		for _, s := range result.Skipped {
			t.Logf("  skipped: %s (skip_cost=%.2f, run_cost=%.0f)", s.CheckID, s.SkipCost, s.RunCost)
		}
	}
}

func TestSelect_PrerequisiteInclusion(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:build", Kind: graph.KindCheck, Name: "build",
		Properties: map[string]any{"avg_duration": 120.0}})
	_ = g.AddNode(&graph.Node{ID: "check:test", Kind: graph.KindCheck, Name: "test"})
	_ = g.AddEdge(&graph.Edge{
		From: "check:build", To: "check:test", Kind: graph.EdgePrerequisite,
		Source: graph.SourceManual, Confidence: 1, Strength: 1,
	})

	// P(fail)=0.5, RunCost=600s, merge_queue miss_cost=7200s
	// skip_cost = 0.5 × 7200 = 3600 > 600 → SELECTED, and build is prereq
	scores := []scorer.Score{
		{CheckID: "check:test", PFail: 0.5, RunCost: 600, Explanation: "test"},
	}

	result := Select(scores, StageMergeQueue, g)

	foundBuild := false
	foundTest := false
	for _, s := range result.Selections {
		if s.CheckID == "check:build" {
			foundBuild = true
		}
		if s.CheckID == "check:test" {
			foundTest = true
		}
	}
	if !foundTest {
		t.Error("expected test to be selected")
	}
	if !foundBuild {
		t.Error("expected build prerequisite to be auto-included")
	}
}

func TestSelect_ResultOrdering(t *testing.T) {
	g := emptyGraph()
	_ = g.AddNode(&graph.Node{ID: "check:low", Kind: graph.KindCheck, Name: "low"})
	_ = g.AddNode(&graph.Node{ID: "check:high", Kind: graph.KindCheck, Name: "high"})

	scores := []scorer.Score{
		{CheckID: "check:low", PFail: 0.1, RunCost: 5},
		{CheckID: "check:high", PFail: 0.9, RunCost: 5},
	}

	result := Select(scores, StageMergeQueue, g)

	if len(result.Selections) < 2 {
		t.Fatalf("expected 2 selections, got %d", len(result.Selections))
	}
	if result.Selections[0].CheckID != "check:high" {
		t.Error("expected highest skip_cost first")
	}
}
