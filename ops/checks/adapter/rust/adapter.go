// Package rust provides a graph adapter for Rust workspaces.
//
// Crates are compilation units, analogous to Go packages. The adapter
// emits:
//   - `rs:<crate>` crate nodes (Granularity=crate) for every workspace
//     member, with their manifest directory stored as a property so
//     freshness can resolve file nodes to filesystem paths.
//   - `rs:<crate>/<rel>.rs` file nodes (Granularity=file) for every
//     .rs file under src/, tests/, benches/, examples/ in each crate.
//
// File nodes are coverage-target leaves: no import edges, no
// containment edges — the ID shape and the crate's `dir` property are
// sufficient to resolve back to disk. Matches the pattern established
// for Go, which matches what Solidity already had.
//
// Intra-workspace crate import edges are not yet emitted. When we
// start consuming Cargo.toml [dependencies] for reachability, they'll
// drop in as `rs:<a> → rs:<b>` imports edges alongside external mod:
// nodes, same shape as Go's internal/external split.
package rust

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/rustmeta"
)

// Crate is re-exported from rustmeta for backwards-compat with tests
// that constructed crates inline. New callers should use rustmeta.Crate
// directly.
type Crate = rustmeta.Crate

// RustAdapter builds crate + file nodes from cargo metadata output.
type RustAdapter struct {
	// WorkspaceDir is the path to the Rust workspace relative to rootDir.
	// Default: "rust".
	WorkspaceDir string

	// CratesFor returns workspace members given an absolute workspace
	// directory. Default shells to `cargo metadata --no-deps` via
	// rustmeta.Load; tests can substitute deterministic data.
	CratesFor rustmeta.Fetcher
}

// New returns a RustAdapter with default settings.
func New() *RustAdapter {
	return &RustAdapter{WorkspaceDir: "rust"}
}

// Name returns "rust".
func (a *RustAdapter) Name() string { return "rust" }

// Analyze walks the Rust workspace at <rootDir>/<WorkspaceDir>,
// enumerates workspace member crates, and emits a crate node plus one
// file node per .rs file for each member. Silently no-ops if the
// workspace directory is missing or cargo metadata fails — callers
// with a Go-only repo shouldn't see errors.
func (a *RustAdapter) Analyze(g *graph.Graph, rootDir string) error {
	workDir := rootDir
	if a.WorkspaceDir != "" {
		workDir = filepath.Join(rootDir, a.WorkspaceDir)
	}
	if _, err := os.Stat(filepath.Join(workDir, "Cargo.toml")); err != nil {
		return nil
	}

	cratesFor := a.CratesFor
	if cratesFor == nil {
		cratesFor = rustmeta.Load
	}
	crates, err := cratesFor(workDir)
	if err != nil {
		// Degrade: a missing cargo shouldn't fail graph construction.
		return nil
	}

	workspaceMembers := make(map[string]bool, len(crates))
	for _, crate := range crates {
		workspaceMembers[crate.Name] = true
	}

	for _, crate := range crates {
		crateNodeID := "rs:" + crate.Name
		_ = g.AddNode(&graph.Node{
			ID:          crateNodeID,
			Kind:        graph.KindSource,
			Granularity: "crate",
			Name:        crate.Name,
			Properties: map[string]any{
				"dir": crate.ManifestDir,
			},
		})

		for _, rel := range findRustFiles(crate.ManifestDir) {
			fileNodeID := crateNodeID + "/" + rel
			_ = g.AddNode(&graph.Node{
				ID:          fileNodeID,
				Kind:        graph.KindSource,
				Granularity: "file",
				Name:        crate.Name + "/" + rel,
				Properties: map[string]any{
					"crate": crateNodeID,
					"dir":   filepath.Dir(filepath.Join(crate.ManifestDir, rel)),
				},
			})
		}
	}

	// Second pass: emit import edges now that all crate + mod nodes exist.
	// Internal deps → crate-to-crate; external deps → mod: nodes (same
	// pattern Go uses for require-block entries).
	for _, crate := range crates {
		fromID := "rs:" + crate.Name
		for _, dep := range crate.Dependencies {
			if workspaceMembers[dep] {
				_ = g.AddEdge(&graph.Edge{
					From:       fromID,
					To:         "rs:" + dep,
					Kind:       graph.EdgeImports,
					Source:     graph.SourceStatic,
					Confidence: 1.0,
					Strength:   0.8,
				})
				continue
			}
			// External dep — ensure a mod: node exists, then edge.
			modID := "mod:" + dep
			_ = g.AddNode(&graph.Node{
				ID:          modID,
				Kind:        graph.KindModule,
				Granularity: "module",
				Name:        dep,
			})
			_ = g.AddEdge(&graph.Edge{
				From:       fromID,
				To:         modID,
				Kind:       graph.EdgeImports,
				Source:     graph.SourceStatic,
				Confidence: 1.0,
				Strength:   0.7,
			})
		}
	}
	return nil
}

// findRustFiles walks crateDir's conventional source directories and
// returns .rs files as paths relative to crateDir. Skips hidden dirs
// (.git, etc.) and the target/ build directory.
func findRustFiles(crateDir string) []string {
	var out []string
	for _, sub := range []string{"src", "tests", "benches", "examples"} {
		root := filepath.Join(crateDir, sub)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				if name == "target" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".rs") {
				return nil
			}
			rel, err := filepath.Rel(crateDir, path)
			if err != nil {
				return nil
			}
			out = append(out, rel)
			return nil
		})
	}
	return out
}
