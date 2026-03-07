package graph

import "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"

// DependencyGraph provides the dependency structure for a language ecosystem.
// The core engine uses it to compute affected targets.
type DependencyGraph interface {
	// Build constructs the full dependency graph for the repository.
	Build(repoRoot string) (Graph, error)

	// ChangedTargets maps changed file paths to the targets they belong to.
	ChangedTargets(changedFiles []string) []model.Target

	// ReverseDeps returns all targets that transitively depend on the given targets.
	ReverseDeps(g Graph, changed []model.Target) []model.Target

	// TestTargets filters targets to only those that contain tests.
	TestTargets(targets []model.Target) []model.Target

	// Configurations returns the build configurations relevant to the given targets.
	Configurations(targets []model.Target) []model.Configuration

	// Language returns the adapter's language identifier.
	Language() string
}

// Graph is the dependency graph structure.
type Graph interface {
	// AllTargets returns every target in the graph.
	AllTargets() []model.Target

	// Contains checks whether a target exists in the graph.
	Contains(id string) bool
}

// AdjacencyGraph is a concrete graph backed by an adjacency list.
type AdjacencyGraph struct {
	// Forward edges: target → targets it depends on.
	Forward map[string][]string

	// Reverse edges: target → targets that depend on it.
	Reverse map[string][]string

	// All targets indexed by ID.
	Targets map[string]model.Target
}

// AllTargets returns every target in the graph.
func (g *AdjacencyGraph) AllTargets() []model.Target {
	targets := make([]model.Target, 0, len(g.Targets))
	for _, t := range g.Targets {
		targets = append(targets, t)
	}
	return targets
}

// Contains checks whether a target exists in the graph.
func (g *AdjacencyGraph) Contains(id string) bool {
	_, ok := g.Targets[id]
	return ok
}

// NewAdjacencyGraph creates an empty adjacency graph.
func NewAdjacencyGraph() *AdjacencyGraph {
	return &AdjacencyGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
		Targets: make(map[string]model.Target),
	}
}

// AddTarget registers a target.
func (g *AdjacencyGraph) AddTarget(t model.Target) {
	g.Targets[t.ID] = t
}

// AddEdge adds a dependency edge: from depends on to.
func (g *AdjacencyGraph) AddEdge(from, to string) {
	g.Forward[from] = append(g.Forward[from], to)
	g.Reverse[to] = append(g.Reverse[to], from)
}
