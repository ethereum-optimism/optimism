package selector

import (
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/scorer"
)

// Selection represents a check that should run.
type Selection struct {
	CheckID       string
	PFail         float64
	RunCost       float64
	SkipCost      float64 // P(fail) × miss_cost
	Explanation   string
	Prerequisites []string
}

// Result is the output of the selection process.
type Result struct {
	Stage      string
	Selections []Selection // ordered by priority (highest skip cost first)
	Skipped    []Selection // checks that didn't meet the threshold
	TotalTime  float64     // estimated total run time (seconds)
}

// Select applies the expected-cost model:
// For each scored check, compute skip_cost = P(fail) × stage.MissCost.
// Include the check if skip_cost > run_cost (normalized).
// Resolve prerequisites: if check X is selected and depends on Y, include Y.
// Sort by skip_cost descending.
func Select(scores []scorer.Score, stage Stage, g *graph.Graph) *Result {
	result := &Result{Stage: stage.Name}

	// Score normalization: convert run_cost seconds to same scale as miss_cost.
	// miss_cost is in abstract "pain" units. We need a conversion factor.
	// Heuristic: 1 miss unit = 60 seconds of engineer time wasted.
	const secondsPerMissUnit = 60.0

	// Compute skip cost and partition
	type candidate struct {
		score    scorer.Score
		skipCost float64
		runCost  float64
		selected bool
	}

	candidates := make(map[string]*candidate)
	for _, sc := range scores {
		skipCost := sc.PFail * stage.MissCost
		normalizedRunCost := sc.RunCost / secondsPerMissUnit

		c := &candidate{
			score:    sc,
			skipCost: skipCost,
			runCost:  normalizedRunCost,
			selected: skipCost > normalizedRunCost,
		}
		candidates[sc.CheckID] = c
	}

	// Resolve prerequisites: if a check is selected, its prerequisites must also be selected
	changed := true
	for changed {
		changed = false
		for _, c := range candidates {
			if !c.selected {
				continue
			}
			prereqs := graph.Prerequisites(g, c.score.CheckID)
			for _, pid := range prereqs {
				if pc, ok := candidates[pid]; ok && !pc.selected {
					pc.selected = true
					changed = true
				}
			}
		}
	}

	// Also add prerequisites that weren't in the scored set
	for _, c := range candidates {
		if !c.selected {
			continue
		}
		prereqs := graph.Prerequisites(g, c.score.CheckID)
		for _, pid := range prereqs {
			if _, ok := candidates[pid]; !ok {
				// Prerequisite wasn't scored — add it as selected with 0 cost
				node := g.GetNode(pid)
				if node == nil {
					continue
				}
				dur, _ := node.Properties["avg_duration"].(float64)
				candidates[pid] = &candidate{
					score: scorer.Score{
						CheckID:     pid,
						PFail:       0,
						RunCost:     dur,
						Explanation: "prerequisite (auto-included)",
					},
					skipCost: 0,
					runCost:  dur / secondsPerMissUnit,
					selected: true,
				}
			}
		}
	}

	// Build result
	for _, c := range candidates {
		prereqs := graph.Prerequisites(g, c.score.CheckID)
		sel := Selection{
			CheckID:       c.score.CheckID,
			PFail:         c.score.PFail,
			RunCost:       c.score.RunCost,
			SkipCost:      c.skipCost,
			Explanation:   c.score.Explanation,
			Prerequisites: prereqs,
		}
		if c.selected {
			result.Selections = append(result.Selections, sel)
			result.TotalTime += c.score.RunCost
		} else {
			result.Skipped = append(result.Skipped, sel)
		}
	}

	// Sort selections by skip cost descending
	sort.Slice(result.Selections, func(i, j int) bool {
		return result.Selections[i].SkipCost > result.Selections[j].SkipCost
	})
	sort.Slice(result.Skipped, func(i, j int) bool {
		return result.Skipped[i].SkipCost > result.Skipped[j].SkipCost
	})

	return result
}
