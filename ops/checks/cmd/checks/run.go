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
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/scorer"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	stageName := fs.String("stage", "commit", "development stage")
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	root := fs.String("root", ".", "repository root for execution")
	dryRun := fs.Bool("dry-run", false, "print commands without executing")
	diffFile := fs.String("diff", "", "path to diff file (default: stdin)")
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
	nodeIDs, _ := diff.FilesToNodeIDs(g, files)

	// Walk, score, select
	reachable := graph.ReachableChecks(g, nodeIDs, 0.01)
	s := scorer.NewSimple()
	scores, err := s.Score(g, files, reachable)
	if err != nil {
		return fmt.Errorf("scoring: %w", err)
	}
	result := selector.Select(scores, stage, g)

	if len(result.Selections) == 0 {
		fmt.Println("No checks to run.")
		return nil
	}

	fmt.Printf("Running %d checks (stage: %s, est. %.0fs wall-clock, %d layers)...\n\n",
		len(result.Selections), stage.Name, result.WallClock, len(result.Schedule.Layers))

	// Execute
	exec := executor.New(*root, *dryRun)
	runResult := exec.Run(result.Selections, cat)

	// Print results
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
		fmt.Printf("  %s %s (%s)\n", icon, r.CheckID, r.Duration.Round(100*1e6))
		if r.Status == executor.StatusFailed || r.Status == executor.StatusError {
			// Print first few lines of output
			lines := strings.Split(r.Output, "\n")
			max := 5
			if len(lines) < max {
				max = len(lines)
			}
			for _, line := range lines[:max] {
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
