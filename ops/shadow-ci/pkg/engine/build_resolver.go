package engine

import (
	"sort"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// BuildResolver resolves test selections to required build artifacts.
type BuildResolver struct {
	scoping model.ScopingConfig
}

// NewBuildResolver creates a new BuildResolver.
func NewBuildResolver(scoping model.ScopingConfig) *BuildResolver {
	return &BuildResolver{scoping: scoping}
}

// Resolve takes the set of needed test categories and returns the build categories they require.
// Uses DependsOn chains from scoping config. Includes transitive dependencies.
func (br *BuildResolver) Resolve(categories map[string]*model.CategoryDecision) []string {
	needed := make(map[string]bool)
	visited := make(map[string]bool)

	for name, cd := range categories {
		if !cd.Needed || cd.Skipped {
			continue
		}
		cat, ok := br.scoping.JobCategories[name]
		if !ok {
			continue
		}
		br.collectBuildDeps(cat.DependsOn, needed, visited)
	}

	result := make([]string, 0, len(needed))
	for name := range needed {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// collectBuildDeps recursively collects build-category dependencies.
func (br *BuildResolver) collectBuildDeps(deps []string, needed, visited map[string]bool) {
	for _, dep := range deps {
		if visited[dep] {
			continue
		}
		visited[dep] = true

		cat, ok := br.scoping.JobCategories[dep]
		if !ok {
			continue
		}
		if cat.Group == "build" {
			needed[dep] = true
		}
		// Recurse into transitive deps.
		br.collectBuildDeps(cat.DependsOn, needed, visited)
	}
}

// contains checks if a string slice contains a value.
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
