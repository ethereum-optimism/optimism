package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelation_PerfectCorrelation(t *testing.T) {
	// A and B always fail together.
	sets := make(map[string]map[string]time.Duration)
	for i := 0; i < 15; i++ {
		pid := fmt.Sprintf("pipeline-%d", i)
		sets[pid] = map[string]time.Duration{
			"pkg/foo/TestA": 1 * time.Second,
			"pkg/foo/TestB": 10 * time.Second,
		}
	}

	matrix := ComputeFromFailureSets(sets, CorrelationConfig{MinCoFailRate: 0.9, MinSampleSize: 10})

	// Should have A->B and B->A correlations.
	require.GreaterOrEqual(t, len(matrix.Correlations), 2)

	var found bool
	for _, c := range matrix.Correlations {
		if c.TestA == "pkg/foo/TestA" && c.TestB == "pkg/foo/TestB" {
			assert.Equal(t, 1.0, c.CoFailRate)
			assert.Equal(t, 15, c.SampleSize)
			assert.InDelta(t, 10.0, c.SpeedRatio, 0.01)
			found = true
		}
	}
	assert.True(t, found, "expected A->B correlation")
}

func TestCorrelation_NoCorrelation(t *testing.T) {
	// A and B never fail together.
	sets := make(map[string]map[string]time.Duration)
	for i := 0; i < 20; i++ {
		pid := fmt.Sprintf("pipeline-%d", i)
		if i%2 == 0 {
			sets[pid] = map[string]time.Duration{"pkg/foo/TestA": time.Second}
		} else {
			sets[pid] = map[string]time.Duration{"pkg/foo/TestB": time.Second}
		}
	}

	matrix := ComputeFromFailureSets(sets, DefaultCorrelationConfig())
	assert.Empty(t, matrix.Correlations)
}

func TestCorrelation_MinSampleSize(t *testing.T) {
	// Only 5 observations -- below threshold.
	sets := make(map[string]map[string]time.Duration)
	for i := 0; i < 5; i++ {
		pid := fmt.Sprintf("pipeline-%d", i)
		sets[pid] = map[string]time.Duration{
			"pkg/foo/TestA": time.Second,
			"pkg/foo/TestB": time.Second,
		}
	}

	matrix := ComputeFromFailureSets(sets, DefaultCorrelationConfig())
	assert.Empty(t, matrix.Correlations)
}

func TestCorrelation_PartialCorrelation(t *testing.T) {
	// A fails 20 times, B co-fails 16 times (80%).
	sets := make(map[string]map[string]time.Duration)
	for i := 0; i < 20; i++ {
		pid := fmt.Sprintf("pipeline-%d", i)
		if i < 16 {
			sets[pid] = map[string]time.Duration{
				"pkg/foo/TestA": time.Second,
				"pkg/foo/TestB": time.Second,
			}
		} else {
			sets[pid] = map[string]time.Duration{
				"pkg/foo/TestA": time.Second,
			}
		}
	}

	// 80% is below the 90% threshold.
	matrix := ComputeFromFailureSets(sets, DefaultCorrelationConfig())

	var found bool
	for _, c := range matrix.Correlations {
		if c.TestA == "pkg/foo/TestA" && c.TestB == "pkg/foo/TestB" {
			found = true
		}
	}
	assert.False(t, found, "80%% co-fail rate should be below 90%% threshold")
}

func TestCorrelation_SpeedRatio(t *testing.T) {
	sets := make(map[string]map[string]time.Duration)
	for i := 0; i < 15; i++ {
		pid := fmt.Sprintf("pipeline-%d", i)
		sets[pid] = map[string]time.Duration{
			"pkg/foo/TestFast": 1 * time.Second,
			"pkg/foo/TestSlow": 10 * time.Second,
		}
	}

	matrix := ComputeFromFailureSets(sets, CorrelationConfig{MinCoFailRate: 0.9, MinSampleSize: 10})

	for _, c := range matrix.Correlations {
		if c.TestA == "pkg/foo/TestFast" && c.TestB == "pkg/foo/TestSlow" {
			assert.InDelta(t, 10.0, c.SpeedRatio, 0.01)
		}
	}
}

func TestCorrelation_EmptyEvents(t *testing.T) {
	sets := make(map[string]map[string]time.Duration)
	matrix := ComputeFromFailureSets(sets, DefaultCorrelationConfig())
	assert.Empty(t, matrix.Correlations)
	assert.Equal(t, 0, matrix.PipelinesAnalyzed)
}
