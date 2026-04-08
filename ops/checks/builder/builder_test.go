package builder

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/adapter"
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

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

func TestBuild_WiresGoSourceToGoTest(t *testing.T) {
	goAdapter := &mockAdapter{
		name: "go",
		nodes: []*graph.Node{
			{ID: "go:example.com/repo/op-node", Kind: graph.KindSource, Name: "op-node"},
			{ID: "go:example.com/repo/op-batcher", Kind: graph.KindSource, Name: "op-batcher"},
		},
	}

	cat, _ := catalog.Parse([]byte(`check_types:
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
    command: "go test"
    scopeable: true
    scope_type: packages
    avg_duration: 600
    prerequisites: ["go-build"]
`))

	b := New([]adapter.Adapter{goAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	// Both Go packages should connect to check:go-test (not separate checks)
	for _, nodeID := range []string{"go:example.com/repo/op-node", "go:example.com/repo/op-batcher"} {
		edges := g.EdgesFrom(nodeID)
		foundGoTest := false
		for _, e := range edges {
			if e.To == "check:go-test" && e.Kind == graph.EdgeTestedBy {
				foundGoTest = true
			}
		}
		if !foundGoTest {
			t.Errorf("expected %s to have tested_by edge to check:go-test", nodeID)
		}
	}

	// Should NOT have check:go-test-op-node (old discrete style)
	if g.GetNode("check:go-test-op-node") != nil {
		t.Error("should not have discrete check nodes")
	}
}

func TestBuild_WiresSolSourceToForgeTest(t *testing.T) {
	solAdapter := &mockAdapter{
		name: "solidity",
		nodes: []*graph.Node{
			{ID: "sol:src/L1/Foo.sol", Kind: graph.KindSource, Name: "Foo.sol", Properties: map[string]any{"language": "solidity"}},
		},
	}

	cat, _ := catalog.Parse([]byte(`check_types:
  - id: forge-build
    name: "Forge build"
    kind: build
    language: solidity
    command: "forge build"
    avg_duration: 180
  - id: forge-test
    name: "Forge tests"
    kind: test
    language: solidity
    command: "forge test"
    scopeable: true
    scope_type: paths
    avg_duration: 3600
    prerequisites: ["forge-build"]
`))

	b := New([]adapter.Adapter{solAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	edges := g.EdgesFrom("sol:src/L1/Foo.sol")
	foundForgeTest := false
	for _, e := range edges {
		if e.To == "check:forge-test" && e.Kind == graph.EdgeTestedBy {
			foundForgeTest = true
		}
	}
	if !foundForgeTest {
		t.Error("expected sol:src/L1/Foo.sol to have tested_by edge to check:forge-test")
	}
}

func TestBuild_PrerequisiteEdges(t *testing.T) {
	cat, _ := catalog.Parse([]byte(`check_types:
  - id: go-build
    name: "Go build"
    kind: build
    language: go
    command: "go build"
    avg_duration: 120
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test"
    avg_duration: 600
    prerequisites: ["go-build"]
`))

	b := New(nil, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	prereqs := graph.Prerequisites(g, "check:go-test")
	if len(prereqs) != 1 || prereqs[0] != "check:go-build" {
		t.Errorf("expected prerequisite [check:go-build], got %v", prereqs)
	}
}
