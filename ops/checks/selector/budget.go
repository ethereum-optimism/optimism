package selector

import (
	"sort"
	"time"
)

// TrimToBudget drops low-value items from a Result until the
// schedule fits under budget. "Low-value" is defined strictly:
// density < 1.0, i.e. SkipCost < RunCost — the optimizer's own
// "this is worth running" threshold inverted. Items above that
// threshold are never trimmed; the optimizer already said they're
// worth the cost. If the budget can't be met by dropping only
// low-value items, the plan is returned over budget — correctness
// and regret-minimization beat hard budget compliance.
//
// Three invariants, checked each iteration:
//
//   - Force-run items (SkipCost = RunCost + 1 pattern) are
//     permanently protected.
//   - Items where density ≥ 1 are never dropped. When the sorted
//     scan reaches one, trimming stops.
//   - Prerequisites of kept items are never dropped. `canDrop`
//     evaluates this dynamically, and an orphan sweep runs after
//     each drop so consumer+prereq pairs drop together when both
//     qualify.
//
// Dropped items move to Skipped — still visible to `select --why`
// and JSON output. Schedule, WallClock, TotalCPU are recomputed
// over the kept set.
func TrimToBudget(result *Result, budget time.Duration, maxParallelism int) {
	if result == nil || budget <= 0 || time.Duration(result.WallClock*float64(time.Second)) <= budget {
		return
	}

	byID := make(map[string]int, len(result.Items))
	for i := range result.Items {
		byID[result.Items[i].ID] = i
	}

	protected := make(map[int]bool)
	for i, item := range result.Items {
		if isForceRun(item) {
			protected[i] = true
		}
	}

	drop := make(map[int]bool)

	// canDrop: idx is safe to drop iff no currently-kept item has it
	// as a prerequisite. Evaluated dynamically so that dropping a
	// consumer enables later dropping its now-orphaned prereqs.
	canDrop := func(idx int) bool {
		if protected[idx] {
			return false
		}
		id := result.Items[idx].ID
		for i, item := range result.Items {
			if i == idx || drop[i] {
				continue
			}
			for _, p := range item.Prerequisites {
				if p == id {
					return false
				}
			}
		}
		return true
	}

	// sweepOrphanedPrereqs: drop any pure-prereq item (SkipCost=0)
	// whose consumers are all now dropped. Iterates until stable so
	// a chain of prereqs all go together.
	sweepOrphanedPrereqs := func() {
		for {
			changed := false
			for i, item := range result.Items {
				if drop[i] || protected[i] {
					continue
				}
				if item.SkipCost != 0 {
					continue
				}
				if canDrop(i) {
					drop[i] = true
					changed = true
				}
			}
			if !changed {
				break
			}
		}
	}

	// Sort non-protected, non-zero-runcost items by value density
	// ascending (cheapest-per-second-of-regret drops first). Ties
	// broken by RunCost descending (drop the more expensive).
	type scored struct {
		idx     int
		density float64
		runCost float64
	}
	scores := make([]scored, 0, len(result.Items))
	for i, item := range result.Items {
		if protected[i] {
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
	for _, s := range scores {
		if s.density >= 1.0 {
			break // never drop items the optimizer committed to
		}
		if drop[s.idx] {
			continue // already dropped by an earlier sweep
		}
		if !canDrop(s.idx) {
			continue // would orphan a consumer
		}

		drop[s.idx] = true
		sweepOrphanedPrereqs()

		remaining := remainingItems(result.Items, drop)
		sched := ComputeSchedule(remaining, maxParallelism)
		if sched.WallClock <= budgetSecs {
			break
		}
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
	_ = byID
}

func remainingItems(items []ExecutionItem, drop map[int]bool) []ExecutionItem {
	out := make([]ExecutionItem, 0, len(items))
	for i, item := range items {
		if drop[i] {
			continue
		}
		out = append(out, item)
	}
	return out
}

// isForceRun reports whether the optimizer classified this item as
// high-signal enough to be non-negotiable (SkipCost set to cost + 1
// pattern from itemsForGroup / scopeableAllItem).
func isForceRun(item ExecutionItem) bool {
	return item.SkipCost > item.RunCost && item.SkipCost-item.RunCost <= 1.5
}
