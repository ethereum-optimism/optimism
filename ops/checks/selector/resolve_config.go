package selector

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Config-file diff handling for go.mod and Cargo.toml. Structural
// changes (go version, toolchain, workspace membership, features, etc.)
// language-wide fan out by seeding every node of that language into
// the dataflow walker's invalidated set. Dep-table changes produce
// synthetic changed-node IDs (mod: / rs:) that flow through the same
// walker via imports edges.

// configSeeds carries the synthetic seed nodes derived from a config
// diff. sourceNodes are added directly to the dataflow walker's
// invalidated set. forceAll signals a language-wide structural change
// (e.g. go version bump) — the caller seeds every node of that
// language via allGoSourceNodes / allRustSourceNodes.
type configSeeds struct {
	sourceNodes []string
	forceAll    bool
}

// expandGoModDiffs runs the go.mod analyzer on every go.mod file in
// the diff and returns dataflow seeds. Dep-table changes resolve to
// mod:<path> nodes; structural changes (go, toolchain, require blocks
// the analyzer flags) set forceAll so the caller invalidates every
// Go source node — replacing the old blast-radius fanout.
func expandGoModDiffs(diffs []diff.FileDiff, g *graph.Graph) configSeeds {
	var out configSeeds
	seen := make(map[string]bool)
	for _, d := range diffs {
		change := diff.AnalyzeGoModChange(d)
		if change.ForceBlast {
			out.forceAll = true
		}
		for _, m := range change.AffectedModules {
			modID := "mod:" + m
			if g.GetNode(modID) == nil {
				continue
			}
			if !seen[modID] {
				seen[modID] = true
				out.sourceNodes = append(out.sourceNodes, modID)
			}
			// Reverse-walk imports to pre-seed packages that import
			// the affected module — those packages are the ones whose
			// dataflow consumers (go-test, go-build, …) must fire.
			// Plain mod: seeds wouldn't propagate through consumes
			// edges because no check declares a consumes to a module
			// node.
			for _, e := range g.EdgesTo(modID) {
				if e.Kind != graph.EdgeImports {
					continue
				}
				if !seen[e.From] {
					seen[e.From] = true
					out.sourceNodes = append(out.sourceNodes, e.From)
				}
			}
		}
	}
	return out
}

// expandCargoTomlDiffs runs the Cargo.toml analyzer on each Cargo.toml
// in the diff. Dep-table changes resolve to rs:<crate> (workspace
// member) or mod:<crate> (external) nodes. Feature/toolchain structural
// changes set forceAll. The crate owning the changed Cargo.toml is
// always added since feature-gate-adjacent edits can change behavior
// the analyzer doesn't classify.
func expandCargoTomlDiffs(diffs []diff.FileDiff, g *graph.Graph) configSeeds {
	var out configSeeds
	seen := make(map[string]bool)
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out.sourceNodes = append(out.sourceNodes, id)
	}
	for _, d := range diffs {
		if d.Path != "Cargo.toml" && !strings.HasSuffix(d.Path, "/Cargo.toml") {
			continue
		}
		change := diff.AnalyzeCargoTomlChange(d)
		if change.ForceBlast {
			out.forceAll = true
		}
		for _, dep := range change.AffectedDeps {
			switch {
			case g.GetNode("rs:"+dep) != nil:
				add("rs:" + dep)
				// Also seed crates that import this one.
				for _, e := range g.EdgesTo("rs:" + dep) {
					if e.Kind == graph.EdgeImports {
						add(e.From)
					}
				}
			case g.GetNode("mod:"+dep) != nil:
				add("mod:" + dep)
				for _, e := range g.EdgesTo("mod:" + dep) {
					if e.Kind == graph.EdgeImports {
						add(e.From)
					}
				}
			}
		}
		if owner := findCrateForManifest(d.Path, g); owner != "" {
			add(owner)
		}
	}
	return out
}

// findCrateForManifest finds the crate node whose `dir` matches the
// directory of the given Cargo.toml path. Returns "" if no match.
func findCrateForManifest(cargoTomlPath string, g *graph.Graph) string {
	dir := strings.TrimSuffix(cargoTomlPath, "/Cargo.toml")
	if dir == "Cargo.toml" {
		dir = "."
	}
	for _, node := range g.NodesOfKind(graph.KindSource) {
		if node.Granularity != "crate" {
			continue
		}
		nodeDir, _ := node.Properties["dir"].(string)
		if nodeDir == "" {
			continue
		}
		if nodeDir == dir || strings.HasSuffix(nodeDir, "/"+dir) {
			return node.ID
		}
	}
	return ""
}

// allGoSourceNodes returns every go: source node ID. Used when a
// structural go.mod change invalidates the whole Go toolchain surface.
func allGoSourceNodes(g *graph.Graph) []string {
	var out []string
	for _, n := range g.NodesOfKind(graph.KindSource) {
		if strings.HasPrefix(n.ID, "go:") {
			out = append(out, n.ID)
		}
	}
	return out
}

// allRustSourceNodes returns every rs: source node ID. Used when a
// structural Cargo.toml change invalidates the whole Rust surface.
func allRustSourceNodes(g *graph.Graph) []string {
	var out []string
	for _, n := range g.NodesOfKind(graph.KindSource) {
		if strings.HasPrefix(n.ID, "rs:") {
			out = append(out, n.ID)
		}
	}
	return out
}
