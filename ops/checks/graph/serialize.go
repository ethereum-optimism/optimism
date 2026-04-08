package graph

import (
	"encoding/json"
	"os"
)

// serializedGraph is the JSON representation of a graph.
// We separate this from Graph to avoid serializing internal index maps.
type serializedGraph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`
}

// Save writes the graph to a JSON file.
func Save(g *Graph, path string) error {
	g.mu.RLock()
	sg := serializedGraph{
		Nodes: g.Nodes,
		Edges: g.Edges,
	}
	g.mu.RUnlock()

	data, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads a graph from a JSON file and rebuilds internal indexes.
func Load(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sg serializedGraph
	if err := json.Unmarshal(data, &sg); err != nil {
		return nil, err
	}

	g := &Graph{
		Nodes:    sg.Nodes,
		Edges:    sg.Edges,
		outgoing: make(map[string][]*Edge),
		incoming: make(map[string][]*Edge),
	}

	if g.Nodes == nil {
		g.Nodes = make(map[string]*Node)
	}
	if g.Edges == nil {
		g.Edges = make([]*Edge, 0)
	}

	// Rebuild indexes
	for _, e := range g.Edges {
		g.outgoing[e.From] = append(g.outgoing[e.From], e)
		g.incoming[e.To] = append(g.incoming[e.To], e)
	}

	return g, nil
}
