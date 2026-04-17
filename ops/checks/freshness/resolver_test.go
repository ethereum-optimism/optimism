package freshness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// TestPathToNodeID_Solidity — packages/contracts-bedrock/... paths
// round-trip via PathToNodeID → Resolve.
func TestPathToNodeID_Solidity(t *testing.T) {
	r := NewResolver("", nil) // sol: doesn't need any state
	got := r.PathToNodeID("packages/contracts-bedrock/test/L1/X.t.sol")
	want := "sol:test/L1/X.t.sol"
	if got != want {
		t.Errorf("PathToNodeID = %q, want %q", got, want)
	}
	// Round-trip back.
	if rel := r.Resolve(got); rel != "packages/contracts-bedrock/test/L1/X.t.sol" {
		t.Errorf("round-trip Resolve = %q", rel)
	}
}

// TestPathToNodeID_Go — Go file paths get the module prefix.
func TestPathToNodeID_Go(t *testing.T) {
	root := t.TempDir()
	goMod := "module github.com/acme/widgets\n\ngo 1.24.0\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	r := NewResolver(root, nil)
	got := r.PathToNodeID("widget/core/thing.go")
	want := "go:github.com/acme/widgets/widget/core/thing.go"
	if got != want {
		t.Errorf("PathToNodeID = %q, want %q", got, want)
	}
}

// TestPathToNodeID_GoWithoutGoMod — no module path available, Go
// paths don't resolve.
func TestPathToNodeID_GoWithoutGoMod(t *testing.T) {
	r := NewResolver(t.TempDir(), nil) // no go.mod in tempdir
	if got := r.PathToNodeID("foo.go"); got != "" {
		t.Errorf("PathToNodeID = %q, want \"\" (no module path)", got)
	}
}

// TestPathToNodeID_Rust — Rust file paths look up the containing
// crate via the graph and produce rs:<crate>/<rel>.
func TestPathToNodeID_Rust(t *testing.T) {
	root := t.TempDir()
	crateDir := filepath.Join(root, "rust", "crates", "kona-derive")

	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "rs:kona-derive", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": crateDir},
	})

	r := NewResolver(root, g)
	got := r.PathToNodeID("rust/crates/kona-derive/src/lib.rs")
	want := "rs:kona-derive/src/lib.rs"
	if got != want {
		t.Errorf("PathToNodeID = %q, want %q", got, want)
	}
}

// TestPathToNodeID_RustOutsideCrates — a Rust path outside every
// known crate's manifest dir doesn't resolve.
func TestPathToNodeID_RustOutsideCrates(t *testing.T) {
	root := t.TempDir()
	g := graph.NewGraph()
	_ = g.AddNode(&graph.Node{
		ID: "rs:alpha", Kind: graph.KindSource, Granularity: "crate",
		Properties: map[string]any{"dir": filepath.Join(root, "rust", "alpha")},
	})

	r := NewResolver(root, g)
	if got := r.PathToNodeID("rust/other/src/lib.rs"); got != "" {
		t.Errorf("PathToNodeID = %q, want \"\" (outside every crate)", got)
	}
}

// TestPathToNodeID_UnknownExtension — non-sol/go/rs paths don't
// resolve even with a fully-configured resolver.
func TestPathToNodeID_UnknownExtension(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x.y/z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(root, graph.NewGraph())
	if got := r.PathToNodeID("README.md"); got != "" {
		t.Errorf("PathToNodeID = %q, want \"\" (unknown extension)", got)
	}
}
