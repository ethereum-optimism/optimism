package golang

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/graph"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// GoGraph implements graph.DependencyGraph using `go list -json ./...`.
type GoGraph struct {
	specialPaths []string
	root         string // relative path within repo (e.g., ".")
}

// NewGraph creates a new Go dependency graph adapter.
func NewGraph(root string, specialPaths []string) *GoGraph {
	return &GoGraph{root: root, specialPaths: specialPaths}
}

// goListPackage represents the JSON output of `go list -json`.
type goListPackage struct {
	ImportPath  string   `json:"ImportPath"`
	Dir         string   `json:"Dir"`
	GoFiles     []string `json:"GoFiles"`
	TestGoFiles []string `json:"TestGoFiles"`
	XTestGoFiles []string `json:"XTestGoFiles"`
	Imports      []string `json:"Imports"`
	TestImports  []string `json:"TestImports"`
	XTestImports []string `json:"XTestImports"`
}

func (g *GoGraph) Language() string { return "go" }

func (g *GoGraph) Build(repoRoot string) (graph.Graph, error) {
	dir := filepath.Join(repoRoot, g.root)
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	ag := graph.NewAdjacencyGraph()
	dec := json.NewDecoder(strings.NewReader(string(out)))

	for dec.More() {
		var pkg goListPackage
		if err := dec.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decoding go list output: %w", err)
		}

		hasTests := len(pkg.TestGoFiles) > 0 || len(pkg.XTestGoFiles) > 0
		target := model.Target{
			ID:       pkg.ImportPath,
			Language: "go",
			Scope:    model.ScopeAffected,
		}
		if hasTests {
			target.Configurations = []model.Configuration{{Name: "default", Env: map[string]string{}}}
		}
		ag.AddTarget(target)

		allImports := make([]string, 0, len(pkg.Imports)+len(pkg.TestImports)+len(pkg.XTestImports))
		allImports = append(allImports, pkg.Imports...)
		allImports = append(allImports, pkg.TestImports...)
		allImports = append(allImports, pkg.XTestImports...)

		for _, imp := range allImports {
			ag.AddEdge(pkg.ImportPath, imp)
		}
	}

	return ag, nil
}

func (g *GoGraph) ChangedTargets(changedFiles []string) []model.Target {
	// Check for special paths that force full run.
	for _, f := range changedFiles {
		for _, sp := range g.specialPaths {
			if strings.HasPrefix(f, sp) || f == sp {
				return nil // nil signals "run everything"
			}
		}
	}

	seen := make(map[string]bool)
	var targets []model.Target

	for _, f := range changedFiles {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		// Map file to its package directory, then to the Go import path.
		dir := filepath.Dir(f)
		if dir == "." {
			dir = ""
		}
		if !seen[dir] {
			seen[dir] = true
			targets = append(targets, model.Target{
				ID:       dir,
				Language: "go",
			})
		}
	}
	return targets
}

func (g *GoGraph) ReverseDeps(gr graph.Graph, changed []model.Target) []model.Target {
	ag, ok := gr.(*graph.AdjacencyGraph)
	if !ok {
		return changed
	}

	// Resolve changed targets to their full import paths.
	resolved := make([]model.Target, 0, len(changed))
	for _, c := range changed {
		for id, t := range ag.Targets {
			if strings.HasSuffix(id, "/"+c.ID) || id == c.ID {
				resolved = append(resolved, t)
			}
		}
	}

	return graph.TransitiveReverseDeps(ag, resolved)
}

func (g *GoGraph) TestTargets(targets []model.Target) []model.Target {
	var result []model.Target
	for _, t := range targets {
		if len(t.Configurations) > 0 {
			result = append(result, t)
		}
	}
	return result
}

func (g *GoGraph) Configurations(_ []model.Target) []model.Configuration {
	return []model.Configuration{{Name: "default", Env: map[string]string{}}}
}
