package graph

import (
	"sort"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestTransitiveReverseDeps(t *testing.T) {
	g := NewAdjacencyGraph()

	// Build a diamond dependency graph:
	// A -> B -> D
	// A -> C -> D
	for _, id := range []string{"A", "B", "C", "D"} {
		g.AddTarget(model.Target{ID: id, Language: "go"})
	}
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", "D")
	g.AddEdge("C", "D")

	tests := []struct {
		name     string
		changed  []model.Target
		expected []string
	}{
		{
			name:     "change D affects everything",
			changed:  []model.Target{{ID: "D"}},
			expected: []string{"A", "B", "C", "D"},
		},
		{
			name:     "change B affects A and B",
			changed:  []model.Target{{ID: "B"}},
			expected: []string{"A", "B"},
		},
		{
			name:     "change A affects only A",
			changed:  []model.Target{{ID: "A"}},
			expected: []string{"A"},
		},
		{
			name:     "change C affects A and C",
			changed:  []model.Target{{ID: "C"}},
			expected: []string{"A", "C"},
		},
		{
			name:     "multiple changes",
			changed:  []model.Target{{ID: "B"}, {ID: "C"}},
			expected: []string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransitiveReverseDeps(g, tt.changed)
			ids := make([]string, len(result))
			for i, r := range result {
				ids[i] = r.ID
			}
			sort.Strings(ids)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, ids)
		})
	}
}

func TestTransitiveReverseDeps_Linear(t *testing.T) {
	g := NewAdjacencyGraph()

	// Linear chain: A -> B -> C -> D -> E
	for _, id := range []string{"A", "B", "C", "D", "E"} {
		g.AddTarget(model.Target{ID: id, Language: "go"})
	}
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", "D")
	g.AddEdge("D", "E")

	result := TransitiveReverseDeps(g, []model.Target{{ID: "E"}})
	ids := make([]string, len(result))
	for i, r := range result {
		ids[i] = r.ID
	}
	sort.Strings(ids)
	assert.Equal(t, []string{"A", "B", "C", "D", "E"}, ids)
}

func TestTransitiveReverseDeps_Isolated(t *testing.T) {
	g := NewAdjacencyGraph()

	g.AddTarget(model.Target{ID: "A", Language: "go"})
	g.AddTarget(model.Target{ID: "B", Language: "go"})
	// No edges — they're isolated.

	result := TransitiveReverseDeps(g, []model.Target{{ID: "A"}})
	assert.Len(t, result, 1)
	assert.Equal(t, "A", result[0].ID)
}
