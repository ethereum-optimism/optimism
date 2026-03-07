package engine

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// PlacementOptimizer computes optimal stage assignments for test categories.
type PlacementOptimizer struct {
	store  events.Store
	config PlacementOptimizerConfig
}

// PlacementOptimizerConfig controls optimizer behavior.
type PlacementOptimizerConfig struct {
	MinPipelines           int     // minimum pipelines before making changes (default: 50)
	CoFailThreshold        float64 // co-failure threshold for deferral (default: 0.95)
	SpeedRatioThreshold    float64 // speed ratio threshold for deferral (default: 3.0)
	MergeQueueMaxFlakeRate float64 // max flake rate for merge queue (default: 0.01)
}

// DefaultPlacementOptimizerConfig returns sensible defaults.
func DefaultPlacementOptimizerConfig() PlacementOptimizerConfig {
	return PlacementOptimizerConfig{
		MinPipelines:           50,
		CoFailThreshold:        0.95,
		SpeedRatioThreshold:    3.0,
		MergeQueueMaxFlakeRate: 0.01,
	}
}

// PlacementRecommendation is a proposed change to a category's stage.
type PlacementRecommendation struct {
	Category      string      `json:"category"`
	CurrentStage  model.Stage `json:"current_stage"`
	ProposedStage model.Stage `json:"proposed_stage"`
	Reason        string      `json:"reason"`
	Confidence    float64     `json:"confidence"`
}

// NewPlacementOptimizer creates a new PlacementOptimizer.
func NewPlacementOptimizer(store events.Store, config PlacementOptimizerConfig) *PlacementOptimizer {
	return &PlacementOptimizer{store: store, config: config}
}

// Optimize analyzes historical data and produces placement recommendations.
func (po *PlacementOptimizer) Optimize(
	currentPlacement model.PlacementConfig,
	correlations *CorrelationMatrix,
	categoryStats map[string]*CategoryStats,
) []PlacementRecommendation {
	var recommendations []PlacementRecommendation

	for category, stats := range categoryStats {
		current := currentPlacement.GetCategoryStage(category)

		// Check if pinned -- never modify pinned categories.
		if po.isPinned(currentPlacement, category) {
			continue
		}

		// Rule 1: High flake rate demotes from merge queue.
		if current == model.StageMergeQueue && stats.FlakeRate > po.config.MergeQueueMaxFlakeRate {
			recommendations = append(recommendations, PlacementRecommendation{
				Category:      category,
				CurrentStage:  current,
				ProposedStage: model.StagePostMerge,
				Reason:        fmt.Sprintf("flake rate %.1f%% exceeds merge queue threshold %.1f%%", stats.FlakeRate*100, po.config.MergeQueueMaxFlakeRate*100),
				Confidence:    stats.FlakeRate,
			})
			continue
		}

		// Rule 2: Correlation deferral -- if a faster correlated test exists.
		if correlations != nil {
			if rec := po.checkCorrelationDeferral(category, current, correlations); rec != nil {
				recommendations = append(recommendations, *rec)
				continue
			}
		}

		// Rule 3: Slow tests that never fail -> move to later stage.
		if stats.MeanDuration > 30*time.Minute && stats.RealFailureRate == 0 && stats.TotalRuns >= po.config.MinPipelines {
			nextStage := po.nextStage(current)
			if nextStage != current {
				recommendations = append(recommendations, PlacementRecommendation{
					Category:      category,
					CurrentStage:  current,
					ProposedStage: nextStage,
					Reason:        fmt.Sprintf("never fails, mean duration %s -- defer to %s", stats.MeanDuration.Round(time.Minute), nextStage),
					Confidence:    0.9,
				})
			}
		}

		// Rule 4: False negative feedback -- promote back.
		if stats.FalseNegativeCount > 0 {
			prevStage := po.prevStage(current)
			if prevStage != current {
				recommendations = append(recommendations, PlacementRecommendation{
					Category:      category,
					CurrentStage:  current,
					ProposedStage: prevStage,
					Reason:        fmt.Sprintf("%d false negatives detected -- promoting back to %s", stats.FalseNegativeCount, prevStage),
					Confidence:    0.95,
				})
			}
		}
	}

	return recommendations
}

// ApplyRecommendations applies recommendations to a placement config, respecting constraints.
func (po *PlacementOptimizer) ApplyRecommendations(
	placement *model.PlacementConfig,
	recommendations []PlacementRecommendation,
	emitter *events.Emitter,
) {
	for _, rec := range recommendations {
		if po.isPinned(*placement, rec.Category) {
			continue
		}
		if po.violatesMinStage(*placement, rec.Category, rec.ProposedStage) {
			continue
		}

		if placement.Assignments == nil {
			placement.Assignments = make(map[string]model.CategoryPlacement)
		}
		placement.Assignments[rec.Category] = model.CategoryPlacement{
			Stage:     rec.ProposedStage,
			Source:    "optimizer",
			Reason:    rec.Reason,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if emitter != nil {
			emitter.Emit(model.EventPlacementChanged, rec)
		}
	}
}

// CategoryStats holds aggregated stats for a test category.
type CategoryStats struct {
	TotalRuns          int
	FlakeRate          float64
	RealFailureRate    float64
	MeanDuration       time.Duration
	FalseNegativeCount int
}

func (po *PlacementOptimizer) isPinned(placement model.PlacementConfig, category string) bool {
	for _, c := range placement.Constraints {
		if c.Category == category && c.PinnedStage != "" {
			return true
		}
	}
	return false
}

func (po *PlacementOptimizer) violatesMinStage(placement model.PlacementConfig, category string, proposed model.Stage) bool {
	for _, c := range placement.Constraints {
		if c.Category == category && c.MinStage != "" {
			// MinStage means must run at this stage or earlier.
			return model.StageIndex(proposed) > model.StageIndex(c.MinStage)
		}
	}
	return false
}

func (po *PlacementOptimizer) checkCorrelationDeferral(category string, current model.Stage, correlations *CorrelationMatrix) *PlacementRecommendation {
	for _, c := range correlations.Correlations {
		if c.TestA == category || c.TestB == category {
			if c.CoFailRate >= po.config.CoFailThreshold && c.SpeedRatio >= po.config.SpeedRatioThreshold {
				// The slower test can be deferred.
				slower := c.TestB
				if c.SpeedRatio < 1 {
					slower = c.TestA
				}
				if slower != category {
					continue
				}
				nextStage := po.nextStage(current)
				if nextStage == current {
					continue
				}
				return &PlacementRecommendation{
					Category:      category,
					CurrentStage:  current,
					ProposedStage: nextStage,
					Reason:        fmt.Sprintf("%.0f%% correlated with faster test, speed ratio %.1f×", c.CoFailRate*100, c.SpeedRatio),
					Confidence:    c.Confidence,
				}
			}
		}
	}
	return nil
}

func (po *PlacementOptimizer) nextStage(s model.Stage) model.Stage {
	switch s {
	case model.StagePR:
		return model.StageMergeQueue
	case model.StageMergeQueue:
		return model.StagePostMerge
	case model.StagePostMerge:
		return model.StageNightly
	default:
		return s
	}
}

func (po *PlacementOptimizer) prevStage(s model.Stage) model.Stage {
	switch s {
	case model.StageNightly:
		return model.StagePostMerge
	case model.StagePostMerge:
		return model.StageMergeQueue
	case model.StageMergeQueue:
		return model.StagePR
	default:
		return s
	}
}
