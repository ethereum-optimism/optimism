package graph

import (
	"fmt"
	"sync"
)

// NodeKind classifies what a node represents.
type NodeKind string

const (
	KindSource   NodeKind = "source"
	KindCheck    NodeKind = "check"
	KindArtifact NodeKind = "artifact"
	// KindModule represents an external dependency (e.g. a Go module)
	// that packages in this repo import. Config-file changes like
	// go.mod version bumps feed module IDs into Phase 1 so the reverse-
	// walk discovers affected consumer packages via the normal
	// transitiveConsumers path.
	KindModule NodeKind = "module"
)

// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
	EdgeImports             EdgeKind = "imports"
	EdgeGenerates           EdgeKind = "generates"
	EdgeTestedBy            EdgeKind = "tested_by"
	EdgePrerequisite        EdgeKind = "prerequisite"
	EdgeObservedCorrelation EdgeKind = "observed_correlation"
	EdgeAIAnnotated         EdgeKind = "ai_annotated"
)

// EdgeSource records how an edge was discovered.
type EdgeSource string

const (
	SourceStatic    EdgeSource = "static"
	SourceCoverage  EdgeSource = "coverage"
	SourceAI        EdgeSource = "ai"
	SourceCIHistory EdgeSource = "ci_history"
	SourceManual    EdgeSource = "manual"
)

// Node represents an entity in the dependency graph.
type Node struct {
	ID          string         `json:"id"`
	Kind        NodeKind       `json:"kind"`
	Granularity string         `json:"granularity"`
	Name        string         `json:"name"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From       string         `json:"from"`
	To         string         `json:"to"`
	Kind       EdgeKind       `json:"kind"`
	Source     EdgeSource     `json:"source"`
	Confidence float64        `json:"confidence"`
	Strength   float64        `json:"strength"`
	Properties map[string]any `json:"properties,omitempty"`
}

// Graph is the top-level container for nodes and edges.
type Graph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []*Edge          `json:"edges"`

	mu       sync.RWMutex
	outgoing map[string][]*Edge // from -> edges
	incoming map[string][]*Edge // to -> edges
}

// NewGraph returns an initialized empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Edges:    make([]*Edge, 0),
		outgoing: make(map[string][]*Edge),
		incoming: make(map[string][]*Edge),
	}
}

// AddNode adds a node, returning an error if the ID already exists.
func (g *Graph) AddNode(n *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Nodes[n.ID]; exists {
		return fmt.Errorf("node %q already exists", n.ID)
	}
	g.Nodes[n.ID] = n
	return nil
}

// AddEdge adds a directed edge. Both From and To must exist.
func (g *Graph) AddEdge(e *Edge) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.Nodes[e.From]; !ok {
		return fmt.Errorf("source node %q not found", e.From)
	}
	if _, ok := g.Nodes[e.To]; !ok {
		return fmt.Errorf("target node %q not found", e.To)
	}
	g.Edges = append(g.Edges, e)
	g.outgoing[e.From] = append(g.outgoing[e.From], e)
	g.incoming[e.To] = append(g.incoming[e.To], e)
	return nil
}

// Node returns a node by ID, or nil if not found.
func (g *Graph) GetNode(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Nodes[id]
}

// NodesOfKind returns all nodes matching the given kind.
func (g *Graph) NodesOfKind(kind NodeKind) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Node
	for _, n := range g.Nodes {
		if n.Kind == kind {
			result = append(result, n)
		}
	}
	return result
}

// EdgesFrom returns all outgoing edges from a node.
func (g *Graph) EdgesFrom(id string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.outgoing[id]
}

// EdgesTo returns all incoming edges to a node.
func (g *Graph) EdgesTo(id string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.incoming[id]
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Nodes)
}

// EdgeCount returns the number of edges in the graph.
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Edges)
}
