package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/golang"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/rust"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/solidity"
	"github.com/ethereum-optimism/optimism/ops/checks/builder"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/coverage"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func cmdBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	root := fs.String("root", "", "repository root directory (default: auto-discover via git)")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	output := fs.String("output", "ops/checks/graph.json", "output path for graph")
	coverageDir := fs.String("coverage-dir", "ops/checks/coverage-data",
		"coverage reports directory to auto-ingest (pass empty to skip)")
	fs.Parse(args)

	resolvedRoot := *root
	if resolvedRoot == "" {
		resolvedRoot = findRepoRoot()
	}
	resolvedCatalog := resolveFromRootDir(*catalogPath, resolvedRoot)
	resolvedOutput := resolveFromRootDir(*output, resolvedRoot)

	cat, err := catalog.Load(resolvedCatalog)
	if err != nil {
		return fmt.Errorf("loading catalog %s: %w", resolvedCatalog, err)
	}
	if err := cat.Validate(); err != nil {
		return fmt.Errorf("validating catalog: %w", err)
	}

	adapters := []adapter.Adapter{
		golang.New(),
		solidity.New(),
		rust.New(),
	}

	b := builder.New(adapters, cat)
	g, err := b.Build(resolvedRoot)
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	// Auto-ingest coverage reports if the directory exists. Without
	// this, a newly-built graph has zero tested_by/coverage edges,
	// and Phase 1's coverage path silently returns nothing —
	// producing under-selection on real diffs where coverage-based
	// scoping is the primary signal.
	covIngested := 0
	if *coverageDir != "" {
		covPath := resolveFromRootDir(*coverageDir, resolvedRoot)
		if _, err := os.Stat(covPath); err == nil {
			reports, err := coverage.LoadReports(covPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: loading coverage reports: %v\n", err)
			} else if err := coverage.IngestReports(g, reports); err != nil {
				fmt.Fprintf(os.Stderr, "warning: ingesting coverage: %v\n", err)
			} else {
				covIngested = len(reports)
			}
		}
	}

	if err := graph.Save(g, resolvedOutput); err != nil {
		return fmt.Errorf("saving graph: %w", err)
	}

	if covIngested > 0 {
		fmt.Printf("Graph built: %d nodes, %d edges (+%d coverage reports) → %s\n",
			g.NodeCount(), g.EdgeCount(), covIngested, resolvedOutput)
	} else {
		fmt.Printf("Graph built: %d nodes, %d edges → %s\n", g.NodeCount(), g.EdgeCount(), resolvedOutput)
	}
	return nil
}
