package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// createTestRustWorkspace lays out a two-crate workspace on disk with
// .rs files under src/ and tests/, so tests can exercise the adapter's
// filesystem walk without running cargo.
func createTestRustWorkspace(t *testing.T) (root, workDir string) {
	t.Helper()
	root = t.TempDir()
	workDir = filepath.Join(root, "rust")

	// Crate A: src/lib.rs + tests/smoke.rs
	crateA := filepath.Join(workDir, "crates", "alpha")
	writeFile(t, filepath.Join(crateA, "Cargo.toml"), `[package]
name = "alpha"
version = "0.1.0"
`)
	writeFile(t, filepath.Join(crateA, "src", "lib.rs"), "pub fn a() {}\n")
	writeFile(t, filepath.Join(crateA, "tests", "smoke.rs"), "#[test] fn s() {}\n")

	// Crate B: src/lib.rs only
	crateB := filepath.Join(workDir, "crates", "beta")
	writeFile(t, filepath.Join(crateB, "Cargo.toml"), `[package]
name = "beta"
version = "0.1.0"
`)
	writeFile(t, filepath.Join(crateB, "src", "lib.rs"), "pub fn b() {}\n")

	// Workspace root.
	writeFile(t, filepath.Join(workDir, "Cargo.toml"), `[workspace]
members = ["crates/alpha", "crates/beta"]
`)

	return root, workDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestAnalyze_WorkspaceMembers — crate and file nodes are emitted for
// every member, with Granularity split and manifest-dir stored on the
// crate node so freshness can resolve file nodes later.
func TestAnalyze_WorkspaceMembers(t *testing.T) {
	root, workDir := createTestRustWorkspace(t)

	g := graph.NewGraph()
	a := &RustAdapter{
		WorkspaceDir: "rust",
		CratesFor: func(wd string) ([]Crate, error) {
			return []Crate{
				{Name: "alpha", ManifestDir: filepath.Join(wd, "crates", "alpha")},
				{Name: "beta", ManifestDir: filepath.Join(wd, "crates", "beta")},
			}, nil
		},
	}
	if err := a.Analyze(g, root); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Crate nodes exist with Granularity=crate and manifest dir stored.
	alpha := g.GetNode("rs:alpha")
	if alpha == nil {
		t.Fatal("missing rs:alpha crate node")
	}
	if alpha.Granularity != "crate" {
		t.Errorf("rs:alpha Granularity=%q, want crate", alpha.Granularity)
	}
	if dir, _ := alpha.Properties["dir"].(string); dir != filepath.Join(workDir, "crates", "alpha") {
		t.Errorf("rs:alpha dir=%q, want %q", dir, filepath.Join(workDir, "crates", "alpha"))
	}

	// File nodes under each crate.
	cases := []string{
		"rs:alpha/src/lib.rs",
		"rs:alpha/tests/smoke.rs",
		"rs:beta/src/lib.rs",
	}
	for _, id := range cases {
		n := g.GetNode(id)
		if n == nil {
			t.Errorf("missing file node %s", id)
			continue
		}
		if n.Granularity != "file" {
			t.Errorf("%s Granularity=%q, want file", id, n.Granularity)
		}
	}

	// File nodes must not have outgoing edges from the adapter
	// (coverage-target leaves, like Go file nodes).
	if edges := g.EdgesFrom("rs:alpha/src/lib.rs"); len(edges) != 0 {
		t.Errorf("file node should have no outgoing edges, got %d", len(edges))
	}
}

// TestAnalyze_MissingWorkspace — no Cargo.toml at WorkspaceDir → silent no-op.
func TestAnalyze_MissingWorkspace(t *testing.T) {
	root := t.TempDir()
	g := graph.NewGraph()
	if err := New().Analyze(g, root); err != nil {
		t.Errorf("expected silent no-op, got: %v", err)
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
}

// TestAnalyze_CargoMetadataFailure — CratesFor returning an error is
// degraded-silent, not fatal.
func TestAnalyze_CargoMetadataFailure(t *testing.T) {
	root, _ := createTestRustWorkspace(t)
	g := graph.NewGraph()
	a := &RustAdapter{
		WorkspaceDir: "rust",
		CratesFor:    func(string) ([]Crate, error) { return nil, os.ErrPermission },
	}
	if err := a.Analyze(g, root); err != nil {
		t.Errorf("expected silent degrade, got: %v", err)
	}
	// No crate nodes emitted; the Cargo.toml workspace-root exists so
	// the gate passed, but no metadata means no members.
	if len(g.NodesOfKind(graph.KindSource)) != 0 {
		t.Errorf("expected 0 source nodes, got %d", len(g.NodesOfKind(graph.KindSource)))
	}
}

func TestName(t *testing.T) {
	if New().Name() != "rust" {
		t.Errorf("Name() = %q, want rust", New().Name())
	}
}

// TestAnalyze_CrateDepEdges — internal deps emit crate-to-crate
// edges, external deps emit mod: nodes + crate-to-mod edges. Same
// pattern Go uses for go.mod requires.
func TestAnalyze_CrateDepEdges(t *testing.T) {
	root, workDir := createTestRustWorkspace(t)

	g := graph.NewGraph()
	a := &RustAdapter{
		WorkspaceDir: "rust",
		CratesFor: func(wd string) ([]Crate, error) {
			return []Crate{
				{
					Name:        "alpha",
					ManifestDir: filepath.Join(wd, "crates", "alpha"),
					// alpha depends on beta (internal) and serde (external)
					Dependencies: []string{"beta", "serde"},
				},
				{
					Name:         "beta",
					ManifestDir:  filepath.Join(wd, "crates", "beta"),
					Dependencies: nil,
				},
			}, nil
		},
	}
	if err := a.Analyze(g, root); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Internal dep: rs:alpha → rs:beta.
	foundInternal := false
	for _, e := range g.EdgesFrom("rs:alpha") {
		if e.Kind == graph.EdgeImports && e.To == "rs:beta" {
			foundInternal = true
		}
	}
	if !foundInternal {
		t.Error("expected rs:alpha → rs:beta imports edge (internal dep)")
	}

	// External dep: rs:alpha → mod:serde, and mod:serde node exists.
	if g.GetNode("mod:serde") == nil {
		t.Error("expected mod:serde node for external dep")
	}
	foundExternal := false
	for _, e := range g.EdgesFrom("rs:alpha") {
		if e.Kind == graph.EdgeImports && e.To == "mod:serde" {
			foundExternal = true
		}
	}
	if !foundExternal {
		t.Error("expected rs:alpha → mod:serde imports edge (external dep)")
	}

	_ = workDir
}
