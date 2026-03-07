package engine

import (
	"encoding/json"
	"math"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Correlation represents a co-failure relationship between two tests.
type Correlation struct {
	TestA      string  `json:"test_a"`
	TestB      string  `json:"test_b"`
	CoFailRate float64 `json:"co_fail_rate"` // P(B fails | A fails)
	SampleSize int     `json:"sample_size"`  // how many A-failures observed
	SpeedRatio float64 `json:"speed_ratio"`  // duration_B / duration_A
	Confidence float64 `json:"confidence"`   // statistical confidence
}

// CorrelationMatrix is the full set of discovered correlations.
type CorrelationMatrix struct {
	Correlations      []Correlation `json:"correlations"`
	ComputedAt        time.Time     `json:"computed_at"`
	WindowStart       time.Time     `json:"window_start"`
	WindowEnd         time.Time     `json:"window_end"`
	PipelinesAnalyzed int           `json:"pipelines_analyzed"`
}

// CorrelationEngine analyzes historical test failures to find co-failure patterns.
type CorrelationEngine struct {
	store events.Store
}

// NewCorrelationEngine creates a new CorrelationEngine.
func NewCorrelationEngine(store events.Store) *CorrelationEngine {
	return &CorrelationEngine{store: store}
}

// CorrelationConfig controls correlation computation thresholds.
type CorrelationConfig struct {
	MinCoFailRate float64 // minimum co-failure rate to include (default: 0.9)
	MinSampleSize int     // minimum A-failures to include (default: 10)
}

// DefaultCorrelationConfig returns sensible defaults.
func DefaultCorrelationConfig() CorrelationConfig {
	return CorrelationConfig{
		MinCoFailRate: 0.9,
		MinSampleSize: 10,
	}
}

// Compute analyzes events in [start, end] and returns the correlation matrix.
func (ce *CorrelationEngine) Compute(start, end time.Time, config CorrelationConfig) (*CorrelationMatrix, error) {
	filter := events.EventFilter{
		Types:  []model.EventType{model.EventTestFailed, model.EventRealFailure},
		After:  start,
		Before: end,
	}
	evts, err := ce.store.Query(filter)
	if err != nil {
		return nil, err
	}

	// Group failures by pipeline_id.
	pipelineFailures := make(map[string]map[string]time.Duration) // pipeline -> test_key -> duration
	for _, evt := range evts {
		testKey, duration := extractTestInfo(evt)
		if testKey == "" {
			continue
		}
		if pipelineFailures[evt.PipelineID] == nil {
			pipelineFailures[evt.PipelineID] = make(map[string]time.Duration)
		}
		pipelineFailures[evt.PipelineID][testKey] = duration
	}

	matrix := computeFromFailureMap(pipelineFailures, config)
	matrix.WindowStart = start
	matrix.WindowEnd = end
	return matrix, nil
}

// ComputeFromFailureSets computes correlations from pre-built failure sets.
// Each entry maps pipeline ID to the set of failed test keys with durations.
// This is useful for testing without needing a real event store.
func ComputeFromFailureSets(failureSets map[string]map[string]time.Duration, config CorrelationConfig) *CorrelationMatrix {
	return computeFromFailureMap(failureSets, config)
}

// computeFromFailureMap is the shared implementation for both Compute and ComputeFromFailureSets.
func computeFromFailureMap(pipelineFailures map[string]map[string]time.Duration, config CorrelationConfig) *CorrelationMatrix {
	// failCount[A] = number of pipelines where A failed
	// coFailCount[A][B] = number of pipelines where both A and B failed
	failCount := make(map[string]int)
	coFailCount := make(map[string]map[string]int)
	totalDuration := make(map[string]time.Duration)
	durationCount := make(map[string]int)

	for _, failures := range pipelineFailures {
		keys := make([]string, 0, len(failures))
		for k, d := range failures {
			keys = append(keys, k)
			failCount[k]++
			totalDuration[k] += d
			durationCount[k]++
		}
		// Record co-occurrences.
		for i, a := range keys {
			for j, b := range keys {
				if i == j {
					continue
				}
				if coFailCount[a] == nil {
					coFailCount[a] = make(map[string]int)
				}
				coFailCount[a][b]++
			}
		}
	}

	// Compute correlations.
	var correlations []Correlation
	for a, bMap := range coFailCount {
		for b, coCount := range bMap {
			aCount := failCount[a]
			if aCount < config.MinSampleSize {
				continue
			}
			coFailRate := float64(coCount) / float64(aCount)
			if coFailRate < config.MinCoFailRate {
				continue
			}

			// Compute speed ratio.
			var speedRatio float64
			if durationCount[a] > 0 && durationCount[b] > 0 {
				avgA := float64(totalDuration[a]) / float64(durationCount[a])
				avgB := float64(totalDuration[b]) / float64(durationCount[b])
				if avgA > 0 {
					speedRatio = avgB / avgA
				}
			}

			// Wilson score confidence interval (lower bound).
			confidence := wilsonLowerBound(coCount, aCount)

			correlations = append(correlations, Correlation{
				TestA:      a,
				TestB:      b,
				CoFailRate: coFailRate,
				SampleSize: aCount,
				SpeedRatio: speedRatio,
				Confidence: confidence,
			})
		}
	}

	return &CorrelationMatrix{
		Correlations:      correlations,
		ComputedAt:        time.Now().UTC(),
		PipelinesAnalyzed: len(pipelineFailures),
	}
}

// extractTestInfo extracts a test key and duration from a test failure event.
// Handles two payload shapes:
//   - FlakePayload / result-based: {"result": {"test": {...}, "duration": ...}}
//   - Map-based (EventRealFailure): {"test": {"name": ..., "package": ...}}
func extractTestInfo(evt model.Event) (string, time.Duration) {
	// Try result-based payload first (FlakePayload shape).
	var resultPayload struct {
		Result model.TestResult `json:"result"`
	}
	if err := json.Unmarshal(evt.Payload, &resultPayload); err == nil {
		if resultPayload.Result.Test.Package != "" || resultPayload.Result.Test.Name != "" {
			return resultPayload.Result.Test.Key(), resultPayload.Result.Duration
		}
	}

	// Try map-based payload (EventRealFailure shape).
	var mapPayload struct {
		Test model.TestIdentifier `json:"test"`
	}
	if err := json.Unmarshal(evt.Payload, &mapPayload); err == nil {
		if mapPayload.Test.Package != "" || mapPayload.Test.Name != "" {
			return mapPayload.Test.Key(), 0
		}
	}

	return "", 0
}

// wilsonLowerBound computes the lower bound of a Wilson score confidence interval
// at 95% confidence (z = 1.96).
func wilsonLowerBound(successes, total int) float64 {
	if total == 0 {
		return 0
	}
	n := float64(total)
	p := float64(successes) / n

	z := 1.96 // 95% confidence
	denominator := 1 + z*z/n
	centre := p + z*z/(2*n)
	adjust := z * math.Sqrt((p*(1-p)+z*z/(4*n))/n)

	return (centre - adjust) / denominator
}
