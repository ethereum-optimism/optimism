package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ExitOnError)
	stageName := fs.String("stage", "commit", "development stage (save, commit, pr, merge_queue, develop)")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
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

	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

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

	diffs := diff.ParseUnifiedDiff(string(data))
	if len(diffs) == 0 {
		fmt.Println("No changed files.")
		return nil
	}

	// Phase 1: Reachability — find all candidates
	candidates := selector.FindCandidates(g, diffs, cat)

	// Phase 2: Optimization — produce execution plan
	optimizer := selector.NewSimpleOptimizer()
	result, err := optimizer.Optimize(g, candidates, diffs, stage, cat)
	if err != nil {
		return fmt.Errorf("optimizing: %w", err)
	}

	if *format == "json" {
		return printResultJSON(result, cat)
	}
	return printResultText(result, cat)
}

func printResultText(result *selector.Result, cat *catalog.Catalog) error {
	fmt.Printf("Stage: %s\n\n", result.Stage)

	if len(result.Items) == 0 {
		fmt.Println("No checks selected.")
		return nil
	}

	fmt.Printf("Execution plan (%d items):\n", len(result.Items))
	for i, layer := range result.Schedule.Layers {
		fmt.Printf("  Layer %d (%.0fs):\n", i+1, layer.Duration)
		for _, itemID := range layer.ItemIDs {
			for _, item := range result.Items {
				if item.ID == itemID {
					ct := cat.ByID(item.CheckTypeID)
					cmd := item.ResolvedCommand(ct)
					fmt.Printf("    %s  (signal=%.2f)\n", cmd, item.Signal)
				}
			}
		}
	}

	fmt.Printf("\nEstimated: %.0fs wall-clock, %.0fs CPU",
		result.WallClock, result.TotalCPU)
	if result.TotalCPU > 0 && result.WallClock > 0 {
		fmt.Printf(" (%.1fx speedup)", result.TotalCPU/result.WallClock)
	}
	fmt.Println()

	if len(result.Skipped) > 0 {
		fmt.Printf("\nSkipped (%d):\n", len(result.Skipped))
		for _, item := range result.Skipped {
			ct := cat.ByID(item.CheckTypeID)
			cmd := item.ResolvedCommand(ct)
			fmt.Printf("  - %s  (signal=%.2f, cost=%.0fs)\n", cmd, item.Signal, item.RunCost)
		}
	}

	return nil
}

func printResultJSON(result *selector.Result, cat *catalog.Catalog) error {
	fmt.Println("{")
	fmt.Printf("  \"stage\": %q,\n", result.Stage)
	fmt.Printf("  \"wall_clock\": %.0f,\n", result.WallClock)
	fmt.Printf("  \"total_cpu\": %.0f,\n", result.TotalCPU)
	fmt.Println("  \"items\": [")
	for i, item := range result.Items {
		ct := cat.ByID(item.CheckTypeID)
		cmd := item.ResolvedCommand(ct)
		comma := ","
		if i == len(result.Items)-1 {
			comma = ""
		}
		scope := "null"
		if len(item.Scope) > 0 {
			scope = fmt.Sprintf("[%q]", strings.Join(item.Scope, "\", \""))
		}
		fmt.Printf("    {\"id\": %q, \"command\": %q, \"scope\": %s, \"signal\": %.3f, \"run_cost\": %.0f}%s\n",
			item.ID, cmd, scope, item.Signal, item.RunCost, comma)
	}
	fmt.Println("  ],")
	fmt.Println("  \"skipped\": [")
	for i, item := range result.Skipped {
		ct := cat.ByID(item.CheckTypeID)
		cmd := item.ResolvedCommand(ct)
		comma := ","
		if i == len(result.Skipped)-1 {
			comma = ""
		}
		fmt.Printf("    {\"id\": %q, \"command\": %q, \"signal\": %.3f, \"run_cost\": %.0f}%s\n",
			item.ID, cmd, item.Signal, item.RunCost, comma)
	}
	fmt.Println("  ]")
	fmt.Println("}")
	return nil
}
