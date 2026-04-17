package selector

import (
	"sort"
	"time"
)

// TrimToBudget drops the lowest-value items from a Result until the
// schedule's wall-clock estimate fits under budget. Value density is
// SkipCost / RunCost — the regret-per-second-of-execution. Items are
// removed lowest-density-first; force-run items (SkipCost > RunCost)
// and items with RunCost=0 (pure prerequisites) are preserved.
//
// Prerequisites of a kept item also stay, even if they'd normally be
// below the budget line.
//
// This is a post-optimization operation: the optimizer decides WHAT
// is valuable, the budget decides HOW MUCH of that value fits. If
// budget is zero or negative, or the plan already fits, the result
// is unchanged.
//
// Dropped items are moved to Skipped so explain/JSON output shows
// them with their original cost metadata — they remain debuggable.
// Schedule, WallClock, and TotalCPU are recomputed over the kept
// items.
func TrimToBudget(result *Result, budget time.Duration, maxParallelism int) {
	if result == nil || budget <= 0 || time.Duration(result.WallClock*float64(time.Second)) <= budget {
		return
	}

	// Build the protected set: force-run items + any item whose id is
	// reachable as a prerequisite of a force-run item.
	protected := make(map[string]bool)
	byID := make(map[string]*ExecutionItem, len(result.Items))
	for i := range result.Items {
		byID[result.Items[i].ID] = &result.Items[i]
		if isForceRun(result.Items[i]) || result.Items[i].RunCost == 0 {
			protected[result.Items[i].ID] = true
		}
	}
	// Transitively protect prerequisites of protected items.
	changed := true
	for changed {
		changed = false
		for id := range protected {
			item, ok := byID[id]
			if !ok {
				continue
			}
			for _, p := range item.Prerequisites {
				if !protected[p] {
					protected[p] = true
					changed = true
				}
			}
		}
	}

	// Sort non-protected items by value density ascending (lowest first
	// — these are the drop candidates). Ties broken by RunCost
	// descending (drop the expensive one first).
	type scored struct {
		idx     int
		density float64
		runCost float64
	}
	scores := make([]scored, 0, len(result.Items))
	for i, item := range result.Items {
		if protected[item.ID] {
			continue
		}
		density := 0.0
		if item.RunCost > 0 {
			density = item.SkipCost / item.RunCost
		}
		scores = append(scores, scored{idx: i, density: density, runCost: item.RunCost})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].density != scores[j].density {
			return scores[i].density < scores[j].density
		}
		return scores[i].runCost > scores[j].runCost
	})

	budgetSecs := budget.Seconds()
	drop := make(map[int]bool)

	for _, s := range scores {
		// Recompute schedule over remaining items to check fit.
		remaining := make([]ExecutionItem, 0, len(result.Items))
		for i, item := range result.Items {
			if drop[i] {
				continue
			}
			remaining = append(remaining, item)
		}
		sched := ComputeSchedule(remaining, maxParallelism)
		if sched.WallClock <= budgetSecs {
			break
		}
		drop[s.idx] = true
	}

	if len(drop) == 0 {
		return
	}

	kept := make([]ExecutionItem, 0, len(result.Items)-len(drop))
	for i, item := range result.Items {
		if drop[i] {
			result.Skipped = append(result.Skipped, item)
			continue
		}
		kept = append(kept, item)
	}
	result.Items = kept

	result.Schedule = ComputeSchedule(result.Items, maxParallelism)
	result.WallClock = result.Schedule.WallClock
	result.TotalCPU = result.Schedule.TotalCPU
}

// isForceRun reports whether the optimizer classified this item as
// high-signal enough to be non-negotiable (SkipCost set to cost + 1
// pattern from itemsForGroup / scopeableAllItem).
func isForceRun(item ExecutionItem) bool {
	return item.SkipCost > item.RunCost && item.SkipCost-item.RunCost <= 1.5
}
