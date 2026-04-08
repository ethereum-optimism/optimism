package graph

import "testing"

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 edges, got %d", g.EdgeCount())
	}
}

func TestAddNode(t *testing.T) {
	g := NewGraph()
	err := g.AddNode(&Node{ID: "a", Kind: KindSource, Name: "node a"})
	if err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", g.NodeCount())
	}
	if g.GetNode("a") == nil {
		t.Error("expected to find node 'a'")
	}
}

func TestAddNode_Duplicate(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Kind: KindSource, Name: "node a"})
	err := g.AddNode(&Node{ID: "a", Kind: KindSource, Name: "node a again"})
	if err == nil {
		t.Error("expected error for duplicate node ID")
	}
}

func TestAddEdge(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Kind: KindSource, Name: "a"})
	_ = g.AddNode(&Node{ID: "b", Kind: KindSource, Name: "b"})

	err := g.AddEdge(&Edge{
		From: "a", To: "b", Kind: EdgeImports,
		Source: SourceStatic, Confidence: 1.0, Strength: 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.EdgeCount() != 1 {
		t.Errorf("expected 1 edge, got %d", g.EdgeCount())
	}

	from := g.EdgesFrom("a")
	if len(from) != 1 {
		t.Errorf("expected 1 outgoing edge from 'a', got %d", len(from))
	}
	to := g.EdgesTo("b")
	if len(to) != 1 {
		t.Errorf("expected 1 incoming edge to 'b', got %d", len(to))
	}
}

func TestAddEdge_MissingNodes(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Kind: KindSource, Name: "a"})

	err := g.AddEdge(&Edge{From: "a", To: "missing", Kind: EdgeImports})
	if err == nil {
		t.Error("expected error for missing target node")
	}

	err = g.AddEdge(&Edge{From: "missing", To: "a", Kind: EdgeImports})
	if err == nil {
		t.Error("expected error for missing source node")
	}
}

func TestNodesOfKind(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "src1", Kind: KindSource, Name: "src1"})
	_ = g.AddNode(&Node{ID: "src2", Kind: KindSource, Name: "src2"})
	_ = g.AddNode(&Node{ID: "chk1", Kind: KindCheck, Name: "chk1"})

	sources := g.NodesOfKind(KindSource)
	if len(sources) != 2 {
		t.Errorf("expected 2 source nodes, got %d", len(sources))
	}
	checks := g.NodesOfKind(KindCheck)
	if len(checks) != 1 {
		t.Errorf("expected 1 check node, got %d", len(checks))
	}
}

func TestGetNode_NotFound(t *testing.T) {
	g := NewGraph()
	if g.GetNode("nonexistent") != nil {
		t.Error("expected nil for nonexistent node")
	}
}
