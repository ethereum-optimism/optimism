package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "src", Kind: KindSource, Granularity: "package", Name: "op-node"})
	_ = g.AddNode(&Node{ID: "chk", Kind: KindCheck, Name: "go-test-op-node", Properties: map[string]any{
		"avg_duration": 900,
	}})
	_ = g.AddEdge(&Edge{
		From: "chk", To: "src", Kind: EdgeConsumes,
		Source: SourceStatic, Confidence: 1.0, Strength: 1.0,
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")

	if err := Save(g, path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", loaded.NodeCount())
	}
	if loaded.EdgeCount() != 1 {
		t.Errorf("expected 1 edge, got %d", loaded.EdgeCount())
	}

	node := loaded.GetNode("chk")
	if node == nil {
		t.Fatal("expected to find check node")
	}
	if node.Name != "go-test-op-node" {
		t.Errorf("expected name 'go-test-op-node', got %q", node.Name)
	}
	dur, ok := node.Properties["avg_duration"]
	if !ok {
		t.Fatal("expected avg_duration property")
	}
	// JSON unmarshals numbers as float64
	if dur.(float64) != 900 {
		t.Errorf("expected avg_duration=900, got %v", dur)
	}

	// Verify indexes rebuilt (edge is chk → src; consumes direction).
	edges := loaded.EdgesFrom("chk")
	if len(edges) != 1 {
		t.Errorf("expected 1 outgoing edge from 'chk', got %d", len(edges))
	}
	edges = loaded.EdgesTo("src")
	if len(edges) != 1 {
		t.Errorf("expected 1 incoming edge to 'src', got %d", len(edges))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/graph.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(path, []byte("not json"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
