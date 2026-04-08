package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/scorer"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ExitOnError)
	stageName := fs.String("stage", "commit", "development stage (save, commit, pr, merge_queue, develop)")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	diffFile := fs.String("diff", "", "path to diff file (default: stdin)")
	format := fs.String("format", "text", "output format (text, json)")
	fs.Parse(args)

	stage, err := selector.StageByName(*stageName)
	if err != nil {
		return err
	}

	g, err := graph.Load(*graphPath)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Read diff
	var diffInput io.Reader
	if *diffFile != "" {
		f, err := os.Open(*diffFile)
		if err != nil {
			return fmt.Errorf("opening diff file: %w", err)
		}
		defer f.Close()
		diffInput = f
	} else {
		diffInput = os.Stdin
	}

	data, err := io.ReadAll(diffInput)
	if err != nil {
		return fmt.Errorf("reading diff: %w", err)
	}

	files := diff.ChangedFiles(string(data))
	if len(files) == 0 {
		fmt.Println("No changed files.")
		return nil
	}

	// Check blast radius
	if blast, matches := diff.BlastRadiusFiles(files); blast {
		fmt.Printf("⚠ Blast radius files changed: %s\n", strings.Join(matches, ", "))
		fmt.Println("All checks recommended.")
	}

	// Map files to node IDs
	nodeIDs, unknown := diff.FilesToNodeIDs(g, files)
	if len(unknown) > 0 {
		fmt.Fprintf(os.Stderr, "Unmapped files: %s\n", strings.Join(unknown, ", "))
	}

	if len(nodeIDs) == 0 {
		fmt.Println("No graph nodes matched the changed files.")
		return nil
	}

	// Walk graph to find reachable checks
	reachable := graph.ReachableChecks(g, nodeIDs, 0.01)

	// Score
	s := scorer.NewSimple()
	scores, err := s.Score(g, files, reachable)
	if err != nil {
		return fmt.Errorf("scoring: %w", err)
	}

	// Select
	result := selector.Select(scores, stage, g)

	// Output
	if *format == "json" {
		return printResultJSON(result)
	}
	return printResultText(result)
}

func printResultText(result *selector.Result) error {
	fmt.Printf("Stage: %s\n\n", result.Stage)

	if len(result.Selections) == 0 {
		fmt.Println("No checks selected.")
		return nil
	}

	fmt.Printf("Selected checks (%d, est. %.0fs):\n", len(result.Selections), result.TotalTime)
	for i, sel := range result.Selections {
		fmt.Printf("  %d. %s  P(fail)=%.3f  cost=%.0fs  skip_cost=%.2f\n",
			i+1, sel.CheckID, sel.PFail, sel.RunCost, sel.SkipCost)
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("\nSkipped checks (%d):\n", len(result.Skipped))
		for _, sel := range result.Skipped {
			fmt.Printf("  - %s  P(fail)=%.3f  cost=%.0fs\n",
				sel.CheckID, sel.PFail, sel.RunCost)
		}
	}

	return nil
}

func printResultJSON(result *selector.Result) error {
	// Simple JSON output
	fmt.Println("{")
	fmt.Printf("  \"stage\": %q,\n", result.Stage)
	fmt.Printf("  \"total_time\": %.0f,\n", result.TotalTime)
	fmt.Println("  \"selected\": [")
	for i, sel := range result.Selections {
		comma := ","
		if i == len(result.Selections)-1 {
			comma = ""
		}
		fmt.Printf("    {\"check\": %q, \"p_fail\": %.3f, \"run_cost\": %.0f, \"skip_cost\": %.2f}%s\n",
			sel.CheckID, sel.PFail, sel.RunCost, sel.SkipCost, comma)
	}
	fmt.Println("  ],")
	fmt.Println("  \"skipped\": [")
	for i, sel := range result.Skipped {
		comma := ","
		if i == len(result.Skipped)-1 {
			comma = ""
		}
		fmt.Printf("    {\"check\": %q, \"p_fail\": %.3f, \"run_cost\": %.0f}%s\n",
			sel.CheckID, sel.PFail, sel.RunCost, comma)
	}
	fmt.Println("  ]")
	fmt.Println("}")
	return nil
}
