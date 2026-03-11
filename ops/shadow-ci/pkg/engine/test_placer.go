package engine

import (
	"fmt"
	"sort"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// TestPlacer computes per-test stage assignments using the marginal coverage algorithm.
type TestPlacer struct {
	correlationIndex map[string][]Correlation // pre-indexed by test key
	testStats        map[string]*TestStats
	constraints      []model.PlacementConstraint
	config           TestPlacerConfig
	llmOverrides     []PlacementOverride
}

// TestPlacerConfig controls per-test placement behavior.
type TestPlacerConfig struct {
	PRMissRateBudget        float64 // default: 0.05
	MQMissRateBudget        float64 // default: 0.001
	PostMergeMissRateBudget float64 // default: 0.0001
	ShadowMode              bool    // when true, record would_defer but don't actually filter
}

// DefaultTestPlacerConfig returns sensible defaults.
func DefaultTestPlacerConfig() TestPlacerConfig {
	return TestPlacerConfig{
		PRMissRateBudget:        0.05,
		MQMissRateBudget:        0.001,
		PostMergeMissRateBudget: 0.0001,
		ShadowMode:              true, // shadow mode by default
	}
}

// PlacementOverride is an LLM-generated override for a test's placement.
type PlacementOverride struct {
	TestKey    string      `json:"test_key"`
	OverrideTo model.Stage `json:"override_to"`
	Reason     string      `json:"reason"`
	Confidence float64     `json:"confidence"`
}

// NewTestPlacer creates a new TestPlacer.
func NewTestPlacer(
	correlations *CorrelationMatrix,
	testStats map[string]*TestStats,
	constraints []model.PlacementConstraint,
	config TestPlacerConfig,
) *TestPlacer {
	tp := &TestPlacer{
		correlationIndex: make(map[string][]Correlation),
		testStats:        testStats,
		constraints:      constraints,
		config:           config,
	}

	// Pre-index correlations by test key for O(1) lookup.
	if correlations != nil {
		for _, c := range correlations.Correlations {
			tp.correlationIndex[c.TestA] = append(tp.correlationIndex[c.TestA], c)
			tp.correlationIndex[c.TestB] = append(tp.correlationIndex[c.TestB], c)
		}
	}

	return tp
}

// SetLLMOverrides sets LLM-generated placement overrides.
func (tp *TestPlacer) SetLLMOverrides(overrides []PlacementOverride) {
	tp.llmOverrides = overrides
}

// PlaceTests produces per-test placement decisions for a set of test keys at a given stage.
func (tp *TestPlacer) PlaceTests(testKeys []string, currentStage model.Stage) []model.TestPlacement {
	placements := tp.computeStatisticalPlacements(testKeys, currentStage)

	// Apply LLM overrides.
	if len(tp.llmOverrides) > 0 {
		overrideMap := make(map[string]PlacementOverride)
		for _, o := range tp.llmOverrides {
			overrideMap[o.TestKey] = o
		}
		for i := range placements {
			if override, ok := overrideMap[placements[i].TestKey]; ok {
				// LLM cannot override pinned constraints.
				if tp.isPinned(placements[i].TestKey) {
					continue
				}
				placements[i].AssignedStage = override.OverrideTo
				placements[i].Reason = fmt.Sprintf("LLM override: %s", override.Reason)
				placements[i].Confidence = override.Confidence
				// Recalculate deferral status.
				placements[i].WouldDefer = model.StageIndex(override.OverrideTo) > model.StageIndex(currentStage)
				if placements[i].WouldDefer {
					placements[i].DeferTo = override.OverrideTo
				} else {
					placements[i].DeferTo = ""
				}
			}
		}
	}

	return placements
}

// computeStatisticalPlacements applies the marginal coverage algorithm.
func (tp *TestPlacer) computeStatisticalPlacements(testKeys []string, currentStage model.Stage) []model.TestPlacement {
	budget := tp.budgetForStage(currentStage)
	nextStage := nextPlacementStage(currentStage)

	type testCandidate struct {
		key           string
		marginalValue float64 // P(catches unique bug)
		duration      float64 // seconds
		density       float64 // marginalValue / duration
		pinned        bool
	}

	candidates := make([]testCandidate, 0, len(testKeys))
	for _, key := range testKeys {
		tc := testCandidate{
			key:           key,
			marginalValue: 1.0, // default: full value (cold start)
			duration:      10,  // default: 10 seconds
			pinned:        tp.isPinned(key),
		}

		// Use stats if available.
		if stats, ok := tp.testStats[key]; ok && stats.MeanDuration > 0 {
			tc.duration = stats.MeanDuration.Seconds()
		}

		// Compute marginal value from correlations.
		tc.marginalValue = tp.computeMarginalValue(key, testKeys)

		if tc.duration > 0 {
			tc.density = tc.marginalValue / tc.duration
		}

		candidates = append(candidates, tc)
	}

	// Sort by signal density (highest first).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].density > candidates[j].density
	})

	// Greedily assign: high-density tests stay at current stage, others deferred.
	missRateUsed := 0.0
	placements := make([]model.TestPlacement, 0, len(candidates))

	for _, tc := range candidates {
		p := model.TestPlacement{
			TestKey:       tc.key,
			AssignedStage: currentStage,
			Confidence:    1.0 - tc.marginalValue, // higher confidence when marginal value is low
		}

		// Pinned tests always run at current stage.
		if tc.pinned {
			p.Reason = "pinned constraint"
			p.Confidence = 1.0
			placements = append(placements, p)
			continue
		}

		// Check min-stage constraint.
		minStage := tp.getMinStage(tc.key)
		if minStage != "" && model.StageIndex(nextStage) > model.StageIndex(minStage) {
			p.Reason = fmt.Sprintf("min-stage constraint: must run at %s or earlier", minStage)
			placements = append(placements, p)
			continue
		}

		// Check if deferring this test would exceed the miss rate budget.
		// marginalValue represents the probability this test catches a bug no other test catches.
		if missRateUsed+tc.marginalValue <= budget && nextStage != currentStage {
			// Defer this test.
			missRateUsed += tc.marginalValue
			p.WouldDefer = true
			p.DeferTo = nextStage
			p.AssignedStage = nextStage
			p.Reason = fmt.Sprintf("deferred: marginal value %.3f, density %.4f", tc.marginalValue, tc.density)

			// Find correlated faster test.
			if faster := tp.findFasterCorrelate(tc.key, testKeys); faster != "" {
				p.CorrelatedWith = faster
				p.Reason = fmt.Sprintf("deferred: correlated with %s (faster), marginal value %.3f", faster, tc.marginalValue)
			}
		} else {
			p.Reason = fmt.Sprintf("retained: marginal value %.3f, density %.4f", tc.marginalValue, tc.density)
		}

		placements = append(placements, p)
	}

	return placements
}

// computeMarginalValue computes the probability that this test catches a bug
// that no other test in the same set catches.
func (tp *TestPlacer) computeMarginalValue(testKey string, allKeys []string) float64 {
	corrs := tp.correlationIndex[testKey]
	if len(corrs) == 0 {
		return 1.0 // no correlations known → full marginal value
	}

	// Build a set of the other keys for fast lookup.
	keySet := make(map[string]bool, len(allKeys))
	for _, k := range allKeys {
		keySet[k] = true
	}

	// Find the highest co-failure rate with a faster test in the same set.
	bestCoFailRate := 0.0
	for _, c := range corrs {
		var otherKey string
		var speedRatio float64
		if c.TestA == testKey {
			otherKey = c.TestB
			// SpeedRatio = duration_B / duration_A. If > 1, B is slower → A is faster.
			// We want the case where the OTHER test is faster than us.
			// If A == testKey and SpeedRatio > 1, then B is slower, A is faster.
			// We're looking for cases where the OTHER test (not us) is faster.
			speedRatio = c.SpeedRatio
			if speedRatio > 1 {
				continue // the other test is slower, not useful
			}
		} else {
			otherKey = c.TestA
			speedRatio = 1 / c.SpeedRatio
			if speedRatio > 1 {
				continue
			}
		}

		if !keySet[otherKey] {
			continue // the other test isn't in this test set
		}

		if c.CoFailRate > bestCoFailRate {
			bestCoFailRate = c.CoFailRate
		}
	}

	// Marginal value = 1 - bestCoFailRate.
	// If a faster test co-fails with us 99% of the time, our marginal value is 1%.
	return 1.0 - bestCoFailRate
}

// findFasterCorrelate finds the fastest correlated test in the same set.
func (tp *TestPlacer) findFasterCorrelate(testKey string, allKeys []string) string {
	keySet := make(map[string]bool, len(allKeys))
	for _, k := range allKeys {
		keySet[k] = true
	}

	var bestKey string
	bestCoFail := 0.0

	for _, c := range tp.correlationIndex[testKey] {
		var otherKey string
		if c.TestA == testKey {
			otherKey = c.TestB
			if c.SpeedRatio > 1 {
				continue // other is slower
			}
		} else {
			otherKey = c.TestA
			if c.SpeedRatio < 1 {
				continue // other is slower
			}
		}
		if !keySet[otherKey] {
			continue
		}
		if c.CoFailRate > bestCoFail {
			bestCoFail = c.CoFailRate
			bestKey = otherKey
		}
	}

	return bestKey
}

func (tp *TestPlacer) isPinned(testKey string) bool {
	for _, c := range tp.constraints {
		if c.Category == testKey && c.PinnedStage != "" {
			return true
		}
	}
	return false
}

func (tp *TestPlacer) getMinStage(testKey string) model.Stage {
	for _, c := range tp.constraints {
		if c.Category == testKey && c.MinStage != "" {
			return c.MinStage
		}
	}
	return ""
}

func (tp *TestPlacer) budgetForStage(stage model.Stage) float64 {
	switch stage {
	case model.StagePR:
		return tp.config.PRMissRateBudget
	case model.StageMergeQueue:
		return tp.config.MQMissRateBudget
	case model.StagePostMerge:
		return tp.config.PostMergeMissRateBudget
	default:
		return 0 // nightly: no deferral
	}
}

func nextPlacementStage(s model.Stage) model.Stage {
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
