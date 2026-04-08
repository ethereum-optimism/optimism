package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Strategy computes an execution plan from a diff, stage, and graph.
// This is the pluggable algorithm — the core of the engine.
type Strategy interface {
	Plan(g *graph.Graph, diffs []diff.FileDiff, stage Stage, cat *catalog.Catalog) (*Result, error)
}
