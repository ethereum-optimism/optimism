package diff

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

func TestChangedFiles_ParsesGitOutput(t *testing.T) {
	output := `op-node/rollup/derive/batch_queue.go
op-node/rollup/derive/batch_queue_test.go
packages/contracts-bedrock/src/L1/OptimismPortal2.sol
`
	files := ChangedFiles(output)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestChangedFiles_EmptyInput(t *testing.T) {
	files := ChangedFiles("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestFilesToNodeIDs_GoFile(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "go:github.com/org/repo/op-node/rollup/derive", Kind: graph.KindSource,
		Name: "derive", Properties: map[string]any{"dir": "/repo/op-node/rollup/derive"},
	})

	ids, unknown := FilesToNodeIDs(g, []string{"op-node/rollup/derive/batch_queue.go"})
	if len(ids) != 1 {
		t.Errorf("expected 1 node ID, got %d (unknown: %v)", len(ids), unknown)
	}
}

func TestFilesToNodeIDs_SolidityFile(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "sol:src/L1/OptimismPortal2.sol", Kind: graph.KindSource,
		Name: "OptimismPortal2.sol",
	})

	ids, _ := FilesToNodeIDs(g, []string{"packages/contracts-bedrock/src/L1/OptimismPortal2.sol"})
	if len(ids) != 1 {
		t.Errorf("expected 1 node ID, got %d", len(ids))
	}
	if len(ids) > 0 && ids[0] != "sol:src/L1/OptimismPortal2.sol" {
		t.Errorf("expected sol:src/L1/OptimismPortal2.sol, got %s", ids[0])
	}
}

func TestFilesToNodeIDs_UnknownFile(t *testing.T) {
	g := graph.NewGraph()
	_, unknown := FilesToNodeIDs(g, []string{"README.md"})
	if len(unknown) != 1 {
		t.Errorf("expected 1 unknown file, got %d", len(unknown))
	}
}

func TestBlastRadiusFiles_GoMod(t *testing.T) {
	blast, matches := BlastRadiusFiles([]string{"go.mod"})
	if !blast {
		t.Error("expected go.mod to be blast radius")
	}
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
}

func TestBlastRadiusFiles_CircleCI(t *testing.T) {
	blast, _ := BlastRadiusFiles([]string{".circleci/config.yml"})
	if !blast {
		t.Error("expected .circleci/ to be blast radius")
	}
}

func TestBlastRadiusFiles_NormalFile(t *testing.T) {
	blast, _ := BlastRadiusFiles([]string{"op-node/rollup/derive/batch_queue.go"})
	if blast {
		t.Error("expected normal Go file to NOT be blast radius")
	}
}
