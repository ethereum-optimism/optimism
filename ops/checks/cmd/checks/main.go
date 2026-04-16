package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func loadGraph(path string) (*graph.Graph, error) {
	g, err := graph.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}
	return g, nil
}

func saveGraph(g *graph.Graph, path string) error {
	if err := graph.Save(g, path); err != nil {
		return fmt.Errorf("saving graph: %w", err)
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "build":
		err = cmdBuild(os.Args[2:])
	case "select":
		err = cmdSelect(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "explain":
		err = cmdExplain(os.Args[2:])
	case "info":
		err = cmdInfo(os.Args[2:])
	case "coverage":
		err = cmdCoverage(os.Args[2:])
	case "ingest":
		err = cmdIngest(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`checks — smart check selection engine

Usage: checks <command> [flags]

Commands:
  build     Build the dependency graph from source analysis + catalog
  select    Select checks to run for a given diff and stage
  run       Select and execute checks
  explain   Show which checks are affected by a file and why
  info      Print graph statistics
  coverage  Collect and ingest test coverage data
  ingest    Ingest external signals (ci-history) into graph + learned policy

Examples:
  checks build
  git diff develop | checks select --stage commit
  git diff --name-only develop | checks select --stage pr
  git diff HEAD~1 | checks run --stage commit --dry-run`)
}
