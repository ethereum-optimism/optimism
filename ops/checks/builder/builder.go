package builder

import (
	"fmt"
	"path/filepath"
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

	// Step 5: Create `produces` edges from each check node to the
	// source nodes whose freshness it's responsible for. The
	// catalog declares this via the `produces:` field (list of
	// file globs). During selection, a reverse walk that passes
	// through a produced node records the check as a per-file
	// prerequisite of downstream consumer Candidates.
	for _, ct := range b.catalog.CheckTypes {
		if len(ct.Produces) == 0 {
			continue
		}
		checkID := "check:" + ct.ID
		for _, node := range g.NodesOfKind(graph.KindSource) {
			path := nodePath(node)
			if path == "" || !matchesAnyGlob(path, ct.Produces) {
				continue
			}
			_ = g.AddEdge(&graph.Edge{
				From:       checkID,
				To:         node.ID,
				Kind:       graph.EdgeProduces,
				Source:     graph.SourceStatic,
				Confidence: 1.0,
				Strength:   1.0,
			})
		}
	}

	// Step 6: pipeline-model dataflow edges. For each check declaring
	// Inputs / Outputs / Tools, wire consumes/produces edges to source
	// and artifact nodes. This runs in parallel with the legacy
	// Triggers / Prerequisites / Produces wiring — checks that haven't
	// migrated emit no edges here and stay on the legacy path.
	for _, ct := range b.catalog.CheckTypes {
		if len(ct.Inputs) == 0 && len(ct.Outputs) == 0 && len(ct.Tools) == 0 {
			continue
		}
		b.emitDataflowEdges(g, ct)
	}

	return g, nil
}

// emitDataflowEdges wires pipeline-model edges for a single check.
// It creates artifact: nodes on demand (toolchain artifacts from
// Tools, output artifacts from Outputs that use artifact: refs),
// matches source nodes against input globs, and creates consumes /
// produces edges accordingly.
func (b *Builder) emitDataflowEdges(g *graph.Graph, ct catalog.CheckType) {
	checkID := "check:" + ct.ID

	// Tools expand into consumes: [artifact:toolchain/<tool>].
	toolInputs := make([]string, 0, len(ct.Tools))
	for _, t := range ct.Tools {
		toolInputs = append(toolInputs, "artifact:toolchain/"+t)
	}
	allInputs := append(append([]string{}, toolInputs...), ct.Inputs...)

	for _, in := range allInputs {
		if strings.HasPrefix(in, "artifact:") {
			// Ensure the artifact node exists, then consumes edge.
			ensureArtifactNode(g, in)
			_ = g.AddEdge(&graph.Edge{
				From: checkID, To: in, Kind: graph.EdgeConsumes,
				Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
			})
			continue
		}
		// Path glob: match against existing source nodes.
		for _, node := range g.NodesOfKind(graph.KindSource) {
			path := nodePath(node)
			if path == "" || !matchesAnyGlob(path, []string{in}) {
				continue
			}
			_ = g.AddEdge(&graph.Edge{
				From: checkID, To: node.ID, Kind: graph.EdgeConsumes,
				Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
			})
		}
	}

	for _, out := range ct.Outputs {
		if strings.HasPrefix(out, "artifact:") {
			ensureArtifactNode(g, out)
			_ = g.AddEdge(&graph.Edge{
				From: checkID, To: out, Kind: graph.EdgeProduces,
				Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
			})
			continue
		}
		// Path-glob output: for checked-in generated files whose
		// source nodes already exist, match as before. For
		// transient outputs, the catalog author should use the
		// artifact: ref form.
		for _, node := range g.NodesOfKind(graph.KindSource) {
			path := nodePath(node)
			if path == "" || !matchesAnyGlob(path, []string{out}) {
				continue
			}
			_ = g.AddEdge(&graph.Edge{
				From: checkID, To: node.ID, Kind: graph.EdgeProduces,
				Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
			})
		}
	}
}

// ensureArtifactNode creates an artifact: node if absent. Artifact
// nodes are virtual — they may not correspond to any file on disk
// (e.g. artifact:toolchain/forge represents the installed forge
// binary, artifact:forge-artifacts/**/*.json is a path glob that
// covers whatever forge-build emits).
func ensureArtifactNode(g *graph.Graph, id string) {
	if g.GetNode(id) != nil {
		return
	}
	_ = g.AddNode(&graph.Node{
		ID:          id,
		Kind:        graph.KindArtifact,
		Granularity: "artifact",
		Name:        strings.TrimPrefix(id, "artifact:"),
	})
}

// nodePath extracts the repo-relative path a source node represents,
// matching how CheckType.Produces globs are written (repo-relative).
// Returns "" for module nodes and other non-file source nodes.
func nodePath(n *graph.Node) string {
	switch {
	case strings.HasPrefix(n.ID, "sol:"):
		return "packages/contracts-bedrock/" + strings.TrimPrefix(n.ID, "sol:")
	case strings.HasPrefix(n.ID, "go:"):
		// Go nodes key by import path, not filesystem path; fall
		// back to the node's dir property when present.
		if dir, ok := n.Properties["dir"].(string); ok {
			return dir
		}
		return ""
	case strings.HasPrefix(n.ID, "rs:"):
		if dir, ok := n.Properties["dir"].(string); ok {
			return dir
		}
		return ""
	}
	return ""
}

// matchesAnyGlob reports whether path matches any of the supplied
// patterns. `**` suffix is stripped and treated as a prefix match.
func matchesAnyGlob(path string, patterns []string) bool {
	for _, p := range patterns {
		if strings.HasSuffix(p, "/**") {
			prefix := strings.TrimSuffix(p, "/**")
			if strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix) {
				return true
			}
			continue
		}
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		// Allow deep-suffix match: a glob "interfaces/**/*.sol"
		// should match an absolute go: dir ending with
		// "/interfaces/foo/bar.sol" too. The filesystem-path case
		// handles it directly; for the contracts-bedrock-rooted
		// case we also match without the prefix.
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
	}
	return false
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
