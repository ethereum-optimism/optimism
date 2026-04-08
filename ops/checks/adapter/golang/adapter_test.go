package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func createTestGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// go.mod
	goMod := `module example.com/testmod

go 1.24.0
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)

	// Package A imports B
	os.MkdirAll(filepath.Join(dir, "pkga"), 0755)
	os.WriteFile(filepath.Join(dir, "pkga", "a.go"), []byte(`package pkga

import "example.com/testmod/pkgb"

var _ = pkgb.Hello
`), 0644)

	// Package B (standalone)
	os.MkdirAll(filepath.Join(dir, "pkgb"), 0755)
	os.WriteFile(filepath.Join(dir, "pkgb", "b.go"), []byte(`package pkgb

var Hello = "hello"
`), 0644)

	// Package C with tests that import B
	os.MkdirAll(filepath.Join(dir, "pkgc"), 0755)
	os.WriteFile(filepath.Join(dir, "pkgc", "c.go"), []byte(`package pkgc

var World = "world"
`), 0644)
	os.WriteFile(filepath.Join(dir, "pkgc", "c_test.go"), []byte(`package pkgc

import (
	"testing"
	"example.com/testmod/pkgb"
)

func TestC(t *testing.T) {
	_ = pkgb.Hello
}
`), 0644)

	return dir
}

func TestAnalyze_SmallProject(t *testing.T) {
	dir := createTestGoProject(t)
	g := graph.NewGraph()
	adapter := New()

	if err := adapter.Analyze(g, dir); err != nil {
		t.Fatal(err)
	}

	// Should have 3 package nodes
	nodes := g.NodesOfKind(graph.KindSource)
	if len(nodes) != 3 {
		t.Errorf("expected 3 source nodes, got %d", len(nodes))
		for _, n := range nodes {
			t.Logf("  node: %s", n.ID)
		}
	}

	// pkga should have an import edge to pkgb
	edges := g.EdgesFrom("go:example.com/testmod/pkga")
	found := false
	for _, e := range edges {
		if e.To == "go:example.com/testmod/pkgb" && e.Kind == graph.EdgeImports {
			found = true
		}
	}
	if !found {
		t.Error("expected import edge from pkga to pkgb")
	}
}

func TestAnalyze_ExternalImportsExcluded(t *testing.T) {
	dir := createTestGoProject(t)
	g := graph.NewGraph()
	adapter := New()

	if err := adapter.Analyze(g, dir); err != nil {
		t.Fatal(err)
	}

	// No stdlib or external packages should be nodes
	for _, n := range g.NodesOfKind(graph.KindSource) {
		if n.ID == "go:fmt" || n.ID == "go:testing" {
			t.Errorf("stdlib package should not be a node: %s", n.ID)
		}
	}
}

func TestAnalyze_TestImports(t *testing.T) {
	dir := createTestGoProject(t)
	g := graph.NewGraph()
	adapter := New()

	if err := adapter.Analyze(g, dir); err != nil {
		t.Fatal(err)
	}

	// pkgc should have a test import edge to pkgb
	edges := g.EdgesFrom("go:example.com/testmod/pkgc")
	found := false
	for _, e := range edges {
		if e.To == "go:example.com/testmod/pkgb" && e.Kind == graph.EdgeImports {
			if props, ok := e.Properties["test_import"]; ok && props.(bool) {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected test import edge from pkgc to pkgb")
	}
}

func TestName(t *testing.T) {
	a := New()
	if a.Name() != "go" {
		t.Errorf("expected name 'go', got %q", a.Name())
	}
}
