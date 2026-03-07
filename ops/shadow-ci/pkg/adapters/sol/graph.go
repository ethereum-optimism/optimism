package sol

import (
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/graph"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// SolGraph implements graph.DependencyGraph via Solidity import parsing.
type SolGraph struct {
	root           string   // e.g., "packages/contracts-bedrock"
	sourceDirs     []string // e.g., ["src/", "test/", "scripts/"]
	remappingsFile string   // e.g., "remappings.txt"
	specialPaths   []string
	features       []model.FeatureRule
}

// NewGraph creates a new Solidity dependency graph adapter.
func NewGraph(root string, sourceDirs []string, remappingsFile string, specialPaths []string, features []model.FeatureRule) *SolGraph {
	return &SolGraph{
		root:           root,
		sourceDirs:     sourceDirs,
		remappingsFile: remappingsFile,
		specialPaths:   specialPaths,
		features:       features,
	}
}

func (g *SolGraph) Language() string { return "sol" }

func (g *SolGraph) Build(repoRoot string) (graph.Graph, error) {
	contractsRoot := filepath.Join(repoRoot, g.root)

	// Parse remappings.
	remappings, err := ParseRemappings(filepath.Join(contractsRoot, g.remappingsFile))
	if err != nil {
		return nil, err
	}

	// Collect all .sol files.
	files, err := CollectSolFiles(contractsRoot, g.sourceDirs)
	if err != nil {
		return nil, err
	}

	ag := graph.NewAdjacencyGraph()

	// Register all files as targets.
	for _, f := range files {
		isTest := strings.HasSuffix(f, ".t.sol")
		target := model.Target{
			ID:       f,
			Language: "sol",
			Scope:    model.ScopeAffected,
		}
		if isTest {
			target.Configurations = []model.Configuration{{Name: "main", Env: map[string]string{"FOUNDRY_PROFILE": "ci"}}}
		}
		ag.AddTarget(target)
	}

	// Build import graph.
	for _, f := range files {
		absPath := filepath.Join(contractsRoot, f)
		imports, err := ParseImports(absPath)
		if err != nil {
			continue
		}
		for _, imp := range imports {
			resolved := remappings.ResolveImport(imp)
			// Normalize to relative path within contracts root.
			if !strings.HasPrefix(resolved, "/") {
				resolved = filepath.Clean(resolved)
			}
			if ag.Contains(resolved) {
				ag.AddEdge(f, resolved)
			}
		}
	}

	return ag, nil
}

func (g *SolGraph) ChangedTargets(changedFiles []string) []model.Target {
	for _, f := range changedFiles {
		for _, sp := range g.specialPaths {
			if strings.HasPrefix(f, sp) || f == sp {
				return nil // nil signals "run everything"
			}
		}
	}

	var targets []model.Target
	seen := make(map[string]bool)

	prefix := g.root + "/"
	for _, f := range changedFiles {
		if !strings.HasSuffix(f, ".sol") {
			continue
		}
		// Strip the contracts root prefix to get the relative path.
		rel := f
		if strings.HasPrefix(f, prefix) {
			rel = strings.TrimPrefix(f, prefix)
		}
		if !seen[rel] {
			seen[rel] = true
			targets = append(targets, model.Target{
				ID:       rel,
				Language: "sol",
			})
		}
	}
	return targets
}

func (g *SolGraph) ReverseDeps(gr graph.Graph, changed []model.Target) []model.Target {
	ag, ok := gr.(*graph.AdjacencyGraph)
	if !ok {
		return changed
	}
	return graph.TransitiveReverseDeps(ag, changed)
}

func (g *SolGraph) TestTargets(targets []model.Target) []model.Target {
	var result []model.Target
	for _, t := range targets {
		if strings.HasSuffix(t.ID, ".t.sol") {
			result = append(result, t)
		}
	}
	return result
}

func (g *SolGraph) Configurations(targets []model.Target) []model.Configuration {
	configs := []model.Configuration{}

	for _, rule := range g.features {
		if rule.Always {
			configs = append(configs, model.Configuration{
				Name: rule.Name,
				Env:  rule.Env,
			})
			continue
		}

		// Check if any target matches the feature's trigger paths.
		matched := false
		for _, target := range targets {
			for _, tp := range rule.TriggerPaths {
				if strings.Contains(target.ID, tp) || strings.HasPrefix(target.ID, tp) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			configs = append(configs, model.Configuration{
				Name: rule.Name,
				Env:  rule.Env,
			})
		}
	}

	if len(configs) == 0 {
		configs = append(configs, model.Configuration{Name: "main", Env: map[string]string{"FOUNDRY_PROFILE": "ci"}})
	}

	return deduplicate(configs)
}

func deduplicate(configs []model.Configuration) []model.Configuration {
	seen := make(map[string]bool)
	var result []model.Configuration
	for _, c := range configs {
		if !seen[c.Name] {
			seen[c.Name] = true
			result = append(result, c)
		}
	}
	return result
}
