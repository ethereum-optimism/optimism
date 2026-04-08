package main

import (
	"flag"
	"fmt"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/golang"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/rust"
	"github.com/ethereum-optimism/optimism/ops/checks/adapter/solidity"
	"github.com/ethereum-optimism/optimism/ops/checks/builder"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func cmdBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	root := fs.String("root", ".", "repository root directory")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	output := fs.String("output", "ops/checks/graph.json", "output path for graph")
	fs.Parse(args)

	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
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
	g, err := b.Build(*root)
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	if err := graph.Save(g, *output); err != nil {
		return fmt.Errorf("saving graph: %w", err)
	}

	fmt.Printf("Graph built: %d nodes, %d edges → %s\n", g.NodeCount(), g.EdgeCount(), *output)
	return nil
}
