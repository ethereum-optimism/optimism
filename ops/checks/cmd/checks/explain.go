package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func cmdExplain(args []string) error {
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	fs.Parse(args)

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: checks explain [--graph FILE] FILE_PATH")
	}

	filePath := fs.Arg(0)

	g, err := graph.Load(*graphPath)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Map file to node IDs
	nodeIDs, unknown := diff.FilesToNodeIDs(g, []string{filePath})
	if len(unknown) > 0 {
		fmt.Printf("File %q did not match any graph node.\n", filePath)
		return nil
	}

	fmt.Printf("File: %s\n", filePath)
	for _, nodeID := range nodeIDs {
		fmt.Printf("Node: %s\n\n", nodeID)

		// Show direct edges
		edges := g.EdgesFrom(nodeID)
		if len(edges) > 0 {
			fmt.Println("Direct edges:")
			for _, e := range edges {
				fmt.Printf("  → %s (%s, strength=%.2f, confidence=%.2f)\n",
					e.To, e.Kind, e.Strength, e.Confidence)
			}
		}

		// Show reachable checks
		reachable := graph.ReachableChecks(g, []string{nodeID}, 0.01)
		if len(reachable) > 0 {
			fmt.Printf("\nReachable checks (%d):\n", len(reachable))
			for _, r := range reachable {
				pathStr := strings.Join(r.Path, " → ")
				fmt.Printf("  %s (signal=%.3f)\n    path: %s\n", r.CheckID, r.Signal, pathStr)
			}
		}
	}

	return nil
}
