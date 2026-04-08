package builder

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Builder constructs a complete graph from adapters + catalog.
type Builder struct {
	adapters []adapter.Adapter
	catalog  *catalog.Catalog
}

// New creates a new Builder.
func New(adapters []adapter.Adapter, cat *catalog.Catalog) *Builder {
	return &Builder{adapters: adapters, catalog: cat}
}

// Build constructs the graph:
// 1. Run each adapter to populate source nodes and import edges
// 2. Create check type nodes from the catalog
// 3. Wire source nodes to check type nodes via tested_by edges (by language)
// 4. Create prerequisite edges between check types
func (b *Builder) Build(rootDir string) (*graph.Graph, error) {
	g := graph.NewGraph()

	// Step 1: Run adapters
	for _, a := range b.adapters {
		if err := a.Analyze(g, rootDir); err != nil {
			return nil, fmt.Errorf("adapter %q: %w", a.Name(), err)
		}
	}

	// Step 2: Create check type nodes
	for _, ct := range b.catalog.CheckTypes {
		props := map[string]any{
			"avg_duration": ct.AvgDuration,
			"kind":         ct.Kind,
			"language":     ct.Language,
			"scopeable":    ct.Scopeable,
		}
		if ct.Command != "" {
			props["command"] = ct.Command
		}

		_ = g.AddNode(&graph.Node{
			ID:         "check:" + ct.ID,
			Kind:       graph.KindCheck,
			Name:       ct.Name,
			Properties: props,
		})
	}

	// Step 3: Wire source nodes to check type nodes by language
	for _, ct := range b.catalog.CheckTypes {
		checkID := "check:" + ct.ID
		for _, node := range g.NodesOfKind(graph.KindSource) {
			lang := nodeLanguage(node)
			if ct.Language == "*" || lang == ct.Language {
				_ = g.AddEdge(&graph.Edge{
					From:       node.ID,
					To:         checkID,
					Kind:       graph.EdgeTestedBy,
					Source:     graph.SourceStatic,
					Confidence: 0.8,
					Strength:   0.9,
				})
			}
		}
	}

	// Step 4: Create prerequisite edges
	for _, ct := range b.catalog.CheckTypes {
		checkID := "check:" + ct.ID
		for _, prereq := range ct.Prerequisites {
			prereqID := "check:" + prereq
			_ = g.AddEdge(&graph.Edge{
				From:       prereqID,
				To:         checkID,
				Kind:       graph.EdgePrerequisite,
				Source:     graph.SourceManual,
				Confidence: 1.0,
				Strength:   1.0,
			})
		}
	}

	return g, nil
}

func nodeLanguage(node *graph.Node) string {
	if lang, ok := node.Properties["language"].(string); ok {
		return lang
	}
	if strings.HasPrefix(node.ID, "go:") {
		return "go"
	}
	if strings.HasPrefix(node.ID, "sol:") {
		return "solidity"
	}
	if strings.HasPrefix(node.ID, "rs:") {
		return "rust"
	}
	return ""
}
