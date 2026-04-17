package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/cihistory"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/replay"
)

func cmdReplay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	eventsPath := fs.String("events", "", "path to events JSON (same format as `ingest ci-history --events`)")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	policyPath := fs.String("policy", "", "optional policy override YAML")
	stageName := fs.String("stage", "pr", "stage to replay against (save/commit/pr/merge_queue/develop)")
	format := fs.String("format", "text", "output format (text, json)")
	showMisses := fs.Bool("show-misses", false, "print per-event details for every missed failure")
	fs.Parse(args)

	if *eventsPath == "" {
		return fmt.Errorf("--events is required")
	}

	resolvedGraph := resolveFromRoot(*graphPath)
	resolvedCatalog := resolveFromRoot(*catalogPath)

	pol, err := loadPolicy(*policyPath)
	if err != nil {
		return err
	}

	events, err := cihistory.LoadEvents(*eventsPath)
	if err != nil {
		return fmt.Errorf("loading events: %w", err)
	}

	warnIfGraphStale(resolvedGraph, findRepoRoot())
	g, err := graph.Load(resolvedGraph)
	if err != nil {
		return missingGraphError(resolvedGraph, err)
	}
	cat, err := catalog.Load(resolvedCatalog)
	if err != nil {
		return fmt.Errorf("loading catalog %s: %w", resolvedCatalog, err)
	}

	results, summary, err := replay.Run(events, g, cat, pol, *stageName)
	if err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	if *format == "json" {
		return printReplayJSON(summary, results)
	}

	fmt.Print(replay.FormatSummary(summary))

	if *showMisses && summary.MissedFailures > 0 {
		fmt.Println("\nMissed-failure events:")
		for _, r := range results {
			if len(r.MissedFailures) == 0 {
				continue
			}
			fmt.Printf("  PR #%d (%d files)\n", r.PR, r.ChangedFiles)
			fmt.Printf("    actually failed: %v\n", r.ActuallyFailed)
			fmt.Printf("    selector picked: %v\n", r.SelectorPicked)
			fmt.Printf("    MISSED: %v\n", r.MissedFailures)
		}
	}

	// Exit nonzero if any failures were missed — makes this easy to
	// use as a gate in CI ("is recall above threshold?").
	if summary.MissedFailures > 0 {
		return fmt.Errorf("%d failures were not covered by selector (recall %.2f%%)",
			summary.MissedFailures, summary.FailureRecall*100)
	}
	return nil
}

type replayJSON struct {
	Summary *replay.Summary  `json:"summary"`
	Results []replay.Result  `json:"results"`
}

func printReplayJSON(summary *replay.Summary, results []replay.Result) error {
	out := replayJSON{Summary: summary, Results: results}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
