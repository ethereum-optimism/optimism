package rust

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/graph"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// RustGraph implements graph.DependencyGraph using `cargo metadata`.
type RustGraph struct {
	root         string   // e.g., "rust"
	specialPaths []string
}

// NewGraph creates a new Rust dependency graph adapter.
func NewGraph(root string, specialPaths []string) *RustGraph {
	return &RustGraph{root: root, specialPaths: specialPaths}
}

func (g *RustGraph) Language() string { return "rust" }

// cargoMetadata is the output structure of `cargo metadata --format-version=1`.
type cargoMetadata struct {
	Packages []cargoPackage `json:"packages"`
	Resolve  cargoResolve   `json:"resolve"`
}

type cargoPackage struct {
	Name         string `json:"name"`
	ManifestPath string `json:"manifest_path"`
}

type cargoResolve struct {
	Nodes []cargoNode `json:"nodes"`
}

type cargoNode struct {
	ID   string      `json:"id"`
	Deps []cargoDep  `json:"deps"`
}

type cargoDep struct {
	Name string `json:"name"`
	Pkg  string `json:"pkg"`
}

func (g *RustGraph) Build(repoRoot string) (graph.Graph, error) {
	rustRoot := filepath.Join(repoRoot, g.root)
	cmd := exec.Command("cargo", "metadata", "--format-version=1", "--manifest-path", filepath.Join(rustRoot, "Cargo.toml"))
	cmd.Dir = rustRoot

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo metadata: %w", err)
	}

	var meta cargoMetadata
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("parsing cargo metadata: %w", err)
	}

	ag := graph.NewAdjacencyGraph()

	// Index packages by their cargo ID prefix (name).
	pkgByName := make(map[string]cargoPackage)
	for _, pkg := range meta.Packages {
		// Only include workspace packages (those with manifests under our root).
		if strings.HasPrefix(pkg.ManifestPath, rustRoot) {
			pkgByName[pkg.Name] = pkg
			ag.AddTarget(model.Target{
				ID:             pkg.Name,
				Language:       "rust",
				Scope:          model.ScopeAffected,
				Configurations: []model.Configuration{{Name: "default", Env: map[string]string{}}},
			})
		}
	}

	// Build dependency edges from the resolve graph.
	for _, node := range meta.Resolve.Nodes {
		nodeName := extractCrateName(node.ID)
		if _, ok := pkgByName[nodeName]; !ok {
			continue
		}
		for _, dep := range node.Deps {
			depName := dep.Name
			if _, ok := pkgByName[depName]; ok {
				ag.AddEdge(nodeName, depName)
			}
		}
	}

	return ag, nil
}

func (g *RustGraph) ChangedTargets(changedFiles []string) []model.Target {
	for _, f := range changedFiles {
		for _, sp := range g.specialPaths {
			if strings.HasPrefix(f, sp) || f == sp {
				return nil // nil signals "run everything"
			}
		}
	}

	// Map changed files to crates by finding the nearest Cargo.toml.
	seen := make(map[string]bool)
	var targets []model.Target

	prefix := g.root + "/"
	for _, f := range changedFiles {
		if !strings.HasPrefix(f, prefix) {
			continue
		}
		// Map changed files to crate directories.
		// Since we can't resolve crate names without cargo metadata, use the directory as ID.
		rel := strings.TrimPrefix(f, prefix)
		cratePath := filepath.Dir(rel)
		if !seen[cratePath] {
			seen[cratePath] = true
			targets = append(targets, model.Target{
				ID:       cratePath,
				Language: "rust",
			})
		}
	}
	return targets
}

func (g *RustGraph) ReverseDeps(gr graph.Graph, changed []model.Target) []model.Target {
	ag, ok := gr.(*graph.AdjacencyGraph)
	if !ok {
		return changed
	}

	// Resolve directory-based targets to crate names.
	resolved := make([]model.Target, 0, len(changed))
	for _, c := range changed {
		for id, t := range ag.Targets {
			// Match if the crate directory contains the changed path.
			if strings.Contains(c.ID, id) || id == c.ID {
				resolved = append(resolved, t)
			}
		}
	}

	return graph.TransitiveReverseDeps(ag, resolved)
}

func (g *RustGraph) TestTargets(targets []model.Target) []model.Target {
	// All Rust crates can have tests — return all.
	return targets
}

func (g *RustGraph) Configurations(_ []model.Target) []model.Configuration {
	return []model.Configuration{{Name: "default", Env: map[string]string{}}}
}

// extractCrateName extracts the crate name from a cargo package ID.
// Cargo IDs look like "kona-derive 0.4.5 (path+file:///...)"
func extractCrateName(id string) string {
	parts := strings.Fields(id)
	if len(parts) > 0 {
		return parts[0]
	}
	return id
}
