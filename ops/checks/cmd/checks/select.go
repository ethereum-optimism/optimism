package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

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
	whyFilter := fs.String("why", "", "substring match against item IDs; prints matching items with full provenance")
	budget := fs.Duration("budget", 0, "max wall-clock budget (e.g. 30m). Lowest-value items are trimmed to fit; force-run items and their prerequisites are preserved")
	compareDataflow := fs.Bool("compare-dataflow", false, "also run the pipeline-model dataflow selector and report which check_types each picks")
	fs.Parse(args)

	resolvedGraph := resolveFromRoot(*graphPath)
	resolvedCatalog := resolveFromRoot(*catalogPath)

	pol, err := loadPolicy(*policyPath)
	if err != nil {
		return err
	}
	stage, err := pol.Stage(*stageName)
	if err != nil {
		return err
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
	fresh := freshness.New(findRepoRoot(), pol, g)
	candidates := selector.Resolve(g, diffs, cat, pol, fresh)

	if *compareDataflow {
		dfCands := selector.SelectViaDataflowWithCatalog(g, diffs, cat)
		legacySet := make(map[string]bool)
		for _, c := range candidates {
			legacySet[c.CheckID] = true
		}
		dfSet := make(map[string]bool)
		for _, c := range dfCands {
			dfSet[c.CheckID] = true
		}
		both, onlyLegacy, onlyDataflow := []string{}, []string{}, []string{}
		for k := range legacySet {
			if dfSet[k] {
				both = append(both, k)
			} else {
				onlyLegacy = append(onlyLegacy, k)
			}
		}
		for k := range dfSet {
			if !legacySet[k] {
				onlyDataflow = append(onlyDataflow, k)
			}
		}
		sort.Strings(both)
		sort.Strings(onlyLegacy)
		sort.Strings(onlyDataflow)
		fmt.Fprintf(os.Stderr, "\n=== dataflow vs legacy ===\n")
		fmt.Fprintf(os.Stderr, "both (%d):            %v\n", len(both), both)
		fmt.Fprintf(os.Stderr, "only legacy (%d):     %v\n", len(onlyLegacy), onlyLegacy)
		fmt.Fprintf(os.Stderr, "only dataflow (%d):   %v\n", len(onlyDataflow), onlyDataflow)
	}

	// Phase 2: Optimize — pure candidates → plan, no graph access.
	optimizer := selector.NewSimpleOptimizer(pol)
	result, err := optimizer.Optimize(candidates, stage, cat)
	if err != nil {
		return fmt.Errorf("optimizing: %w", err)
	}

	if *budget > 0 {
		selector.TrimToBudget(result, *budget, pol.MaxParallelism())
		if result.WallClock > budget.Seconds() {
			fmt.Fprintf(os.Stderr,
				"note: budget %s could not be met without dropping items where "+
					"skipping costs more regret than it saves runtime. "+
					"Plan estimated at %.0fs.\n",
				*budget, result.WallClock)
		}
	}

	if *format == "json" {
		if err := printResultJSON(result, cat); err != nil {
			return err
		}
	} else {
		if err := printResultText(result, cat); err != nil {
			return err
		}
	}
	if *whyFilter != "" {
		printWhy(result, cat, *whyFilter)
	}
	return nil
}

// printWhy emits a focused provenance dump for every item whose ID
// contains the given substring — items from both the plan and the
// skipped list. Meant for "the selector did X, what was the
// evidence?" debugging; complements `explain FILE` which starts from
// the file side.
func printWhy(result *selector.Result, cat *catalog.Catalog, pattern string) {
	fmt.Printf("\n--- why=%q ---\n", pattern)
	matches := 0
	for _, item := range result.Items {
		if containsFold(item.ID, pattern) {
			printItemProvenance(item, cat, "plan")
			matches++
		}
	}
	for _, item := range result.Skipped {
		if containsFold(item.ID, pattern) {
			printItemProvenance(item, cat, "skipped")
			matches++
		}
	}
	if matches == 0 {
		fmt.Printf("No items matched %q. Items in plan:\n", pattern)
		for _, item := range result.Items {
			fmt.Printf("  %s\n", item.ID)
		}
		fmt.Println("Skipped:")
		for _, item := range result.Skipped {
			fmt.Printf("  %s\n", item.ID)
		}
	}
}

func printItemProvenance(item selector.ExecutionItem, cat *catalog.Catalog, bucket string) {
	ct := cat.ByID(item.CheckTypeID)
	cmd := item.ResolvedCommandWithCatalog(ct, cat)

	fmt.Printf("\n[%s] %s\n", bucket, item.ID)
	fmt.Printf("  command:    %s\n", cmd)
	if len(item.Scope) > 0 {
		fmt.Printf("  scope:      %s\n", strings.Join(item.Scope, ", "))
	}
	if item.Profile != "" {
		fmt.Printf("  profile:    %s\n", item.Profile)
	}
	fmt.Printf("  signal:     %.3f\n", item.Signal)
	fmt.Printf("  run_cost:   %.1fs\n", item.RunCost)
	fmt.Printf("  skip_cost:  %.1fs (= signal*prior*miss_cost; force-run if > run_cost)\n", item.SkipCost)
	if len(item.Prerequisites) > 0 {
		fmt.Printf("  prereqs:    %s\n", strings.Join(item.Prerequisites, ", "))
	}
	if len(item.Provenance) == 0 {
		fmt.Println("  provenance: (none — prerequisite-only or synthetic)")
		return
	}
	fmt.Println("  provenance:")
	for _, p := range item.Provenance {
		kind := string(p.EdgeKind)
		if kind == "" {
			kind = "-"
		}
		fmt.Printf("    - %s/%s  contribution=%.3f", p.Source, kind, p.Contribution)
		if len(p.Raw) > 0 {
			fmt.Printf("  %s", formatRawMap(p.Raw))
		}
		fmt.Println()
	}
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func formatRawMap(raw map[string]any) string {
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, raw[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
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
