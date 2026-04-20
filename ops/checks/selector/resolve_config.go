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
	for _, d := range diffs {
		change := diff.AnalyzeGoModChange(d)
		if change.ForceBlast {
			out.forceAll = true
		}
		for _, m := range change.AffectedModules {
			id := "mod:" + m
			if g.GetNode(id) != nil {
				out.sourceNodes = append(out.sourceNodes, id)
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
	for _, d := range diffs {
		if d.Path != "Cargo.toml" && !strings.HasSuffix(d.Path, "/Cargo.toml") {
			continue
		}
		change := diff.AnalyzeCargoTomlChange(d)
		if change.ForceBlast {
			out.forceAll = true
		}
		for _, dep := range change.AffectedDeps {
			if g.GetNode("rs:" + dep) != nil {
				out.sourceNodes = append(out.sourceNodes, "rs:"+dep)
			} else if g.GetNode("mod:" + dep) != nil {
				out.sourceNodes = append(out.sourceNodes, "mod:"+dep)
			}
		}
		if owner := findCrateForManifest(d.Path, g); owner != "" {
			out.sourceNodes = append(out.sourceNodes, owner)
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
