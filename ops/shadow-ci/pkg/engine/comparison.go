package engine

import (
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// ComparisonResult holds the comparison between shadow and main CI.
type ComparisonResult struct {
	// Equivalence.
	MainCIRealFailures int     `json:"main_ci_real_failures"`
	ShadowCICaught     int     `json:"shadow_ci_caught"`
	FalseNegatives     int     `json:"false_negatives"`
	CatchRate          float64 `json:"catch_rate"`

	// Performance.
	MainCIWallTime         time.Duration `json:"main_ci_wall_time"`
	ShadowCIWallTime       time.Duration `json:"shadow_ci_wall_time"`
	Speedup                float64       `json:"speedup"`
	MainCIComputeMinutes   float64       `json:"main_ci_compute_minutes"`
	ShadowCIComputeMinutes float64       `json:"shadow_ci_compute_minutes"`
	ComputeReduction       float64       `json:"compute_reduction"`

	// Flakes.
	MainCIFlakeFailures int      `json:"main_ci_flake_failures"`
	ShadowCIFlakes      int      `json:"shadow_ci_flakes"`
	UniqueFingerprints  []string `json:"unique_fingerprints"`

	// Efficiency by language.
	ByLanguage map[string]LanguageEfficiency `json:"by_language"`

	// Detail on false negatives.
	FalseNegativeDetails []model.FalseNegativeDetail `json:"false_negative_details"`
}

// LanguageEfficiency holds per-language efficiency metrics.
type LanguageEfficiency struct {
	SkipRate       float64 `json:"skip_rate"`
	SelectedTests  int     `json:"selected_tests"`
	TotalTests     int     `json:"total_tests"`
	WallTime       float64 `json:"wall_time_seconds"`
	Configurations int     `json:"configurations"`
}

// ComparisonEngine compares shadow CI results against main CI results.
type ComparisonEngine struct {
	emitter *events.Emitter
}

// NewComparisonEngine creates a new ComparisonEngine.
func NewComparisonEngine(emitter *events.Emitter) *ComparisonEngine {
	return &ComparisonEngine{emitter: emitter}
}

// Compare compares shadow results against main CI results.
func (ce *ComparisonEngine) Compare(shadowResults []model.TestResult, mainResults []model.TestResult) *ComparisonResult {
	result := &ComparisonResult{
		ByLanguage: make(map[string]LanguageEfficiency),
	}

	// Index shadow results by test identity.
	shadowIndex := make(map[string]model.TestResult)
	for _, sr := range shadowResults {
		shadowIndex[sr.Test.Key()] = sr
	}

	// Collect unique flake fingerprints.
	fingerprintSet := make(map[string]bool)
	for _, sr := range shadowResults {
		if sr.Classification == model.Flake && sr.Fingerprint != "" {
			fingerprintSet[sr.Fingerprint] = true
		}
	}

	result.ShadowCIFlakes = len(fingerprintSet)
	for fp := range fingerprintSet {
		result.UniqueFingerprints = append(result.UniqueFingerprints, fp)
	}

	// Find real failures in main CI and check shadow coverage.
	for _, mr := range mainResults {
		if mr.Status != model.StatusFail {
			continue
		}

		result.MainCIRealFailures++

		sr, inShadow := shadowIndex[mr.Test.Key()]

		if !inShadow {
			// FALSE NEGATIVE — shadow CI didn't run this test.
			detail := model.FalseNegativeDetail{
				Test:                mr.Test,
				Language:            mr.Language,
				FailedInMainCI:      true,
				InShadowAffectedSet: false,
				MissedBecause:       "test was not in the affected set — dependency graph gap",
			}
			result.FalseNegativeDetails = append(result.FalseNegativeDetails, detail)
			result.FalseNegatives++

			if ce.emitter != nil {
				ce.emitter.Emit(model.EventFalseNegative, detail)
				ce.emitter.Emit(model.EventGraphGap, map[string]any{
					"test":     mr.Test,
					"language": mr.Language,
					"reason":   "test not in affected set",
				})
			}
		} else if sr.Classification == model.Flake {
			// Main CI saw failure, shadow classified as flake.
			// This means the main CI failure was likely a flake too.
			result.MainCIFlakeFailures++
		} else if sr.Classification == model.RealFailure || sr.Status == model.StatusFail {
			result.ShadowCICaught++
			if ce.emitter != nil {
				ce.emitter.Emit(model.EventMatchConfirmed, map[string]any{
					"test":     mr.Test,
					"language": mr.Language,
				})
			}
		}
	}

	// Compute catch rate.
	if result.MainCIRealFailures > 0 {
		result.CatchRate = float64(result.ShadowCICaught) / float64(result.MainCIRealFailures)
	} else {
		result.CatchRate = 1.0 // no failures to miss
	}

	// Compute performance comparison.
	result.MainCIWallTime = computeWallTime(mainResults)
	result.ShadowCIWallTime = computeWallTime(shadowResults)
	if result.MainCIWallTime > 0 {
		result.Speedup = float64(result.MainCIWallTime) / float64(result.ShadowCIWallTime)
	}

	result.MainCIComputeMinutes = computeMinutes(mainResults)
	result.ShadowCIComputeMinutes = computeMinutes(shadowResults)
	if result.MainCIComputeMinutes > 0 {
		result.ComputeReduction = 1.0 - (result.ShadowCIComputeMinutes / result.MainCIComputeMinutes)
	}

	if ce.emitter != nil {
		ce.emitter.Emit(model.EventComparisonComplete, result)
	}

	return result
}

func computeWallTime(results []model.TestResult) time.Duration {
	var max time.Duration
	for _, r := range results {
		if r.Duration > max {
			max = r.Duration
		}
	}
	return max
}

func computeMinutes(results []model.TestResult) float64 {
	var total time.Duration
	for _, r := range results {
		total += r.Duration
	}
	return total.Minutes()
}
