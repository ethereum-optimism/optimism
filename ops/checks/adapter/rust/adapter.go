package rust

import "github.com/ethereum-optimism/optimism/ops/checks/graph"

// RustAdapter is a placeholder for Rust crate dependency analysis.
type RustAdapter struct{}

// New returns a new RustAdapter.
func New() *RustAdapter { return &RustAdapter{} }

// Name returns "rust".
func (a *RustAdapter) Name() string { return "rust" }

// Analyze is a no-op stub. Rust support is planned for a future iteration.
func (a *RustAdapter) Analyze(_ *graph.Graph, _ string) error { return nil }
