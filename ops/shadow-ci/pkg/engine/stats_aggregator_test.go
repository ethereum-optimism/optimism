package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// mockStore implements events.Store for testing.
type mockStore struct {
	events []model.Event
}

func (m *mockStore) Emit(event model.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockStore) Query(filter events.EventFilter) ([]model.Event, error) {
	var result []model.Event
	for _, e := range m.events {
		if matchesMockFilter(e, filter) {
			result = append(result, e)
		}
	}
	return result, nil
}

func matchesMockFilter(e model.Event, f events.EventFilter) bool {
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if e.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if !f.After.IsZero() && e.Timestamp.Before(f.After) {
		return false
	}
	if !f.Before.IsZero() && e.Timestamp.After(f.Before) {
		return false
	}
	return true
}

func makeTestEvent(t *testing.T, eventType model.EventType, pkg, name string, durationSec float64, ts time.Time) model.Event {
	t.Helper()
	var payload []byte
	var err error

	switch eventType {
	case model.EventTestPassed, model.EventTestFailed:
		payload, err = json.Marshal(map[string]any{
			"test": model.TestIdentifier{
				Package: pkg,
				Name:    name,
			},
			"duration": durationSec,
		})
	case model.EventFlakeDetected:
		payload, err = json.Marshal(model.FlakePayload{
			Result: model.TestResult{
				Test:     model.TestIdentifier{Package: pkg, Name: name},
				Duration: time.Duration(durationSec * float64(time.Second)),
			},
			Fingerprint: "flake-fp",
		})
	case model.EventFalseNegative:
		payload, err = json.Marshal(model.FalseNegativeDetail{
			Test:     model.TestIdentifier{Package: pkg, Name: name},
			Language: "go",
		})
	case model.EventRealFailure:
		payload, err = json.Marshal(map[string]any{
			"test": model.TestIdentifier{Package: pkg, Name: name},
		})
	}
	if err != nil {
		t.Fatal(err)
	}

	return model.Event{
		ID:        "evt-" + name,
		Type:      eventType,
		Timestamp: ts,
		Payload:   payload,
	}
}

func TestStatsAggregator_ComputeTestStats(t *testing.T) {
	now := time.Now().UTC()
	store := &mockStore{
		events: []model.Event{
			makeTestEvent(t, model.EventTestPassed, "github.com/ethereum-optimism/optimism/op-node/rollup", "TestDerive", 2.5, now.Add(-1*time.Hour)),
			makeTestEvent(t, model.EventTestPassed, "github.com/ethereum-optimism/optimism/op-node/rollup", "TestDerive", 3.0, now.Add(-30*time.Minute)),
			makeTestEvent(t, model.EventTestFailed, "github.com/ethereum-optimism/optimism/op-node/rollup", "TestDerive", 1.5, now.Add(-15*time.Minute)),
			makeTestEvent(t, model.EventFlakeDetected, "github.com/ethereum-optimism/optimism/op-node/rollup", "TestDerive", 1.5, now.Add(-14*time.Minute)),
			makeTestEvent(t, model.EventTestPassed, "github.com/ethereum-optimism/optimism/cannon/mips", "TestMIPS", 45.0, now.Add(-2*time.Hour)),
			makeTestEvent(t, model.EventFalseNegative, "github.com/ethereum-optimism/optimism/cannon/mips", "TestMIPS", 0, now.Add(-10*time.Minute)),
		},
	}

	agg := NewStatsAggregator(store)
	stats, err := agg.ComputeTestStats(now.Add(-3*time.Hour), now)
	if err != nil {
		t.Fatalf("ComputeTestStats: %v", err)
	}

	// TestDerive: 2 passes + 1 fail = 3 total runs, 1 flake, 0 false negatives
	deriveKey := "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive"
	ds, ok := stats[deriveKey]
	if !ok {
		t.Fatalf("expected stats for %s", deriveKey)
	}
	if ds.TotalRuns != 3 {
		t.Errorf("TotalRuns = %d, want 3", ds.TotalRuns)
	}
	if ds.PassCount != 2 {
		t.Errorf("PassCount = %d, want 2", ds.PassCount)
	}
	if ds.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", ds.FailCount)
	}
	if ds.FlakeCount != 1 {
		t.Errorf("FlakeCount = %d, want 1", ds.FlakeCount)
	}
	if ds.FalseNegatives != 0 {
		t.Errorf("FalseNegatives = %d, want 0", ds.FalseNegatives)
	}
	if ds.MeanDuration == 0 {
		t.Error("MeanDuration should be > 0")
	}

	// TestMIPS: 1 pass, 0 fails, 0 flakes, 1 false negative
	mipsKey := "github.com/ethereum-optimism/optimism/cannon/mips/TestMIPS"
	ms, ok := stats[mipsKey]
	if !ok {
		t.Fatalf("expected stats for %s", mipsKey)
	}
	if ms.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", ms.TotalRuns)
	}
	if ms.FalseNegatives != 1 {
		t.Errorf("FalseNegatives = %d, want 1", ms.FalseNegatives)
	}
}

func TestStatsAggregator_EmptyStore(t *testing.T) {
	store := &mockStore{}
	agg := NewStatsAggregator(store)

	now := time.Now().UTC()
	stats, err := agg.ComputeTestStats(now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("ComputeTestStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

func TestStatsAggregator_ComputeCategoryStats(t *testing.T) {
	testStats := map[string]*TestStats{
		"github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive": {
			TestKey:      "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive",
			TotalRuns:    100,
			PassCount:    95,
			FailCount:    5,
			FlakeCount:   3,
			MeanDuration: 2 * time.Second,
		},
		"github.com/ethereum-optimism/optimism/op-node/p2p/TestSync": {
			TestKey:        "github.com/ethereum-optimism/optimism/op-node/p2p/TestSync",
			TotalRuns:      50,
			PassCount:      48,
			FailCount:      2,
			FlakeCount:     1,
			MeanDuration:   5 * time.Second,
			FalseNegatives: 1,
		},
		"contracts/test/L1Bridge.sol/test_deposit": {
			TestKey:      "contracts/test/L1Bridge.sol/test_deposit",
			TotalRuns:    30,
			PassCount:    29,
			FailCount:    1,
			FlakeCount:   0,
			MeanDuration: 10 * time.Second,
		},
	}

	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_tests": {
				Language: "go",
				UseGraph: true,
			},
			"sol_tests": {
				Language: "sol",
				UseGraph: true,
			},
			"misc_lint": {
				// No language — should get no stats.
			},
		},
	}

	store := &mockStore{}
	agg := NewStatsAggregator(store)
	catStats, err := agg.ComputeCategoryStats(testStats, scoping)
	if err != nil {
		t.Fatalf("ComputeCategoryStats: %v", err)
	}

	// go_tests should have 2 Go tests aggregated.
	goStats, ok := catStats["go_tests"]
	if !ok {
		t.Fatal("expected stats for go_tests")
	}
	if goStats.TotalRuns != 150 { // 100 + 50
		t.Errorf("go_tests TotalRuns = %d, want 150", goStats.TotalRuns)
	}
	if goStats.FalseNegativeCount != 1 {
		t.Errorf("go_tests FalseNegativeCount = %d, want 1", goStats.FalseNegativeCount)
	}

	// sol_tests should have 1 Sol test.
	solStats, ok := catStats["sol_tests"]
	if !ok {
		t.Fatal("expected stats for sol_tests")
	}
	if solStats.TotalRuns != 30 {
		t.Errorf("sol_tests TotalRuns = %d, want 30", solStats.TotalRuns)
	}

	// misc_lint should not appear (no language, no matching tests).
	if _, ok := catStats["misc_lint"]; ok {
		t.Error("misc_lint should not have stats")
	}
}

func TestTestBelongsToCategory(t *testing.T) {
	tests := []struct {
		name    string
		testKey string
		cat     model.JobCategoryConfig
		want    bool
	}{
		{
			name:    "go test key matches go category",
			testKey: "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive",
			cat:     model.JobCategoryConfig{Language: "go"},
			want:    true,
		},
		{
			name:    "sol test key matches sol category",
			testKey: "contracts/test/L1Bridge.sol/test_deposit",
			cat:     model.JobCategoryConfig{Language: "sol"},
			want:    true,
		},
		{
			name:    "rust test key matches rust category",
			testKey: "kona_derive::test_channel_open",
			cat:     model.JobCategoryConfig{Language: "rust"},
			want:    true,
		},
		{
			name:    "go test key does not match sol category",
			testKey: "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive",
			cat:     model.JobCategoryConfig{Language: "sol"},
			want:    false,
		},
		{
			name:    "no language category matches nothing",
			testKey: "github.com/ethereum-optimism/optimism/op-node/rollup/TestDerive",
			cat:     model.JobCategoryConfig{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testBelongsToCategory(tt.testKey, tt.cat)
			if got != tt.want {
				t.Errorf("testBelongsToCategory(%q, %+v) = %v, want %v", tt.testKey, tt.cat, got, tt.want)
			}
		})
	}
}

func TestDurationStats(t *testing.T) {
	durations := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		4 * time.Second,
		100 * time.Second,
	}

	mean := meanDuration(durations)
	if mean != 22*time.Second {
		t.Errorf("mean = %v, want 22s", mean)
	}

	p95 := percentileDuration(durations, 0.95)
	// With 5 elements, idx = int(4 * 0.95) = 3 → 4th element (4s).
	if p95 != 4*time.Second {
		t.Errorf("p95 = %v, want 4s", p95)
	}

	// Empty case.
	if meanDuration(nil) != 0 {
		t.Error("meanDuration(nil) should be 0")
	}
	if percentileDuration(nil, 0.95) != 0 {
		t.Error("percentileDuration(nil, 0.95) should be 0")
	}
}
