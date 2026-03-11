package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// StatsAggregator computes per-test and per-category statistics from the event store.
type StatsAggregator struct {
	store events.Store
}

// TestStats holds aggregated statistics for a single test.
type TestStats struct {
	TestKey        string        `json:"test_key"`
	TotalRuns      int           `json:"total_runs"`
	PassCount      int           `json:"pass_count"`
	FailCount      int           `json:"fail_count"`
	FlakeCount     int           `json:"flake_count"`
	MeanDuration   time.Duration `json:"mean_duration"`
	P95Duration    time.Duration `json:"p95_duration"`
	LastSeen       time.Time     `json:"last_seen"`
	FalseNegatives int           `json:"false_negatives"`

	// Internal: durations collected for percentile computation.
	durations []time.Duration
}

// NewStatsAggregator creates a new StatsAggregator.
func NewStatsAggregator(store events.Store) *StatsAggregator {
	return &StatsAggregator{store: store}
}

// ComputeTestStats aggregates per-test stats from events in [start, end].
func (sa *StatsAggregator) ComputeTestStats(start, end time.Time) (map[string]*TestStats, error) {
	// Query only test-relevant event types for performance.
	filter := events.EventFilter{
		Types: []model.EventType{
			model.EventTestPassed,
			model.EventTestFailed,
			model.EventFlakeDetected,
			model.EventRealFailure,
			model.EventFalseNegative,
		},
		After:  start,
		Before: end,
	}

	evts, err := sa.store.Query(filter)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}

	if len(evts) == 0 {
		log.Printf("warning: no test events in lookback window [%s, %s] — optimizer will produce no recommendations",
			start.Format(time.RFC3339), end.Format(time.RFC3339))
		return make(map[string]*TestStats), nil
	}

	stats := make(map[string]*TestStats)

	for _, evt := range evts {
		testKey, duration := extractTestStatsInfo(evt)
		if testKey == "" {
			continue
		}

		ts, ok := stats[testKey]
		if !ok {
			ts = &TestStats{TestKey: testKey}
			stats[testKey] = ts
		}

		if evt.Timestamp.After(ts.LastSeen) {
			ts.LastSeen = evt.Timestamp
		}

		switch evt.Type {
		case model.EventTestPassed:
			ts.TotalRuns++
			ts.PassCount++
			if duration > 0 {
				ts.durations = append(ts.durations, duration)
			}
		case model.EventTestFailed:
			ts.TotalRuns++
			ts.FailCount++
			if duration > 0 {
				ts.durations = append(ts.durations, duration)
			}
		case model.EventFlakeDetected:
			ts.FlakeCount++
		case model.EventRealFailure:
			// RealFailure events are classification confirmations, not separate runs.
			// Don't double-count TotalRuns.
		case model.EventFalseNegative:
			ts.FalseNegatives++
		}
	}

	// Compute duration statistics.
	for _, ts := range stats {
		if len(ts.durations) > 0 {
			ts.MeanDuration = meanDuration(ts.durations)
			ts.P95Duration = percentileDuration(ts.durations, 0.95)
		}
	}

	return stats, nil
}

// ComputeCategoryStats maps test-level stats to category-level stats using scoping config.
func (sa *StatsAggregator) ComputeCategoryStats(
	testStats map[string]*TestStats,
	scoping model.ScopingConfig,
) (map[string]*CategoryStats, error) {
	result := make(map[string]*CategoryStats)

	// Build a mapping: for each category, determine which test keys belong to it.
	// Graph-based categories match by language prefix in the test key.
	// Non-graph categories are aggregated from their trigger paths.
	for catName, cat := range scoping.JobCategories {
		cs := &CategoryStats{}
		matched := 0

		for testKey, ts := range testStats {
			if !testBelongsToCategory(testKey, cat) {
				continue
			}
			matched++
			cs.TotalRuns += ts.TotalRuns
			cs.FalseNegativeCount += ts.FalseNegatives

			if ts.TotalRuns > 0 {
				cs.FlakeRate += float64(ts.FlakeCount)
				cs.RealFailureRate += float64(ts.FailCount)
			}
			cs.MeanDuration += ts.MeanDuration
		}

		if matched == 0 {
			continue
		}

		// Normalize rates.
		if cs.TotalRuns > 0 {
			cs.FlakeRate /= float64(cs.TotalRuns)
			cs.RealFailureRate /= float64(cs.TotalRuns)
		}
		// Average duration across tests in the category.
		cs.MeanDuration /= time.Duration(matched)

		result[catName] = cs
	}

	return result, nil
}

// testBelongsToCategory determines if a test key belongs to a category.
// For graph-based categories (use_graph + language), match by language prefix convention:
//   - Go tests: key starts with "github.com/ethereum-optimism/optimism/"
//   - Sol tests: key contains ".sol" or starts with known sol paths
//   - Rust tests: key starts with known rust crate names
//
// For non-graph categories, we can't reliably map individual tests without
// additional metadata, so these are skipped for per-test stats.
func testBelongsToCategory(testKey string, cat model.JobCategoryConfig) bool {
	if cat.Language == "" {
		return false
	}

	switch cat.Language {
	case "go":
		// Go test keys look like "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive"
		return len(testKey) > 0 && testKey[0] != '/' && !isNonGoTestKey(testKey)
	case "sol":
		return isForgeTestKey(testKey)
	case "rust":
		return isRustTestKey(testKey)
	default:
		return false
	}
}

func isNonGoTestKey(key string) bool {
	return isForgeTestKey(key) || isRustTestKey(key)
}

func isForgeTestKey(key string) bool {
	// Forge test keys typically contain ".sol/" or "::test_" patterns.
	for i := 0; i < len(key)-4; i++ {
		if key[i:i+4] == ".sol" {
			return true
		}
	}
	return false
}

func isRustTestKey(key string) bool {
	// Rust test keys use "::" separator (e.g., "kona_derive::test_channel")
	for i := 0; i < len(key)-2; i++ {
		if key[i:i+2] == "::" {
			return true
		}
	}
	return false
}

// extractTestStatsInfo extracts a test key and duration from an event payload.
// Handles multiple payload shapes used across different event types.
func extractTestStatsInfo(evt model.Event) (string, time.Duration) {
	// Try test/duration shape (EventTestPassed, EventTestFailed).
	var testPayload struct {
		Test     model.TestIdentifier `json:"test"`
		Duration float64              `json:"duration"` // seconds
	}
	if err := json.Unmarshal(evt.Payload, &testPayload); err == nil {
		if testPayload.Test.Package != "" || testPayload.Test.Name != "" {
			return testPayload.Test.Key(), time.Duration(testPayload.Duration * float64(time.Second))
		}
	}

	// Try FlakePayload shape (EventFlakeDetected).
	var flakePayload model.FlakePayload
	if err := json.Unmarshal(evt.Payload, &flakePayload); err == nil {
		if flakePayload.Result.Test.Package != "" || flakePayload.Result.Test.Name != "" {
			return flakePayload.Result.Test.Key(), flakePayload.Result.Duration
		}
	}

	// Try FalseNegativeDetail shape (EventFalseNegative).
	var fnPayload model.FalseNegativeDetail
	if err := json.Unmarshal(evt.Payload, &fnPayload); err == nil {
		if fnPayload.Test.Package != "" || fnPayload.Test.Name != "" {
			return fnPayload.Test.Key(), 0
		}
	}

	// Try map-based shape (EventRealFailure).
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

func meanDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func percentileDuration(durations []time.Duration, pct float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(len(sorted)-1) * pct)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
