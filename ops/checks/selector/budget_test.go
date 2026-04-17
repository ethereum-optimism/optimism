package selector

import (
	"testing"
	"time"
)

// TestTrimToBudget_NoOpWhenUnderBudget — a plan already fitting the
// budget is left unchanged.
func TestTrimToBudget_NoOpWhenUnderBudget(t *testing.T) {
	result := &Result{
		Items: []ExecutionItem{
			{ID: "a", RunCost: 60, SkipCost: 100},
			{ID: "b", RunCost: 60, SkipCost: 200},
		},
	}
	result.Schedule = ComputeSchedule(result.Items, 4)
	result.WallClock = result.Schedule.WallClock

	before := len(result.Items)
	TrimToBudget(result, 30*time.Minute, 4)
	if len(result.Items) != before {
		t.Errorf("under-budget plan was trimmed: %d → %d", before, len(result.Items))
	}
}

// TestTrimToBudget_DropsLowestDensity — when over budget, the lowest
// value-density item is removed first.
func TestTrimToBudget_DropsLowestDensity(t *testing.T) {
	result := &Result{
		Items: []ExecutionItem{
			{ID: "high-value", RunCost: 100, SkipCost: 300}, // density 3.0
			{ID: "low-value", RunCost: 100, SkipCost: 50},   // density 0.5
			{ID: "med-value", RunCost: 100, SkipCost: 150},  // density 1.5
		},
	}
	result.Schedule = ComputeSchedule(result.Items, 1) // single slot, serial
	result.WallClock = result.Schedule.WallClock       // 300

	TrimToBudget(result, 250*time.Second, 1)

	kept := make(map[string]bool)
	for _, item := range result.Items {
		kept[item.ID] = true
	}
	if kept["low-value"] {
		t.Errorf("low-density item should have been dropped, got: %v", kept)
	}
	if !kept["high-value"] || !kept["med-value"] {
		t.Errorf("high/med-density items should be kept, got: %v", kept)
	}

	// Dropped item moves to Skipped, not discarded.
	inSkipped := false
	for _, item := range result.Skipped {
		if item.ID == "low-value" {
			inSkipped = true
		}
	}
	if !inSkipped {
		t.Error("low-value item should be in Skipped (not silently lost)")
	}
}

// TestTrimToBudget_ForceRunItemsProtected — items where
// SkipCost = RunCost + 1 (the optimizer's force-run pattern) are
// never dropped even when over budget.
func TestTrimToBudget_ForceRunItemsProtected(t *testing.T) {
	result := &Result{
		Items: []ExecutionItem{
			// Force-run: SkipCost barely exceeds RunCost.
			{ID: "force", RunCost: 1000, SkipCost: 1001},
			// Normal item, lower density.
			{ID: "normal", RunCost: 100, SkipCost: 200},
		},
	}
	result.Schedule = ComputeSchedule(result.Items, 1)
	result.WallClock = result.Schedule.WallClock

	TrimToBudget(result, 10*time.Second, 1) // way under budget

	kept := make(map[string]bool)
	for _, item := range result.Items {
		kept[item.ID] = true
	}
	if !kept["force"] {
		t.Error("force-run item must not be dropped")
	}
	if kept["normal"] {
		t.Error("normal item should have been dropped to fit budget")
	}
}

// TestTrimToBudget_PrerequisitesProtected — a protected item's
// prerequisites stay too, even if their own density would drop them.
func TestTrimToBudget_PrerequisitesProtected(t *testing.T) {
	result := &Result{
		Items: []ExecutionItem{
			{ID: "forge-test", RunCost: 1000, SkipCost: 1001, Prerequisites: []string{"forge-build"}},
			{ID: "forge-build", RunCost: 100, SkipCost: 0}, // prerequisite with no standalone value
			{ID: "lint", RunCost: 50, SkipCost: 60},
		},
	}
	result.Schedule = ComputeSchedule(result.Items, 1)
	result.WallClock = result.Schedule.WallClock

	TrimToBudget(result, 600*time.Second, 1)

	kept := make(map[string]bool)
	for _, item := range result.Items {
		kept[item.ID] = true
	}
	if !kept["forge-test"] {
		t.Error("force-run forge-test must stay")
	}
	if !kept["forge-build"] {
		t.Error("prerequisite forge-build must stay with its consumer")
	}
}

// TestTrimToBudget_ScheduleRecomputed — after trimming, WallClock and
// Schedule reflect the remaining items.
func TestTrimToBudget_ScheduleRecomputed(t *testing.T) {
	result := &Result{
		Items: []ExecutionItem{
			{ID: "a", RunCost: 100, SkipCost: 50},  // low density
			{ID: "b", RunCost: 100, SkipCost: 500}, // high density
		},
	}
	result.Schedule = ComputeSchedule(result.Items, 1)
	result.WallClock = result.Schedule.WallClock // 200 serial

	TrimToBudget(result, 150*time.Second, 1)

	if result.WallClock > 150 {
		t.Errorf("WallClock after trim = %f, want ≤ 150", result.WallClock)
	}
	// Schedule should match the kept items.
	total := 0
	for _, layer := range result.Schedule.Layers {
		total += len(layer.ItemIDs)
	}
	if total != len(result.Items) {
		t.Errorf("schedule covers %d items, plan has %d", total, len(result.Items))
	}
}
