package graph

import "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"

// TransitiveReverseDeps computes all targets that transitively depend on the
// given set of changed targets using BFS over reverse edges.
func TransitiveReverseDeps(g *AdjacencyGraph, changed []model.Target) []model.Target {
	visited := make(map[string]bool)
	queue := make([]string, 0, len(changed))

	for _, t := range changed {
		if !visited[t.ID] {
			visited[t.ID] = true
			queue = append(queue, t.ID)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range g.Reverse[current] {
			if !visited[dep] {
				visited[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	result := make([]model.Target, 0, len(visited))
	for id := range visited {
		if t, ok := g.Targets[id]; ok {
			result = append(result, t)
		}
	}
	return result
}
