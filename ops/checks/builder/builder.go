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
	return &Builder{
		adapters: adapters,
		catalog:  cat,
	}
}

// Build constructs the graph:
// 1. Run each adapter to populate source nodes and import edges
// 2. Create check nodes from the catalog
// 3. Wire check nodes to source nodes via tested_by edges
// 4. Create prerequisite edges between checks
func (b *Builder) Build(rootDir string) (*graph.Graph, error) {
	g := graph.NewGraph()

	// Step 1: Run adapters
	for _, a := range b.adapters {
		if err := a.Analyze(g, rootDir); err != nil {
			return nil, fmt.Errorf("adapter %q: %w", a.Name(), err)
		}
	}

	// Step 2: Create check nodes from catalog
	for _, ch := range b.catalog.Checks {
		props := map[string]any{
			"avg_duration": ch.AvgDuration,
			"kind":         ch.Kind,
			"language":     ch.Language,
			"command":      ch.Command,
		}
		if len(ch.Tags) > 0 {
			props["tags"] = ch.Tags
		}

		_ = g.AddNode(&graph.Node{
			ID:         "check:" + ch.ID,
			Kind:       graph.KindCheck,
			Name:       ch.Name,
			Properties: props,
		})
	}

	// Step 3: Wire checks to source nodes
	for _, ch := range b.catalog.Checks {
		checkID := "check:" + ch.ID

		// Wire by package name
		for _, pkg := range ch.Packages {
			if pkg == "*" {
				// Wildcard: connect to all source nodes of the matching language
				for _, node := range g.NodesOfKind(graph.KindSource) {
					lang, _ := node.Properties["language"].(string)
					if lang == "" {
						// Go packages don't have language property — infer from ID
						if strings.HasPrefix(node.ID, "go:") {
							lang = "go"
						} else if strings.HasPrefix(node.ID, "sol:") {
							lang = "solidity"
						}
					}
					if ch.Language == "*" || lang == ch.Language {
						wireTestedBy(g, node.ID, checkID)
					}
				}
				continue
			}

			// Match Go packages by prefix
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if !strings.HasPrefix(node.ID, "go:") {
					continue
				}
				goPath := strings.TrimPrefix(node.ID, "go:")
				// Match if the package path contains the package name
				// e.g., pkg "op-node" matches "go:.../op-node" and "go:.../op-node/rollup"
				if matchesPackage(goPath, pkg) {
					wireTestedBy(g, node.ID, checkID)
				}
			}
		}

		// Wire by directory
		for _, dir := range ch.Directories {
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if !strings.HasPrefix(node.ID, "sol:") {
					continue
				}
				solPath := strings.TrimPrefix(node.ID, "sol:")
				// Trim the common prefix (e.g., "packages/contracts-bedrock/")
				// and check if the sol path starts with the directory
				if matchesDirectory(solPath, dir) {
					wireTestedBy(g, node.ID, checkID)
				}
			}
		}
	}

	// Step 4: Create prerequisite edges
	for _, ch := range b.catalog.Checks {
		checkID := "check:" + ch.ID
		for _, prereq := range ch.Prerequisites {
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

// matchesPackage checks if a Go import path matches a package name pattern.
func matchesPackage(importPath, pkg string) bool {
	// Split import path into segments
	segments := strings.Split(importPath, "/")
	for _, seg := range segments {
		if seg == pkg {
			return true
		}
	}
	// Also check if the import path ends with the package or contains it as a prefix segment
	return strings.HasSuffix(importPath, "/"+pkg) ||
		strings.Contains(importPath, "/"+pkg+"/")
}

// matchesDirectory checks if a Solidity file path matches a directory pattern.
func matchesDirectory(solPath, dir string) bool {
	// Strip common prefixes
	dir = strings.TrimPrefix(dir, "packages/contracts-bedrock/")
	return strings.HasPrefix(solPath, dir)
}

func wireTestedBy(g *graph.Graph, sourceID, checkID string) {
	_ = g.AddEdge(&graph.Edge{
		From:       sourceID,
		To:         checkID,
		Kind:       graph.EdgeTestedBy,
		Source:     graph.SourceStatic,
		Confidence: 0.8,
		Strength:   0.9,
	})
}
