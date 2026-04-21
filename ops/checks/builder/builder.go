package builder

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/internal/glob"
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

	// Step 3: pipeline-model dataflow edges. For each check, wire
	// consumes/produces edges to source and artifact nodes based on
	// the catalog's inputs/outputs/tools declarations. The dataflow
	// walker uses these edges as the sole selection mechanism.
	for _, ct := range b.catalog.CheckTypes {
		if len(ct.Inputs) == 0 && len(ct.Outputs) == 0 && len(ct.Tools) == 0 && len(b.catalog.UniversalInputs) == 0 {
			continue
		}
		b.emitDataflowEdges(g, ct)
	}

	// Step 7: bridge-imports — for every artifact node produced by a
	// check, emit `imports` edges from Go/Rust package/crate source
	// nodes whose filesystem dir is covered by the artifact's path
	// prefix. Lets the scope-walker's reverse-imports walk reach
	// package consumers of generated bindings even though the bindings
	// themselves are artifacts, not source nodes.
	b.emitBridgeImportsEdges(g)

	return g, nil
}

// emitBridgeImportsEdges wires scope-layer imports from source
// packages to artifact nodes whose path covers them. Gen-go-bindings
// outputs like artifact:op-e2e/bindings/**/*.go become reverse-walk
// reachable from any Go package node whose dir starts with op-e2e/bindings/.
// selectViaDataflow ignores EdgeImports — these edges are scoping-only.
func (b *Builder) emitBridgeImportsEdges(g *graph.Graph) {
	type artifactPrefix struct {
		nodeID string
		prefix string // filesystem-path prefix derived from the artifact path glob
	}
	var prefixes []artifactPrefix
	for _, n := range g.NodesOfKind(graph.KindArtifact) {
		p := strings.TrimPrefix(n.ID, "artifact:")
		// Strip trailing glob segment: "op-e2e/bindings/**/*.go" → "op-e2e/bindings"
		if i := strings.Index(p, "/**"); i >= 0 {
			p = p[:i]
		}
		if p == "" || strings.HasPrefix(p, "toolchain/") || strings.HasPrefix(p, "forge-artifacts") {
			continue
		}
		prefixes = append(prefixes, artifactPrefix{nodeID: n.ID, prefix: p})
	}
	if len(prefixes) == 0 {
		return
	}
	for _, node := range g.NodesOfKind(graph.KindSource) {
		dir := nodePath(node)
		if dir == "" {
			continue
		}
		for _, ap := range prefixes {
			if !(dir == ap.prefix || strings.HasPrefix(dir, ap.prefix+"/") || strings.HasSuffix(dir, "/"+ap.prefix)) {
				continue
			}
			_ = g.AddEdge(&graph.Edge{
				From: node.ID, To: ap.nodeID, Kind: graph.EdgeImports,
				Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
			})
		}
	}
}

// extToLang maps a path-glob extension to the adapter language key.
// A catalog pattern like "**/*.go" is conceptually "all Go source";
// go: nodes are import-path-keyed, not filesystem-keyed, so the glob
// wouldn't match any node's dir. Special-case language matching.
var extToLang = map[string]string{
	".go":  "go",
	".sol": "solidity",
	".rs":  "rust",
}

// languageForGlob returns the language key if the pattern is a
// language-wide glob like `**/*.go`, `**/*.sol`, `**/*.rs`. Returns ""
// for any other pattern, signaling that path-keyed matching should
// be used instead.
func languageForGlob(pattern string) string {
	if !strings.HasPrefix(pattern, "**/*") {
		return ""
	}
	ext := strings.TrimPrefix(pattern, "**/*")
	return extToLang[ext]
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
	// Universal inputs — paths every check implicitly consumes (CI
	// config, github actions). Diffs touching these fan out to every
	// check via dataflow, replacing the old blast_radius_patterns.
	allInputs = append(allInputs, b.catalog.UniversalInputs...)

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
		// Language-wide globs like `**/*.go` match every node of that
		// language (adapter-keyed), regardless of filesystem layout.
		if lang := languageForGlob(in); lang != "" {
			for _, node := range g.NodesOfKind(graph.KindSource) {
				if nodeLanguage(node) != lang {
					continue
				}
				_ = g.AddEdge(&graph.Edge{
					From: checkID, To: node.ID, Kind: graph.EdgeConsumes,
					Source: graph.SourceStatic, Confidence: 1.0, Strength: 1.0,
				})
			}
			continue
		}
		// Path glob: match against existing source nodes by filesystem path.
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
// glob patterns. Wraps glob.MatchAny to keep the import single-sourced.
func matchesAnyGlob(path string, patterns []string) bool {
	return glob.MatchAny(path, patterns)
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
