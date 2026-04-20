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

// TestBuild_WiresDataflowEdges — the builder's only wiring step emits
// `consumes` edges from checks to the source/artifact nodes that match
// their declared inputs. Replaces the pre-cutover tested_by / prereq /
// check→source produces wiring.
func TestBuild_WiresDataflowEdges(t *testing.T) {
	goAdapter := &mockAdapter{
		name: "go",
		nodes: []*graph.Node{
			{ID: "go:example.com/repo/op-node", Kind: graph.KindSource, Name: "op-node",
				Properties: map[string]any{"language": "go"}},
		},
	}

	cat, _ := catalog.Parse([]byte(`check_types:
  - id: go-test
    name: "Go tests"
    kind: test
    language: go
    command: "go test"
    scopeable: true
    scope_type: packages
    avg_duration: 600
    inputs:
      - "**/*.go"
`))

	b := New([]adapter.Adapter{goAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	edges := g.EdgesFrom("check:go-test")
	foundConsumes := false
	for _, e := range edges {
		if e.To == "go:example.com/repo/op-node" && e.Kind == graph.EdgeConsumes {
			foundConsumes = true
		}
	}
	if !foundConsumes {
		t.Errorf("expected check:go-test to have consumes edge to go:example.com/repo/op-node, got edges: %v", edges)
	}
}

// TestBuild_DataflowGivesPrereqOrder — given a catalog where
// forge-build produces forge-artifacts consumed by forge-test,
// graph.CheckPrerequisites derives the prereq order.
func TestBuild_DataflowGivesPrereqOrder(t *testing.T) {
	cat, _ := catalog.Parse([]byte(`check_types:
  - id: forge-build
    name: "Forge build"
    kind: build
    language: solidity
    command: "forge build"
    avg_duration: 180
    outputs:
      - "artifact:forge-artifacts/**"
  - id: forge-test
    name: "Forge tests"
    kind: test
    language: solidity
    command: "forge test"
    scopeable: true
    scope_type: paths
    avg_duration: 3600
    inputs:
      - "artifact:forge-artifacts/**"
`))

	b := New(nil, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	got := graph.CheckPrerequisites(g, "check:forge-test")
	if len(got) != 1 || got[0] != "forge-build" {
		t.Errorf("expected [forge-build], got %v", got)
	}
}

// TestBuild_BridgeImportsGoPackageToArtifact — the builder's bridge
// step emits `imports` edges from go: package nodes whose dir is
// covered by an artifact path prefix to the artifact node itself.
// Enables scoping-layer reverse-walks from invalidated bindings
// artifacts back to the Go packages that consume them.
func TestBuild_BridgeImportsGoPackageToArtifact(t *testing.T) {
	goAdapter := &mockAdapter{
		name: "go",
		nodes: []*graph.Node{
			{
				ID:          "go:example.com/repo/op-e2e/bindings",
				Kind:        graph.KindSource,
				Granularity: "package",
				Properties: map[string]any{
					"language": "go",
					"dir":      "/abs/repo/op-e2e/bindings",
				},
			},
		},
	}

	cat, _ := catalog.Parse([]byte(`check_types:
  - id: gen-go-bindings
    name: "gen-go-bindings"
    kind: gen
    language: go
    command: "gen"
    avg_duration: 60
    outputs:
      - "artifact:op-e2e/bindings/**/*.go"
`))

	b := New([]adapter.Adapter{goAdapter}, cat)
	g, err := b.Build("/fake")
	if err != nil {
		t.Fatal(err)
	}

	foundBridge := false
	for _, e := range g.EdgesFrom("go:example.com/repo/op-e2e/bindings") {
		if e.Kind == graph.EdgeImports && e.To == "artifact:op-e2e/bindings/**/*.go" {
			foundBridge = true
		}
	}
	if !foundBridge {
		t.Errorf("expected imports edge from go:op-e2e/bindings to artifact node; edges: %v",
			g.EdgesFrom("go:example.com/repo/op-e2e/bindings"))
	}
}
