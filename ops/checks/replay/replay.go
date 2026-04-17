// Package replay answers the only question that actually matters
// for the selector: if we had been driving CI with this selector,
// would we have missed any real failures?
//
// Given a set of historical CI events (same shape as
// cihistory.Event: a merged PR's file list + per-check pass/fail),
// the harness replays the selection for each event and compares:
//
//   - Failures the selector would have caught: a check failed in
//     CI, and the selector's plan included it.
//   - Failures the selector would have missed: a check failed in
//     CI, and the selector's plan did NOT include it. THIS IS THE
//     CRITICAL METRIC — each one represents a bug that would have
//     shipped if the selector were driving CI.
//   - Over-runs: checks the selector picked that actually passed in
//     CI. Wasted time, not correctness loss, but the core value
//     proposition of the selector.
//
// Replay is deliberately simple: it does not attempt to be a
// CI-scheduler simulator. It does not model parallelism, caching,
// or retry behavior. The question it answers is recall on failures.
package replay

import (
	"fmt"
	"sort"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/cihistory"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

// Result is the per-event comparison between what CI actually ran /
// failed and what the selector would have done.
type Result struct {
	PR           int      `json:"pr"`
	ChangedFiles int      `json:"changed_files"`

	ActuallyRan    []string `json:"actually_ran"`
	ActuallyFailed []string `json:"actually_failed"`
	SelectorPicked []string `json:"selector_picked"`

	CaughtFailures []string `json:"caught_failures"` // in ActuallyFailed ∩ SelectorPicked
	MissedFailures []string `json:"missed_failures"` // in ActuallyFailed \ SelectorPicked (the critical metric)
	OverRuns       []string `json:"over_runs"`       // in SelectorPicked \ ActuallyFailed (passed in CI)
	UnrunSkips     []string `json:"unrun_skips"`     // in SelectorPicked \ ActuallyRan (CI never even ran it)
}

// Summary aggregates Results across a replay run.
type Summary struct {
	TotalEvents int `json:"total_events"`

	TotalFailures  int `json:"total_failures"`
	CaughtFailures int `json:"caught_failures"`
	MissedFailures int `json:"missed_failures"`

	TotalSelectorPicks int `json:"total_selector_picks"`
	OverRuns           int `json:"over_runs"`

	FailureRecall float64 `json:"failure_recall"` // CaughtFailures / TotalFailures
	OverRunRate   float64 `json:"over_run_rate"`  // OverRuns / TotalSelectorPicks

	// Per-check counts, sorted descending by miss count.
	PerCheckMissed map[string]int `json:"per_check_missed"`
	PerCheckCaught map[string]int `json:"per_check_caught"`
}

// Run replays the selector against each event and returns both the
// per-event results and an aggregate summary.
//
// stage is passed to the optimizer — "pr" is the typical choice
// since events typically correspond to merged PRs.
func Run(
	events []cihistory.Event,
	g *graph.Graph,
	cat *catalog.Catalog,
	pol *policy.Policy,
	stage string,
) ([]Result, *Summary, error) {
	stageCfg, err := pol.Stage(stage)
	if err != nil {
		return nil, nil, err
	}
	optimizer := selector.NewSimpleOptimizer(pol)

	summary := &Summary{
		TotalEvents:    len(events),
		PerCheckMissed: make(map[string]int),
		PerCheckCaught: make(map[string]int),
	}

	results := make([]Result, 0, len(events))
	for _, e := range events {
		r := replayOne(e, g, cat, pol, stageCfg, optimizer)
		results = append(results, r)

		summary.TotalFailures += len(r.ActuallyFailed)
		summary.CaughtFailures += len(r.CaughtFailures)
		summary.MissedFailures += len(r.MissedFailures)
		summary.TotalSelectorPicks += len(r.SelectorPicked)
		summary.OverRuns += len(r.OverRuns)

		for _, id := range r.CaughtFailures {
			summary.PerCheckCaught[id]++
		}
		for _, id := range r.MissedFailures {
			summary.PerCheckMissed[id]++
		}
	}

	if summary.TotalFailures > 0 {
		summary.FailureRecall = float64(summary.CaughtFailures) / float64(summary.TotalFailures)
	} else {
		summary.FailureRecall = 1.0 // vacuously
	}
	if summary.TotalSelectorPicks > 0 {
		summary.OverRunRate = float64(summary.OverRuns) / float64(summary.TotalSelectorPicks)
	}

	return results, summary, nil
}

// replayOne runs the selector on one event and classifies every
// check across four buckets.
func replayOne(
	e cihistory.Event,
	g *graph.Graph,
	cat *catalog.Catalog,
	pol *policy.Policy,
	stage policy.StageConfig,
	optimizer *selector.SimpleOptimizer,
) Result {
	// Reconstruct a bare FileDiff set. We don't have hunks from
	// cihistory events — the selector falls back to file-level match
	// for coverage edges when hunks are absent, which is the right
	// conservative behavior here.
	diffs := make([]diff.FileDiff, 0, len(e.Files))
	for _, f := range e.Files {
		diffs = append(diffs, diff.FileDiff{Path: f})
	}

	candidates := selector.Resolve(g, diffs, cat, pol, freshness.Nop())
	result, err := optimizer.Optimize(candidates, stage, cat)

	picked := make(map[string]bool)
	if err == nil {
		for _, item := range result.Items {
			picked[item.CheckTypeID] = true
		}
	}

	ran := make(map[string]bool)
	failed := make(map[string]bool)
	for _, c := range e.Checks {
		ran[c.ID] = true
		if c.Failed {
			failed[c.ID] = true
		}
	}

	r := Result{PR: e.PR, ChangedFiles: len(e.Files)}
	r.ActuallyRan = keys(ran)
	r.ActuallyFailed = keys(failed)
	r.SelectorPicked = keys(picked)

	for id := range failed {
		if picked[id] {
			r.CaughtFailures = append(r.CaughtFailures, id)
		} else {
			r.MissedFailures = append(r.MissedFailures, id)
		}
	}
	for id := range picked {
		if !failed[id] && ran[id] {
			r.OverRuns = append(r.OverRuns, id)
		}
		if !ran[id] {
			r.UnrunSkips = append(r.UnrunSkips, id)
		}
	}

	sort.Strings(r.ActuallyRan)
	sort.Strings(r.ActuallyFailed)
	sort.Strings(r.SelectorPicked)
	sort.Strings(r.CaughtFailures)
	sort.Strings(r.MissedFailures)
	sort.Strings(r.OverRuns)
	sort.Strings(r.UnrunSkips)
	return r
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// FormatSummary renders a human-readable summary suitable for
// `checks replay` stdout. JSON output is done separately by the
// caller via encoding/json on the Summary struct.
func FormatSummary(s *Summary) string {
	out := fmt.Sprintf(`Replay summary
  Events replayed:       %d
  Total failures in CI:  %d
  Caught by selector:    %d
  Missed by selector:    %d
  Failure recall:        %.2f%%

  Selector picks total:  %d
  Over-runs (passed CI): %d
  Over-run rate:         %.2f%%
`,
		s.TotalEvents, s.TotalFailures, s.CaughtFailures, s.MissedFailures, s.FailureRecall*100,
		s.TotalSelectorPicks, s.OverRuns, s.OverRunRate*100,
	)

	if s.MissedFailures > 0 {
		out += "\n  Missed failures by check (higher = worse):\n"
		type pair struct {
			id    string
			count int
		}
		rows := make([]pair, 0, len(s.PerCheckMissed))
		for id, c := range s.PerCheckMissed {
			rows = append(rows, pair{id, c})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
		for _, r := range rows {
			out += fmt.Sprintf("    %-30s %d\n", r.id, r.count)
		}
	}

	return out
}
