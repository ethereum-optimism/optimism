package selector

import (
	"fmt"
	"math"
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// SimpleOptimizer turns Candidates into ExecutionItems using policy-
// driven tier/cost/knob tables.
//
// For each (CheckID, Profile) group of Candidates:
//   - Candidates are bucketed into signal tiers (from policy.Tiers).
//   - Each tier becomes one ExecutionItem whose Scope is the list of
//     scopes in that tier. Binary candidates (Scope="") become one
//     unscoped ExecutionItem per check.
//   - Run/skip decision: keep the item if SkipCost > RunCost, where
//     SkipCost = signal × kindPrior × stage.MissCost. At signals above
//     policy.HighSignal.Threshold, the prior is overridden and the
//     item is force-run.
//
// Prerequisites are resolved transitively and the plan is ordered into
// a parallel schedule bounded by policy.MaxParallelism.
type SimpleOptimizer struct {
	policy *policy.Policy
}

// NewSimpleOptimizer creates a SimpleOptimizer bound to a policy.
func NewSimpleOptimizer(p *policy.Policy) *SimpleOptimizer {
	return &SimpleOptimizer{policy: p}
}

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

	result.Schedule = ComputeSchedule(result.Items, o.policy.MaxParallelism())
	result.WallClock = result.Schedule.WallClock
	result.TotalCPU = result.Schedule.TotalCPU

	return result, nil
}

// itemsForGroup produces ExecutionItems for all Candidates sharing a
// (CheckID, Profile). Scopeable candidates get tiered per policy;
// binary candidates become one unscoped item.
func (o *SimpleOptimizer) itemsForGroup(
	ct *catalog.CheckType,
	profile string,
	cands []Candidate,
	stage Stage,
) []ExecutionItem {
	if !ct.Scopeable || (len(cands) == 1 && cands[0].Scope == "") {
		c := cands[0]
		if !ct.Scopeable {
			return []ExecutionItem{o.binaryItem(ct, c, stage)}
		}
		// Scopeable but unscoped: blast radius / trigger "run everything".
		return []ExecutionItem{o.scopeableAllItem(ct, c, profile, stage)}
	}

	used := make(map[string]bool)
	var items []ExecutionItem
	for _, tier := range o.policy.Tiers {
		var tierScopes []string
		var maxSignal float64
		var tierProvenance []SignalContribution
		for _, c := range cands {
			if c.Scope == "" || used[c.Scope] {
				continue
			}
			if c.Signal >= tier.MinSignal {
				tierScopes = append(tierScopes, c.Scope)
				used[c.Scope] = true
				if c.Signal > maxSignal {
					maxSignal = c.Signal
				}
				tierProvenance = append(tierProvenance, c.Provenance...)
			}
		}
		if len(tierScopes) == 0 {
			continue
		}

		config := o.policy.KnobConfig(ct, stage.Name, tier.Label)
		cost := o.estimateScopedRunCost(ct, len(tierScopes), config)
		skipCost := o.skipCostFor(ct, maxSignal, stage, cost)

		idSuffix := tier.Label
		if profile != "" {
			idSuffix = profile + ":" + tier.Label
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
			Prerequisites: ct.Prerequisites,
			Provenance:    tierProvenance,
		})
	}
	return items
}

func (o *SimpleOptimizer) binaryItem(ct *catalog.CheckType, c Candidate, stage Stage) ExecutionItem {
	prior := o.policy.Prior(ct.Kind)
	pFail := prior * c.Signal
	return ExecutionItem{
		ID:            ct.ID,
		CheckTypeID:   ct.ID,
		Signal:        c.Signal,
		RunCost:       float64(ct.AvgDuration),
		SkipCost:      pFail * stage.MissCost,
		Prerequisites: ct.Prerequisites,
		Provenance:    c.Provenance,
	}
}

func (o *SimpleOptimizer) scopeableAllItem(
	ct *catalog.CheckType,
	c Candidate,
	profile string,
	stage Stage,
) ExecutionItem {
	tierLabel := ""
	if t := o.policy.TierFor(1.0); t != nil {
		tierLabel = t.Label
	}
	config := o.policy.KnobConfig(ct, stage.Name, tierLabel)
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
		Prerequisites: ct.Prerequisites,
		Provenance:    c.Provenance,
	}
}

// skipCostFor computes the expected regret of skipping this item.
// Above the high-signal threshold, the item is force-run by returning
// a skip cost that exceeds RunCost by a hair.
func (o *SimpleOptimizer) skipCostFor(
	ct *catalog.CheckType,
	signal float64,
	stage Stage,
	runCost float64,
) float64 {
	if signal > o.policy.HighSignal.Threshold {
		return runCost + 1
	}
	prior := o.policy.Prior(ct.Kind)
	return signal * prior * stage.MissCost
}

// resolvePrerequisites ensures prerequisite check types are included
// as items, even if they weren't candidates themselves.
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

// estimateScopedRunCost scales the per-unit duration by the number of
// scopes and fuzz_runs. Marginal cost per extra scope is 30% of base
// (amortized compile + loop overhead).
func (o *SimpleOptimizer) estimateScopedRunCost(
	ct *catalog.CheckType,
	numScopes int,
	config map[string]any,
) float64 {
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
			fuzzMultiplier := 1.0 + 0.7*math.Log2(fuzz/8)
			baseCost *= fuzzMultiplier
		}
	}

	return baseCost
}
