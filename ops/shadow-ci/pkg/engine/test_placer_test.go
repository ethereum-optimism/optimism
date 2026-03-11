package engine

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func TestTestPlacer_MarginalCoverage(t *testing.T) {
	// TestA is fast (2s), TestB is slow (45s), they co-fail 99.5% of the time.
	// TestB should be deferred because TestA covers the same failure mode faster.
	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{
				TestA:      "pkg/TestA",
				TestB:      "pkg/TestB",
				CoFailRate: 0.995,
				SampleSize: 100,
				SpeedRatio: 22.5, // TestB is 22.5x slower than TestA
				Confidence: 0.98,
			},
		},
	}

	stats := map[string]*TestStats{
		"pkg/TestA": {TestKey: "pkg/TestA", MeanDuration: 2 * time.Second},
		"pkg/TestB": {TestKey: "pkg/TestB", MeanDuration: 45 * time.Second},
		"pkg/TestC": {TestKey: "pkg/TestC", MeanDuration: 5 * time.Second},
	}

	placer := NewTestPlacer(correlations, stats, nil, DefaultTestPlacerConfig())
	placements := placer.PlaceTests([]string{"pkg/TestA", "pkg/TestB", "pkg/TestC"}, model.StagePR)

	if len(placements) != 3 {
		t.Fatalf("expected 3 placements, got %d", len(placements))
	}

	// Find TestB's placement.
	var testBPlacement *model.TestPlacement
	for i := range placements {
		if placements[i].TestKey == "pkg/TestB" {
			testBPlacement = &placements[i]
			break
		}
	}
	if testBPlacement == nil {
		t.Fatal("TestB not found in placements")
	}

	if !testBPlacement.WouldDefer {
		t.Error("TestB should be marked WouldDefer (TestA covers the same failure mode faster)")
	}
	if testBPlacement.DeferTo != model.StageMergeQueue {
		t.Errorf("TestB DeferTo = %s, want merge_queue", testBPlacement.DeferTo)
	}

	// TestA should NOT be deferred (it's fast with high marginal value).
	var testAPlacement *model.TestPlacement
	for i := range placements {
		if placements[i].TestKey == "pkg/TestA" {
			testAPlacement = &placements[i]
			break
		}
	}
	if testAPlacement == nil {
		t.Fatal("TestA not found in placements")
	}
	if testAPlacement.WouldDefer {
		t.Error("TestA should NOT be deferred")
	}
}

func TestTestPlacer_BudgetExhaustion(t *testing.T) {
	// Many correlated tests, but only 5% budget. Once budget is consumed, no more deferrals.
	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{TestA: "fast1", TestB: "slow1", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
			{TestA: "fast2", TestB: "slow2", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
			{TestA: "fast3", TestB: "slow3", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
			{TestA: "fast4", TestB: "slow4", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
			{TestA: "fast5", TestB: "slow5", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
			{TestA: "fast6", TestB: "slow6", CoFailRate: 0.99, SpeedRatio: 10, SampleSize: 50},
		},
	}

	stats := make(map[string]*TestStats)
	allKeys := []string{}
	for i := 1; i <= 6; i++ {
		fast := "fast" + string(rune('0'+i))
		slow := "slow" + string(rune('0'+i))
		stats[fast] = &TestStats{TestKey: fast, MeanDuration: 1 * time.Second}
		stats[slow] = &TestStats{TestKey: slow, MeanDuration: 30 * time.Second}
		allKeys = append(allKeys, fast, slow)
	}

	config := DefaultTestPlacerConfig()
	config.PRMissRateBudget = 0.05 // 5% budget

	placer := NewTestPlacer(correlations, stats, nil, config)
	placements := placer.PlaceTests(allKeys, model.StagePR)

	// Each slow test has marginal value ~0.01 (1 - 0.99).
	// Budget = 0.05, each deferral costs 0.01 → can defer 5 of 6 slow tests.
	deferredCount := 0
	for _, p := range placements {
		if p.WouldDefer {
			deferredCount++
		}
	}

	// Budget allows at most 5 deferrals (5 * 0.01 = 0.05).
	if deferredCount > 5 {
		t.Errorf("deferred %d tests, but budget should limit to 5", deferredCount)
	}
	if deferredCount < 4 {
		t.Errorf("deferred only %d tests, expected at least 4 with 5%% budget", deferredCount)
	}
}

func TestTestPlacer_PinnedConstraint(t *testing.T) {
	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{TestA: "fast", TestB: "slow-pinned", CoFailRate: 0.999, SpeedRatio: 20, SampleSize: 100},
		},
	}

	stats := map[string]*TestStats{
		"fast":        {TestKey: "fast", MeanDuration: 1 * time.Second},
		"slow-pinned": {TestKey: "slow-pinned", MeanDuration: 60 * time.Second},
	}

	constraints := []model.PlacementConstraint{
		{Category: "slow-pinned", PinnedStage: model.StagePR, Reason: "critical test"},
	}

	placer := NewTestPlacer(correlations, stats, constraints, DefaultTestPlacerConfig())
	placements := placer.PlaceTests([]string{"fast", "slow-pinned"}, model.StagePR)

	for _, p := range placements {
		if p.TestKey == "slow-pinned" {
			if p.WouldDefer {
				t.Error("pinned test should never be deferred")
			}
			if p.AssignedStage != model.StagePR {
				t.Errorf("pinned test stage = %s, want pr", p.AssignedStage)
			}
			return
		}
	}
	t.Fatal("slow-pinned not found in placements")
}

func TestTestPlacer_ShadowMode(t *testing.T) {
	correlations := &CorrelationMatrix{
		Correlations: []Correlation{
			{TestA: "fast", TestB: "slow", CoFailRate: 0.99, SpeedRatio: 15, SampleSize: 50},
		},
	}

	stats := map[string]*TestStats{
		"fast": {TestKey: "fast", MeanDuration: 2 * time.Second},
		"slow": {TestKey: "slow", MeanDuration: 30 * time.Second},
	}

	config := DefaultTestPlacerConfig()
	config.ShadowMode = true

	placer := NewTestPlacer(correlations, stats, nil, config)
	placements := placer.PlaceTests([]string{"fast", "slow"}, model.StagePR)

	// In shadow mode, all tests get placements but WouldDefer is set for the slow one.
	if len(placements) != 2 {
		t.Fatalf("expected 2 placements, got %d", len(placements))
	}

	// Both tests should have placements — shadow mode doesn't remove any.
	hasDeferred := false
	for _, p := range placements {
		if p.WouldDefer {
			hasDeferred = true
		}
	}
	if !hasDeferred {
		t.Error("expected at least one test marked WouldDefer in shadow mode")
	}
}

func TestTestPlacer_LLMOverride(t *testing.T) {
	stats := map[string]*TestStats{
		"test1": {TestKey: "test1", MeanDuration: 5 * time.Second},
		"test2": {TestKey: "test2", MeanDuration: 30 * time.Second},
	}

	placer := NewTestPlacer(nil, stats, nil, DefaultTestPlacerConfig())

	// LLM says test2 should be promoted to PR stage.
	placer.SetLLMOverrides([]PlacementOverride{
		{TestKey: "test2", OverrideTo: model.StagePR, Reason: "diff changes error handling", Confidence: 0.8},
	})

	placements := placer.PlaceTests([]string{"test1", "test2"}, model.StagePR)

	for _, p := range placements {
		if p.TestKey == "test2" {
			if p.AssignedStage != model.StagePR {
				t.Errorf("test2 stage = %s, want pr (LLM override)", p.AssignedStage)
			}
			if p.WouldDefer {
				t.Error("test2 should not be deferred after LLM override to PR")
			}
			return
		}
	}
	t.Fatal("test2 not found in placements")
}

func TestTestPlacer_LLMOverridePinnedConflict(t *testing.T) {
	stats := map[string]*TestStats{
		"pinned-test": {TestKey: "pinned-test", MeanDuration: 10 * time.Second},
	}

	constraints := []model.PlacementConstraint{
		{Category: "pinned-test", PinnedStage: model.StagePR, Reason: "critical"},
	}

	placer := NewTestPlacer(nil, stats, constraints, DefaultTestPlacerConfig())
	placer.SetLLMOverrides([]PlacementOverride{
		{TestKey: "pinned-test", OverrideTo: model.StageNightly, Reason: "LLM thinks it's safe", Confidence: 0.9},
	})

	placements := placer.PlaceTests([]string{"pinned-test"}, model.StagePR)

	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}
	// Pinned constraint should win over LLM override.
	if placements[0].AssignedStage != model.StagePR {
		t.Errorf("pinned test stage = %s, want pr (constraint should override LLM)", placements[0].AssignedStage)
	}
}

func TestTestPlacer_NightlyNoDeferral(t *testing.T) {
	stats := map[string]*TestStats{
		"test1": {TestKey: "test1", MeanDuration: 100 * time.Second},
	}

	placer := NewTestPlacer(nil, stats, nil, DefaultTestPlacerConfig())
	placements := placer.PlaceTests([]string{"test1"}, model.StageNightly)

	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}
	if placements[0].WouldDefer {
		t.Error("nightly stage should never defer (budget = 0)")
	}
}

func TestTestPlacer_ColdStart(t *testing.T) {
	// No stats, no correlations — everything gets full marginal value and runs at current stage.
	placer := NewTestPlacer(nil, nil, nil, DefaultTestPlacerConfig())
	placements := placer.PlaceTests([]string{"new-test-1", "new-test-2"}, model.StagePR)

	if len(placements) != 2 {
		t.Fatalf("expected 2 placements, got %d", len(placements))
	}

	for _, p := range placements {
		if p.WouldDefer {
			t.Errorf("cold start test %s should not be deferred (marginal value = 1.0 exceeds budget)", p.TestKey)
		}
	}
}
