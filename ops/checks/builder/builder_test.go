package builder

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// mockAdapter is a test adapter that adds predetermined nodes.
type mockAdapter struct {
	name  string
	nodes []*graph.Node
	edges []*graph.Edge
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Analyze(g *graph.Graph, _ string) error {
	for _, n := range m.nodes {
		_ = g.AddNode(n)
	}
	for _, e := range m.edges {
		_ = g.AddEdge(e)
	}
	return nil
}

func TestBuild_WiresChecksToPackages(t *testing.T) {
	goAdapter := &mockAdapter{
		name: "go",
		nodes: []*graph.Node{
			{ID: "go:example.com/repo/op-node", Kind: graph.KindSource, Granularity: "package", Name: "op-node"},
			{ID: "go:example.com/repo/op-node/rollup", Kind: graph.KindSource, Granularity: "package", Name: "op-node/rollup"},
			{ID: "go:example.com/repo/op-batcher", Kind: graph.KindSource, Granularity: "package", Name: "op-batcher"},
		},
	}

	cat, _ := catalog.Parse([]byte(`checks:
  - id: go-build
    name: "Go build"
    kind: build
    language: go
    command: "go build ./..."
    avg_duration: 120
  - id: go-test-op-node
    name: "Go tests: op-node"
    kind: test
    language: go
    command: "go test ./op-node/..."
    avg_duration: 900
    packages: ["op-node"]
    prerequisites: ["go-build"]
`))

	b := New([]adapter.Adapter{goAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	// Check that op-node and op-node/rollup are wired to the test
	opNodeEdges := g.EdgesFrom("go:example.com/repo/op-node")
	found := false
	for _, e := range opNodeEdges {
		if e.To == "check:go-test-op-node" && e.Kind == graph.EdgeTestedBy {
			found = true
		}
	}
	if !found {
		t.Error("expected tested_by edge from op-node to check:go-test-op-node")
	}

	rollupEdges := g.EdgesFrom("go:example.com/repo/op-node/rollup")
	found = false
	for _, e := range rollupEdges {
		if e.To == "check:go-test-op-node" && e.Kind == graph.EdgeTestedBy {
			found = true
		}
	}
	if !found {
		t.Error("expected tested_by edge from op-node/rollup to check:go-test-op-node")
	}

	// op-batcher should NOT be wired to op-node test
	batcherEdges := g.EdgesFrom("go:example.com/repo/op-batcher")
	for _, e := range batcherEdges {
		if e.To == "check:go-test-op-node" {
			t.Error("op-batcher should not be wired to go-test-op-node")
		}
	}
}

func TestBuild_PrerequisiteEdges(t *testing.T) {
	goAdapter := &mockAdapter{name: "go"}
	cat, _ := catalog.Parse([]byte(`checks:
  - id: go-build
    name: "Go build"
    kind: build
    language: go
    command: "go build ./..."
    avg_duration: 120
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test ./..."
    avg_duration: 600
    prerequisites: ["go-build"]
`))

	b := New([]adapter.Adapter{goAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	prereqs := graph.Prerequisites(g, "check:go-test")
	if len(prereqs) != 1 || prereqs[0] != "check:go-build" {
		t.Errorf("expected prerequisite [check:go-build], got %v", prereqs)
	}
}

func TestMatchesPackage(t *testing.T) {
	tests := []struct {
		importPath string
		pkg        string
		want       bool
	}{
		{"github.com/org/repo/op-node", "op-node", true},
		{"github.com/org/repo/op-node/rollup", "op-node", true},
		{"github.com/org/repo/op-batcher", "op-node", false},
		{"github.com/org/repo/op-node-extra", "op-node", false},
	}

	for _, tt := range tests {
		got := matchesPackage(tt.importPath, tt.pkg)
		if got != tt.want {
			t.Errorf("matchesPackage(%q, %q) = %v, want %v", tt.importPath, tt.pkg, got, tt.want)
		}
	}
}

func TestMatchesDirectory(t *testing.T) {
	tests := []struct {
		solPath string
		dir     string
		want    bool
	}{
		{"src/L1/Foo.sol", "packages/contracts-bedrock/src/L1", true},
		{"src/L1/Foo.sol", "src/L1", true},
		{"src/L2/Bar.sol", "src/L1", false},
		{"test/L1/Foo.t.sol", "packages/contracts-bedrock/test/L1", true},
	}

	for _, tt := range tests {
		got := matchesDirectory(tt.solPath, tt.dir)
		if got != tt.want {
			t.Errorf("matchesDirectory(%q, %q) = %v, want %v", tt.solPath, tt.dir, got, tt.want)
		}
	}
}
