package main

import (
	"flag"
	"fmt"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func cmdInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	fs.Parse(args)

	g, err := graph.Load(*graphPath)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Count nodes by kind
	sourceCount := len(g.NodesOfKind(graph.KindSource))
	checkCount := len(g.NodesOfKind(graph.KindCheck))
	artifactCount := len(g.NodesOfKind(graph.KindArtifact))

	// Count edges by kind
	edgeCounts := make(map[graph.EdgeKind]int)
	for _, e := range g.Edges {
		edgeCounts[e.Kind]++
	}

	fmt.Printf("Graph Statistics\n")
	fmt.Printf("================\n\n")
	fmt.Printf("Nodes: %d total\n", g.NodeCount())
	fmt.Printf("  source:   %d\n", sourceCount)
	fmt.Printf("  check:    %d\n", checkCount)
	fmt.Printf("  artifact: %d\n", artifactCount)
	fmt.Printf("\nEdges: %d total\n", g.EdgeCount())
	for kind, count := range edgeCounts {
		fmt.Printf("  %s: %d\n", kind, count)
	}

	return nil
}
