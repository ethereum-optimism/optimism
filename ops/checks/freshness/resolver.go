package freshness

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Resolver maps graph node IDs to repo-relative filesystem paths.
//
// Three node-ID shapes are understood:
//
//	sol:<path>                  → packages/contracts-bedrock/<path>
//	go:<module>/<rel>.go        → <rel> (strips go.mod module prefix)
//	rs:<crate>/<rel>.rs         → path computed from the crate's
//	                              `dir` property in the graph, then
//	                              relativized to rootDir
//
// Non-file nodes (packages, crates, modules, checks) return "".
// Missing state (no go.mod, nil graph) silently degrades: Go and
// Rust nodes just don't resolve, which their callers treat as "file
// not found" and fall back appropriately.
//
// Resolver is cheap to construct — reads go.mod once — and safe to
// share across goroutines since it's read-only after construction.
type Resolver struct {
	rootDir      string
	goModulePath string
	graph        *graph.Graph
}

// NewResolver returns a Resolver backed by the given repo root and
// optional graph. Pass nil for graph if Rust node resolution isn't
// needed (e.g. tests, Go-only repos).
func NewResolver(rootDir string, g *graph.Graph) *Resolver {
	return &Resolver{
		rootDir:      rootDir,
		goModulePath: readGoModulePath(rootDir),
		graph:        g,
	}
}

// Resolve returns the repo-relative file path for nodeID, or "" if
// the node doesn't identify a single file or the required resolver
// state is unavailable.
func (r *Resolver) Resolve(nodeID string) string {
	switch {
	case strings.HasPrefix(nodeID, "sol:"):
		return filepath.Join("packages", "contracts-bedrock", strings.TrimPrefix(nodeID, "sol:"))

	case r.goModulePath != "" && strings.HasPrefix(nodeID, "go:") && strings.HasSuffix(nodeID, ".go"):
		path := strings.TrimPrefix(nodeID, "go:")
		if strings.HasPrefix(path, r.goModulePath+"/") {
			return strings.TrimPrefix(path, r.goModulePath+"/")
		}

	case r.graph != nil && strings.HasPrefix(nodeID, "rs:") && strings.HasSuffix(nodeID, ".rs"):
		trimmed := strings.TrimPrefix(nodeID, "rs:")
		slash := strings.Index(trimmed, "/")
		if slash <= 0 {
			return ""
		}
		crateNode := r.graph.GetNode("rs:" + trimmed[:slash])
		if crateNode == nil {
			return ""
		}
		dir, _ := crateNode.Properties["dir"].(string)
		if dir == "" {
			return ""
		}
		abs := filepath.Join(dir, trimmed[slash+1:])
		rel, err := filepath.Rel(r.rootDir, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return ""
		}
		return rel
	}
	return ""
}

// readGoModulePath returns the module directive from rootDir/go.mod,
// or "" if the file is missing or malformed.
func readGoModulePath(rootDir string) string {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
