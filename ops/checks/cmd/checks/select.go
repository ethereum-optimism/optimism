package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdSelect(args []string) error {
	fs := flag.NewFlagSet("select", flag.ExitOnError)
	stageName := fs.String("stage", "commit", "development stage (save, commit, pr, merge_queue, develop)")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	policyPath := fs.String("policy", "", "optional policy override YAML (stacks on embedded baseline + learned.yaml)")
	diffFile := fs.String("diff", "", "path to diff file (default: stdin)")
	format := fs.String("format", "text", "output format (text, json)")
	fs.Parse(args)

	pol, err := loadPolicy(*policyPath)
	if err != nil {
		return err
	}
	stage, err := pol.Stage(*stageName)
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

	// Phase 1: Resolve — emit candidate items with per-source provenance.
	fresh := freshness.New(findRepoRoot(), pol)
	candidates := selector.Resolve(g, diffs, cat, pol, fresh)

	// Phase 2: Optimize — pure candidates → plan, no graph access.
	optimizer := selector.NewSimpleOptimizer(pol)
	result, err := optimizer.Optimize(candidates, stage, cat)
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
					cmd := item.ResolvedCommandWithCatalog(ct, cat)
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
			cmd := item.ResolvedCommandWithCatalog(ct, cat)
			fmt.Printf("  - %s  (signal=%.2f, cost=%.0fs)\n", cmd, item.Signal, item.RunCost)
		}
	}

	return nil
}

// jsonItem is the wire format for an ExecutionItem in --format=json.
// Mirrors ExecutionItem's JSON tags and adds the resolved command for
// consumers that don't want to re-materialize it from check_type_id +
// config + profile.
type jsonItem struct {
	ID            string                        `json:"id"`
	CheckTypeID   string                        `json:"check_type_id"`
	Command       string                        `json:"command"`
	Scope         []string                      `json:"scope,omitempty"`
	Config        map[string]any                `json:"config,omitempty"`
	Profile       string                        `json:"profile,omitempty"`
	Signal        float64                       `json:"signal"`
	RunCost       float64                       `json:"run_cost"`
	SkipCost      float64                       `json:"skip_cost"`
	Prerequisites []string                      `json:"prerequisites,omitempty"`
	Provenance    []selector.SignalContribution `json:"provenance,omitempty"`
}

type jsonResult struct {
	Stage     string     `json:"stage"`
	WallClock float64    `json:"wall_clock"`
	TotalCPU  float64    `json:"total_cpu"`
	Items     []jsonItem `json:"items"`
	Skipped   []jsonItem `json:"skipped"`
}

func printResultJSON(result *selector.Result, cat *catalog.Catalog) error {
	out := jsonResult{
		Stage:     result.Stage,
		WallClock: result.WallClock,
		TotalCPU:  result.TotalCPU,
		Items:     make([]jsonItem, len(result.Items)),
		Skipped:   make([]jsonItem, len(result.Skipped)),
	}
	for i, item := range result.Items {
		out.Items[i] = toJSONItem(item, cat)
	}
	for i, item := range result.Skipped {
		out.Skipped[i] = toJSONItem(item, cat)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func toJSONItem(item selector.ExecutionItem, cat *catalog.Catalog) jsonItem {
	ct := cat.ByID(item.CheckTypeID)
	return jsonItem{
		ID:            item.ID,
		CheckTypeID:   item.CheckTypeID,
		Command:       item.ResolvedCommandWithCatalog(ct, cat),
		Scope:         item.Scope,
		Config:        item.Config,
		Profile:       item.Profile,
		Signal:        item.Signal,
		RunCost:       item.RunCost,
		SkipCost:      item.SkipCost,
		Prerequisites: item.Prerequisites,
		Provenance:    item.Provenance,
	}
}
