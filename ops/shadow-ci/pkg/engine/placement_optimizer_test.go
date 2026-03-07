package engine

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestOptimizer_HighFlakeRate_DemoteFromMergeQueue(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"acceptance_tests": {Stage: model.StageMergeQueue, Source: "static"},
		},
	}

	stats := map[string]*CategoryStats{
		"acceptance_tests": {TotalRuns: 100, FlakeRate: 0.05, MeanDuration: 10 * time.Minute},
	}

	recs := opt.Optimize(placement, nil, stats)
	assert.Len(t, recs, 1)
	assert.Equal(t, model.StagePostMerge, recs[0].ProposedStage)
	assert.Contains(t, recs[0].Reason, "flake rate")
}

func TestOptimizer_RespectsConstraints(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Constraints: []model.PlacementConstraint{
			{Category: "go_lint", PinnedStage: model.StagePR, Reason: "must run pre-merge"},
		},
	}

	stats := map[string]*CategoryStats{
		"go_lint": {TotalRuns: 100, FlakeRate: 0.5, MeanDuration: 60 * time.Minute},
	}

	recs := opt.Optimize(placement, nil, stats)
	assert.Empty(t, recs, "pinned categories should never get recommendations")
}

func TestOptimizer_FalseNegativePromotion(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"heavy_tests": {Stage: model.StagePostMerge, Source: "optimizer"},
		},
	}

	stats := map[string]*CategoryStats{
		"heavy_tests": {TotalRuns: 100, FalseNegativeCount: 2},
	}

	recs := opt.Optimize(placement, nil, stats)
	assert.Len(t, recs, 1)
	assert.Equal(t, model.StageMergeQueue, recs[0].ProposedStage)
	assert.Contains(t, recs[0].Reason, "false negative")
}

func TestOptimizer_NightlyPromotion(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"slow_tests": {Stage: model.StagePR, Source: "static"},
		},
	}

	stats := map[string]*CategoryStats{
		"slow_tests": {TotalRuns: 100, RealFailureRate: 0, MeanDuration: 45 * time.Minute},
	}

	recs := opt.Optimize(placement, nil, stats)
	assert.Len(t, recs, 1)
	assert.Equal(t, model.StageMergeQueue, recs[0].ProposedStage)
	assert.Contains(t, recs[0].Reason, "never fails")
}

func TestOptimizer_ApplyRecommendations(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments:  map[string]model.CategoryPlacement{},
	}

	recs := []PlacementRecommendation{
		{
			Category:      "slow_tests",
			CurrentStage:  model.StagePR,
			ProposedStage: model.StageMergeQueue,
			Reason:        "test",
			Confidence:    0.9,
		},
	}

	opt.ApplyRecommendations(&placement, recs, nil)
	assert.Equal(t, model.StageMergeQueue, placement.Assignments["slow_tests"].Stage)
	assert.Equal(t, "optimizer", placement.Assignments["slow_tests"].Source)
}

func TestOptimizer_CorrelationDeferral(t *testing.T) {
	config := DefaultPlacementOptimizerConfig()
	opt := NewPlacementOptimizer(nil, config)

	placement := model.PlacementConfig{
		DefaultStage: model.StagePR,
		Assignments: map[string]model.CategoryPlacement{
			"slow_test": {Stage: model.StagePR, Source: "static"},
		},
	}

	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{
				TestA:      "fast_test",
				TestB:      "slow_test",
				CoFailRate: 0.99,
				SampleSize: 50,
				SpeedRatio: 5.0,
				Confidence: 0.95,
			},
		},
	}

	stats := map[string]*CategoryStats{
		"slow_test": {TotalRuns: 100},
	}

	recs := opt.Optimize(placement, correlations, stats)
	assert.Len(t, recs, 1)
	assert.Equal(t, model.StageMergeQueue, recs[0].ProposedStage)
	assert.Contains(t, recs[0].Reason, "correlated")
}
