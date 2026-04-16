package selector

import (
	"fmt"
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// SimpleOptimizer turns Candidates into ExecutionItems using a fixed
// tier/cost model.
//
// For each (CheckID, Profile) group of Candidates:
//   - Candidates are bucketed into signal tiers (high/med/low).
//   - Each tier becomes one ExecutionItem whose Scope is the list of
//     scopes in that tier. Binary candidates (Scope="") become one
//     unscoped ExecutionItem per check.
//   - Run/skip decision: keep the item if its SkipCost exceeds its
//     RunCost, where SkipCost = signal × kindPrior × stage.MissCost.
//
// Prerequisites are then resolved transitively and the plan is ordered
// into a parallel schedule.
type SimpleOptimizer struct{}

// NewSimpleOptimizer creates a SimpleOptimizer.
func NewSimpleOptimizer() *SimpleOptimizer { return &SimpleOptimizer{} }

// Optimize implements Optimizer.
func (o *SimpleOptimizer) Optimize(
	candidates []Candidate,
	stage Stage,
	cat *catalog.Catalog,
) (*Result, error) {
	result := &Result{Stage: stage.Name}
	if len(candidates) == 0 {
		return result, nil
	}

	type groupKey struct {
		checkID string
		profile string
	}
	groups := make(map[groupKey][]Candidate)
	for _, c := range candidates {
		k := groupKey{c.CheckID, c.Profile}
		groups[k] = append(groups[k], c)
	}

	for k, cands := range groups {
		ct := cat.ByID(k.checkID)
		if ct == nil {
			continue
		}
		items := o.itemsForGroup(ct, k.profile, cands, stage)
		for _, item := range items {
			if item.SkipCost > item.RunCost {
				result.Items = append(result.Items, item)
			} else {
				result.Skipped = append(result.Skipped, item)
			}
		}
	}

	o.resolvePrerequisites(result, cat)

	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].SkipCost > result.Items[j].SkipCost
	})

	result.Schedule = ComputeSchedule(result.Items, 0)
	result.WallClock = result.Schedule.WallClock
	result.TotalCPU = result.Schedule.TotalCPU

	return result, nil
}

// itemsForGroup produces ExecutionItems for all Candidates sharing a
// (CheckID, Profile). Scopeable candidates get tiered; binary candidates
// become a single unscoped item.
func (o *SimpleOptimizer) itemsForGroup(
	ct *catalog.CheckType,
	profile string,
	cands []Candidate,
	stage Stage,
) []ExecutionItem {
	// Binary check, or an unscoped (blast-radius / trigger) candidate.
	// There should be at most one of these per group.
	if !ct.Scopeable || (len(cands) == 1 && cands[0].Scope == "") {
		c := cands[0]
		if !ct.Scopeable {
			return []ExecutionItem{o.binaryItem(ct, c, stage)}
		}
		// Scopeable but unscoped: "run everything" path (blast radius or trigger).
		return []ExecutionItem{o.scopeableAllItem(ct, c, profile, stage)}
	}

	tiers := []struct {
		minSignal float64
		label     string
	}{
		{0.6, "high"},
		{0.3, "med"},
		{0.1, "low"},
	}

	used := make(map[string]bool)
	var items []ExecutionItem
	for _, t := range tiers {
		var tierScopes []string
		var maxSignal float64
		for _, c := range cands {
			if c.Scope == "" || used[c.Scope] {
				continue
			}
			if c.Signal >= t.minSignal {
				tierScopes = append(tierScopes, c.Scope)
				used[c.Scope] = true
				if c.Signal > maxSignal {
					maxSignal = c.Signal
				}
			}
		}
		if len(tierScopes) == 0 {
			continue
		}

		config := mapSignalToConfig(ct, maxSignal, stage)
		cost := estimateScopedRunCost(ct, len(tierScopes), config)

		prior := checkTypePrior(ct.Kind)
		if maxSignal > 0.6 {
			prior = 0.7
		}
		pFail := maxSignal * prior
		skipCost := pFail * stage.MissCost
		if maxSignal > 0.6 {
			skipCost = cost + 1
		}

		idSuffix := t.label
		if profile != "" {
			idSuffix = profile + ":" + t.label
		}

		items = append(items, ExecutionItem{
			ID:            fmt.Sprintf("%s:%s", ct.ID, idSuffix),
			CheckTypeID:   ct.ID,
			Scope:         tierScopes,
			Config:        config,
			Profile:       profile,
			Signal:        maxSignal,
			RunCost:       cost,
			SkipCost:      skipCost,
			Prerequisites: prereqItemIDs(ct),
		})
	}
	return items
}

func (o *SimpleOptimizer) binaryItem(ct *catalog.CheckType, c Candidate, stage Stage) ExecutionItem {
	prior := checkTypePrior(ct.Kind)
	pFail := prior * c.Signal
	return ExecutionItem{
		ID:            ct.ID,
		CheckTypeID:   ct.ID,
		Signal:        c.Signal,
		RunCost:       float64(ct.AvgDuration),
		SkipCost:      pFail * stage.MissCost,
		Prerequisites: prereqItemIDs(ct),
	}
}

func (o *SimpleOptimizer) scopeableAllItem(
	ct *catalog.CheckType,
	c Candidate,
	profile string,
	stage Stage,
) ExecutionItem {
	config := mapSignalToConfig(ct, 1.0, stage)
	cost := float64(ct.AvgDuration)
	id := ct.ID + ":all"
	if profile != "" {
		id = ct.ID + ":" + profile + ":all"
	}
	return ExecutionItem{
		ID:            id,
		CheckTypeID:   ct.ID,
		Scope:         nil,
		Config:        config,
		Profile:       profile,
		Signal:        1.0,
		RunCost:       cost,
		SkipCost:      cost + 1,
		Prerequisites: prereqItemIDs(ct),
	}
}

// resolvePrerequisites ensures prerequisite check types are included
// as items, even if they weren't candidates themselves. Prerequisites
// run ordering comes from ComputeSchedule.
func (o *SimpleOptimizer) resolvePrerequisites(result *Result, cat *catalog.Catalog) {
	selected := make(map[string]bool)
	for _, item := range result.Items {
		selected[item.ID] = true
	}

	changed := true
	for changed {
		changed = false
		for _, item := range result.Items {
			for _, prereqID := range item.Prerequisites {
				if selected[prereqID] {
					continue
				}
				ct := cat.ByID(prereqID)
				if ct == nil {
					continue
				}
				result.Items = append(result.Items, ExecutionItem{
					ID:          prereqID,
					CheckTypeID: prereqID,
					Signal:      0,
					RunCost:     float64(ct.AvgDuration),
					SkipCost:    0,
				})
				selected[prereqID] = true
				changed = true
			}
		}
	}
}

// --- policy-ish helpers (will move to a policy module in tweak 2) ---

func mapSignalToConfig(ct *catalog.CheckType, signal float64, stage Stage) map[string]any {
	config := make(map[string]any)
	for _, knob := range ct.Knobs {
		switch knob.Name {
		case "fuzz_runs":
			config[knob.Name] = fuzzRunsForSignalStage(signal, stage)
		case "short":
			config[knob.Name] = signal < 0.8
		case "race":
			config[knob.Name] = signal > 0.8 && stage.MissCost >= StageOnPR.MissCost
		case "timeout":
			if signal > 0.8 {
				config[knob.Name] = "30m"
			} else {
				config[knob.Name] = "10m"
			}
		case "count":
			config[knob.Name] = 1
		default:
			config[knob.Name] = knob.Default
		}
	}
	return config
}

func fuzzRunsForSignalStage(signal float64, stage Stage) int {
	var high, med, low int
	switch {
	case stage.MissCost >= StageDevelop.MissCost:
		high, med, low = 20000, 1024, 128
	case stage.MissCost >= StageMergeQueue.MissCost:
		high, med, low = 1024, 128, 64
	case stage.MissCost >= StageOnPR.MissCost:
		high, med, low = 128, 64, 8
	case stage.MissCost >= StageOnCommit.MissCost:
		high, med, low = 64, 8, 1
	default:
		high, med, low = 1, 1, 1
	}

	switch {
	case signal > 0.6:
		return high
	case signal > 0.3:
		return med
	default:
		return low
	}
}

func checkTypePrior(kind string) float64 {
	switch kind {
	case "lint":
		return 0.8
	case "build":
		return 0.5
	case "test":
		return 0.3
	case "check":
		return 0.4
	default:
		return 0.3
	}
}

func estimateScopedRunCost(ct *catalog.CheckType, numScopes int, config map[string]any) float64 {
	if ct.PerUnitDuration <= 0 {
		return float64(ct.AvgDuration)
	}

	baseCost := float64(ct.PerUnitDuration)
	marginalCost := baseCost * 0.3
	if numScopes > 1 {
		baseCost += marginalCost * float64(numScopes-1)
	}

	if fuzzRuns, ok := config["fuzz_runs"]; ok {
		var fuzz float64
		switch v := fuzzRuns.(type) {
		case int:
			fuzz = float64(v)
		case float64:
			fuzz = v
		}
		if fuzz > 8 {
			fuzzMultiplier := 1.0 + 0.7*log2(fuzz/8)
			baseCost *= fuzzMultiplier
		}
	}

	return baseCost
}

func log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	result := 0.0
	for x >= 2 {
		x /= 2
		result++
	}
	if x > 1 {
		result += x - 1
	}
	return result
}

func prereqItemIDs(ct *catalog.CheckType) []string {
	return ct.Prerequisites
}
