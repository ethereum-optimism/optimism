package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/executor"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)

	// Diff source selection — mutually exclusive; if none specified,
	// defaults to branch-vs-base (auto).
	base := fs.String("base", "", "base branch ref for auto mode (defaults to origin/develop or origin/main)")
	staged := fs.Bool("staged", false, "use `git diff --cached` (staged changes only)")
	uncommitted := fs.Bool("uncommitted", false, "use `git diff HEAD` (staged + unstaged)")
	commitSHA := fs.String("commit", "", "run for a specific commit (diff commit^..commit)")
	diffFile := fs.String("diff", "", "read unified diff from a file (- for stdin)")

	stageName := fs.String("stage", "pr", "development stage (save, commit, pr, merge_queue, develop)")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	policyPath := fs.String("policy", "", "optional policy override YAML")
	root := fs.String("root", ".", "repository root for execution")
	dryRun := fs.Bool("dry-run", false, "print commands without executing")
	plan := fs.Bool("plan", false, "print the plan and exit without running")
	fs.Parse(args)

	resolvedGraph := resolveFromRoot(*graphPath)
	resolvedCatalog := resolveFromRoot(*catalogPath)
	resolvedRoot := *root
	if resolvedRoot == "." {
		resolvedRoot = findRepoRoot()
	}

	// Resolve diff source. If stdin is piped and no explicit source
	// flag is set, treat stdin as the diff — `git diff | checks run`
	// should Just Work without --diff -.
	src, err := pickDiffSource(*staged, *uncommitted, *commitSHA, *diffFile)
	if err != nil {
		return err
	}
	if src == sourceAuto && stdinIsPipe() {
		src = sourceStdin
	}
	if src == sourceAuto && *base == "" {
		*base = defaultBaseBranch(resolvedRoot)
	}
	var diffInput io.Reader = os.Stdin
	if *diffFile != "" && *diffFile != "-" {
		f, err := os.Open(*diffFile)
		if err != nil {
			return fmt.Errorf("opening diff file: %w", err)
		}
		defer f.Close()
		diffInput = f
	}
	diffText, err := computeDiff(src, resolvedRoot, *base, *commitSHA, diffInput)
	if err != nil {
		return err
	}
	diffs := diff.ParseUnifiedDiff(diffText)
	if len(diffs) == 0 {
		fmt.Fprintln(os.Stderr, diffSourceSummary(src, *base, *commitSHA)+": no changes")
		return nil
	}
	fmt.Fprintln(os.Stderr, diffSourceSummary(src, *base, *commitSHA)+fmt.Sprintf(" (%d files)", len(diffs)))

	pol, err := loadPolicy(*policyPath)
	if err != nil {
		return err
	}
	stage, err := pol.Stage(*stageName)
	if err != nil {
		return err
	}

	warnIfGraphStale(resolvedGraph, resolvedRoot)
	g, err := graph.Load(resolvedGraph)
	if err != nil {
		return missingGraphError(resolvedGraph, err)
	}
	cat, err := catalog.Load(resolvedCatalog)
	if err != nil {
		return fmt.Errorf("loading catalog %s: %w", resolvedCatalog, err)
	}

	fresh := freshness.New(resolvedRoot, pol, g)
	candidates := selector.Resolve(g, diffs, cat, pol, fresh)

	optimizer := selector.NewSimpleOptimizer(pol)
	optimizer.SetGraph(g)
	result, err := optimizer.Optimize(candidates, stage, cat)
	if err != nil {
		return fmt.Errorf("optimizing: %w", err)
	}

	if len(result.Items) == 0 {
		fmt.Println("No checks to run for this diff.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nStage: %s   Items: %d   Layers: %d   Est. wall-clock: %s\n",
		stage.Name, len(result.Items), len(result.Schedule.Layers), humanSeconds(result.WallClock))
	printPlan(result, cat)
	if *plan {
		return nil
	}
	fmt.Fprintln(os.Stderr)

	exec := executor.New(resolvedRoot, *dryRun)
	runResult := exec.Run(result.Items, cat)

	// Print failure details (full output, no truncation).
	for _, r := range runResult.Results {
		if r.Status != executor.StatusFailed && r.Status != executor.StatusError {
			continue
		}
		fmt.Println()
		fmt.Println(strings.Repeat("─", 72))
		fmt.Printf("✗ %s  (%s)\n", r.ItemID, r.Duration.Round(100*1e6))
		fmt.Printf("  $ %s\n", r.Command)
		fmt.Println(strings.Repeat("─", 72))
		fmt.Println(r.Output)
	}

	// Final summary.
	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed, %d skipped — %s wall-clock\n",
		runResult.Passed, runResult.Failed, runResult.Skipped,
		humanSeconds(runResult.WallClock.Seconds()))

	if runResult.Failed > 0 {
		return fmt.Errorf("%d checks failed", runResult.Failed)
	}
	return nil
}

func pickDiffSource(staged, uncommitted bool, commitSHA, diffFile string) (diffSource, error) {
	n := 0
	if staged {
		n++
	}
	if uncommitted {
		n++
	}
	if commitSHA != "" {
		n++
	}
	if diffFile != "" {
		n++
	}
	if n > 1 {
		return sourceAuto, fmt.Errorf("--staged / --uncommitted / --commit / --diff are mutually exclusive")
	}
	switch {
	case diffFile != "":
		return sourceStdin, nil
	case staged:
		return sourceStaged, nil
	case uncommitted:
		return sourceUncommitted, nil
	case commitSHA != "":
		return sourceCommit, nil
	default:
		return sourceAuto, nil
	}
}

func diffSourceSummary(src diffSource, base, commitSHA string) string {
	switch src {
	case sourceAuto:
		return fmt.Sprintf("Diff: HEAD vs merge-base(%s, HEAD)", base)
	case sourceStaged:
		return "Diff: staged changes"
	case sourceUncommitted:
		return "Diff: uncommitted changes (staged + unstaged)"
	case sourceCommit:
		return fmt.Sprintf("Diff: commit %s", commitSHA)
	case sourceStdin:
		return "Diff: from file/stdin"
	}
	return "Diff:"
}

// printPlan emits a compact view of what will run, grouped by layer.
func printPlan(result *selector.Result, cat *catalog.Catalog) {
	for i, layer := range result.Schedule.Layers {
		fmt.Fprintf(os.Stderr, "  Layer %d (%s):\n", i+1, humanSeconds(layer.Duration))
		for _, itemID := range layer.ItemIDs {
			item := findItem(result.Items, itemID)
			if item == nil {
				continue
			}
			ct := cat.ByID(item.CheckTypeID)
			cmd := item.ResolvedCommandWithCatalog(ct, cat)
			scopeInfo := ""
			if len(item.Scope) > 0 {
				scopeInfo = fmt.Sprintf("  [%d scopes]", len(item.Scope))
			}
			fmt.Fprintf(os.Stderr, "    - %-40s %s%s\n",
				item.ID, truncate(cmd, 80), scopeInfo)
		}
	}
}

func findItem(items []selector.ExecutionItem, id string) *selector.ExecutionItem {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func humanSeconds(seconds float64) string {
	if seconds < 1 {
		return "<1s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		m := int(seconds) / 60
		s := int(seconds) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
