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
	stageName := fs.String("stage", "commit", "development stage")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	policyPath := fs.String("policy", "", "optional policy override YAML")
	root := fs.String("root", ".", "repository root for execution")
	dryRun := fs.Bool("dry-run", false, "print commands without executing")
	diffFile := fs.String("diff", "", "path to diff file (default: stdin)")
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

	// Phase 1: Resolve — emit candidate items with per-source provenance.
	fresh := freshness.New(*root, pol, g)
	candidates := selector.Resolve(g, diffs, cat, pol, fresh)

	// Phase 2: Optimize — pure candidates → plan, no graph access.
	optimizer := selector.NewSimpleOptimizer(pol)
	result, err := optimizer.Optimize(candidates, stage, cat)
	if err != nil {
		return fmt.Errorf("optimizing: %w", err)
	}

	if len(result.Items) == 0 {
		fmt.Println("No checks to run.")
		return nil
	}

	fmt.Printf("Running %d items (stage: %s, est. %.0fs wall-clock, %d layers)...\n\n",
		len(result.Items), stage.Name, result.WallClock, len(result.Schedule.Layers))

	exec := executor.New(*root, *dryRun)
	runResult := exec.Run(result.Items, cat)

	for _, r := range runResult.Results {
		icon := "✓"
		switch r.Status {
		case executor.StatusFailed:
			icon = "✗"
		case executor.StatusSkipped:
			icon = "→"
		case executor.StatusError:
			icon = "!"
		}
		cmd := r.Command
		if cmd == "" {
			cmd = r.ItemID
		}
		fmt.Printf("  %s %s (%s)\n", icon, cmd, r.Duration.Round(100*1e6))
		if r.Status == executor.StatusFailed || r.Status == executor.StatusError {
			lines := strings.Split(r.Output, "\n")
			maxLines := 5
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for _, line := range lines[:maxLines] {
				fmt.Printf("    %s\n", line)
			}
		}
	}

	fmt.Printf("\nResults: %d passed, %d failed, %d skipped (%.1fs wall-clock)\n",
		runResult.Passed, runResult.Failed, runResult.Skipped, runResult.WallClock.Seconds())

	if runResult.Failed > 0 {
		return fmt.Errorf("%d checks failed", runResult.Failed)
	}
	return nil
}
