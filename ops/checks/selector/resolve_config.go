package selector

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// Config-file diff handling for go.mod and Cargo.toml. Structural
// changes (go version, toolchain, workspace membership, features,
// etc.) force blast radius; dep-table changes produce synthetic
// changed-node IDs (mod: / rs:) that flow through the normal
// reverse-walk infrastructure.

// expandGoModDiffs runs the go.mod analyzer on every go.mod file in
// the diff set and returns the combined mod: node IDs plus a
// ForceBlast summary (any single diff forcing blast propagates).
func expandGoModDiffs(diffs []diff.FileDiff) (modIDs []string, forceBlast bool) {
	for _, d := range diffs {
		change := diff.AnalyzeGoModChange(d)
		if change.ForceBlast {
			forceBlast = true
		}
		for _, m := range change.AffectedModules {
			modIDs = append(modIDs, "mod:"+m)
		}
	}
	return modIDs, forceBlast
}

// expandCargoTomlDiffs runs the Cargo.toml analyzer on each Cargo.toml
// in the diff and returns synthetic changed-node IDs: rs:<crate> for
// workspace members, mod:<crate> for external deps. The crate that
// owns the changed Cargo.toml is also added to the changed set, since
// feature-gate-adjacent edits (that the analyzer might not classify)
// can still affect the crate's own behavior.
func expandCargoTomlDiffs(diffs []diff.FileDiff, g *graph.Graph) (ids []string, forceBlast bool) {
	for _, d := range diffs {
		if d.Path != "Cargo.toml" && !strings.HasSuffix(d.Path, "/Cargo.toml") {
			continue
		}
		change := diff.AnalyzeCargoTomlChange(d)
		if change.ForceBlast {
			forceBlast = true
		}
		for _, dep := range change.AffectedDeps {
			if g.GetNode("rs:" + dep) != nil {
				ids = append(ids, "rs:"+dep)
			} else {
				ids = append(ids, "mod:"+dep)
			}
		}
		if owner := findCrateForManifest(d.Path, g); owner != "" {
			ids = append(ids, owner)
		}
	}
	return ids, forceBlast
}

// findCrateForManifest finds the crate node whose `dir` matches the
// directory of the given Cargo.toml path. Returns "" if no match
// (e.g. a workspace-root Cargo.toml with no crate of its own, or the
// crate node was never built).
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

// configBlastPaths returns every config-file path in the diff set
// (go.mod, Cargo.toml). Used as the `files` list when one of those
// diffs forces a full blast radius.
func configBlastPaths(diffs []diff.FileDiff) []string {
	var out []string
	for _, d := range diffs {
		if d.Path == "go.mod" || strings.HasSuffix(d.Path, "/go.mod") ||
			d.Path == "Cargo.toml" || strings.HasSuffix(d.Path, "/Cargo.toml") {
			out = append(out, d.Path)
		}
	}
	return out
}
