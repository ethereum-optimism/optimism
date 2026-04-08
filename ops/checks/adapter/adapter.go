package adapter

import "github.com/ethereum-optimism/optimism/ops/checks/graph"

// Adapter analyzes source code and adds nodes/edges to a graph.
type Adapter interface {
	// Name returns the adapter name (e.g. "go", "solidity").
	Name() string

	// Analyze scans the repo at rootDir and populates the graph
	// with source nodes and import edges.
	Analyze(g *graph.Graph, rootDir string) error
}
