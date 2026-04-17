package selector

import (
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// Resolve is Phase 1 of selection. Given the diff and the graph, it
// produces the complete set of Candidates — every (check, scope,
// profile) triple that has any non-zero evidence of relevance, with
// per-source provenance attached.
//
// Resolve owns all graph access and all diff interpretation. Phase 2
// (Optimize) consumes only Candidates plus policy.
//
// Resolution strategies, in priority order per check type:
//  1. Blast radius: if any changed file matches a blast-radius pattern,
//     every check becomes an unscoped Candidate with signal=1.0.
//  2. Trigger match (non-scopeable checks): glob against changed paths.
//  3. Coverage-based (scopeable checks with coverage edges): emit one
//     Candidate per (test, profile) whose coverage line ranges intersect
//     the diff's changed lines.
//  4. Import-based fallback (scopeable or binary): walk the static
//     import graph from changed nodes.
//
// Concerns split across:
//
//	resolve.go              — orchestration + shared helpers
//	resolve_config.go       — go.mod / Cargo.toml dep-change expansion
//	resolve_coverage.go     — coverage-edge → (test, profile) candidates
//	resolve_import.go       — import-edge reverse-walk fallback
//	resolve_correlation.go  — CI-history correlation edges
func Resolve(
	g *graph.Graph,
	diffs []diff.FileDiff,
	cat *catalog.Catalog,
	pol *policy.Policy,
	fresh freshness.Checker,
) []Candidate {
	if fresh == nil {
		fresh = freshness.Nop()
	}
	filePaths := extractPaths(diffs)
	if len(filePaths) == 0 {
		return nil
	}

	// go.mod and Cargo.toml get special handling: structural changes
	// force blast radius; dep-table changes become synthetic changed
	// nodes (mod: for external deps, go:/rs: for internal) that feed
	// the reverse-walk infrastructure the same way test-helper
	// changes do.
	goModIDs, goForce := expandGoModDiffs(diffs)
	cargoIDs, cargoForce := expandCargoTomlDiffs(diffs, g)
	if goForce || cargoForce {
		return blastRadiusCandidates(cat, configBlastPaths(diffs))
	}

	if isBlast, files := diff.BlastRadiusFiles(filePaths, pol.BlastRadius); isBlast {
		return blastRadiusCandidates(cat, files)
	}

	changedLines := buildChangedLinesMap(diffs)
	fileChangedIDs, _ := diff.FilesToNodeIDs(g, filePaths)
	changedIDs := make([]string, 0, len(fileChangedIDs)+len(goModIDs)+len(cargoIDs))
	changedIDs = append(changedIDs, fileChangedIDs...)
	for _, id := range goModIDs {
		if g.GetNode(id) != nil {
			changedIDs = append(changedIDs, id)
		}
	}
	for _, id := range cargoIDs {
		if g.GetNode(id) != nil {
			changedIDs = append(changedIDs, id)
		}
	}

	var out []Candidate
	for i := range cat.CheckTypes {
		ct := &cat.CheckTypes[i]
		out = append(out, candidatesForCheck(g, ct, filePaths, changedIDs, changedLines, pol, fresh)...)
	}
	out = append(out, profileTriggerCandidates(cat, filePaths)...)
	return out
}

// profileTriggerCandidates emits one unscoped Candidate per scopeable
// check × triggered profile, signal=1.0. Covers the case where a
// feature-flag profile only runs if specific paths changed (e.g.
// opcm_v2 profile → OPContractsManager files), and coverage data
// wasn't collected under that profile so no coverage edge carries it.
// Without this, such profiles stay at 0% recall on real diffs.
func profileTriggerCandidates(cat *catalog.Catalog, filePaths []string) []Candidate {
	var out []Candidate
	for i := range cat.Profiles {
		p := &cat.Profiles[i]
		if !p.MatchesTriggers(filePaths) {
			continue
		}
		for j := range cat.CheckTypes {
			ct := &cat.CheckTypes[j]
			if !ct.Scopeable {
				continue
			}
			out = append(out, Candidate{
				CheckID: ct.ID,
				Profile: p.Name,
				Signal:  1.0,
				Provenance: []SignalContribution{{
					Source:       graph.SourceStatic,
					EdgeKind:     graph.EdgeKind(""),
					Contribution: 1.0,
					Raw: map[string]any{
						"reason":   "profile_trigger",
						"profile":  p.Name,
						"triggers": p.Triggers,
					},
				}},
			})
		}
	}
	return out
}

// candidatesForCheck produces Candidates for a single check type,
// picking the best resolution strategy available.
func candidatesForCheck(
	g *graph.Graph,
	ct *catalog.CheckType,
	filePaths []string,
	changedIDs []string,
	changedLines map[string]map[int]bool,
	pol *policy.Policy,
	fresh freshness.Checker,
) []Candidate {
	// Trigger match: for non-scopeable checks, a glob hit makes it
	// a candidate with signal=1.0 regardless of graph reachability.
	if len(ct.Triggers) > 0 && ct.MatchesTriggers(filePaths) {
		return []Candidate{{
			CheckID: ct.ID,
			Signal:  1.0,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeKind(""),
				Contribution: 1.0,
				Raw: map[string]any{
					"reason":   "trigger",
					"triggers": ct.Triggers,
				},
			}},
		}}
	}

	if ct.Scopeable {
		if cands := coverageCandidates(g, ct, changedIDs, changedLines, pol, fresh); len(cands) > 0 {
			return cands
		}
		return importScopeCandidates(g, ct, changedIDs)
	}

	// Binary check: aggregate historical correlation (if any) with
	// structural import reachability, taking the stronger signal and
	// union-ing their provenance.
	imports := importBinaryCandidates(g, ct, changedIDs)
	correlation := correlationCandidates(g, ct, changedIDs, fresh)
	return mergeBinaryCandidates(imports, correlation)
}

// blastRadiusCandidates emits one unscoped Candidate per check type,
// with signal=1.0 and provenance pointing at the triggering blast-radius
// files. Blast radius is the "something upstream of everything changed"
// escape hatch — CI config, toolchain, go.mod, foundry.toml, etc.
//
// For scopeable checks with profiles (e.g. forge-test), emits one
// candidate per profile so the full feature matrix runs. Otherwise a
// .circleci/ change would only run forge-test:main, silently dropping
// the feature-profile safety net.
func blastRadiusCandidates(cat *catalog.Catalog, files []string) []Candidate {
	out := make([]Candidate, 0, len(cat.CheckTypes))
	mkProv := func() []SignalContribution {
		return []SignalContribution{{
			Source:       graph.SourceStatic,
			EdgeKind:     graph.EdgeKind(""),
			Contribution: 1.0,
			Raw: map[string]any{
				"reason": "blast_radius",
				"files":  files,
			},
		}}
	}
	for i := range cat.CheckTypes {
		ct := &cat.CheckTypes[i]
		out = append(out, Candidate{
			CheckID:    ct.ID,
			Signal:     1.0,
			Provenance: mkProv(),
		})
		if !ct.Scopeable || len(cat.Profiles) == 0 {
			continue
		}
		for j := range cat.Profiles {
			p := &cat.Profiles[j]
			if p.Name == "" || p.Name == "main" {
				continue
			}
			out = append(out, Candidate{
				CheckID:    ct.ID,
				Profile:    p.Name,
				Signal:     1.0,
				Provenance: mkProv(),
			})
		}
	}
	return out
}

// --- shared helpers used by multiple resolve_*.go files ---

func extractPaths(diffs []diff.FileDiff) []string {
	var paths []string
	for _, d := range diffs {
		if d.Path != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths
}

// buildChangedLinesMap extracts the set of changed line numbers per
// (repo-relative, or Solidity contracts-bedrock-relative) source path.
func buildChangedLinesMap(diffs []diff.FileDiff) map[string]map[int]bool {
	result := make(map[string]map[int]bool)
	for _, d := range diffs {
		if d.Path == "" || len(d.Hunks) == 0 {
			continue
		}
		path := d.Path
		if strings.HasPrefix(path, "packages/contracts-bedrock/") {
			path = strings.TrimPrefix(path, "packages/contracts-bedrock/")
		}
		lines := make(map[int]bool)
		for _, h := range d.Hunks {
			for i := h.NewStart; i < h.NewStart+h.NewCount; i++ {
				lines[i] = true
			}
		}
		if len(lines) > 0 {
			result[path] = lines
		}
	}
	return result
}

// isCandidateTestNode reports whether a node can be a scope candidate
// for this check. Solidity: test files (test/…/*.t.sol). Go: any
// package node — tests live inside packages, and `go test ./pkg/...`
// handles packages without _test.go files gracefully.
func isCandidateTestNode(nodeID string, ct *catalog.CheckType) bool {
	switch ct.Language {
	case "solidity":
		return isTestFileNode(nodeID)
	case "go":
		return strings.HasPrefix(nodeID, "go:")
	}
	return false
}

// scopeForCandidate derives the command-line scope for a candidate.
// The Solidity-specific test-path mapping is only honored when the
// check itself is Solidity — otherwise we'd emit commands like
// `go test ./test/L1/X.t.sol` that don't parse. For other languages
// we use the scope_type-driven derivation.
func scopeForCandidate(nodeID string, ct *catalog.CheckType) string {
	if ct.Language == "solidity" {
		if s := nodeIDToTestPath(nodeID); s != "" {
			return s
		}
	}
	return nodeIDToScope(nodeID, ct.ScopeType)
}

// testNodeMatchesLanguage reports whether nodeID's prefix is the one
// the check's language uses for test-side nodes. Used to reject
// cross-language coverage edges that would otherwise produce
// nonsensical scopes.
func testNodeMatchesLanguage(nodeID, language string) bool {
	switch language {
	case "solidity":
		return strings.HasPrefix(nodeID, "sol:")
	case "go":
		return strings.HasPrefix(nodeID, "go:")
	case "rust":
		return strings.HasPrefix(nodeID, "rs:")
	}
	return false
}

// isTestFileNode reports whether a node ID identifies a Solidity
// test file (path starts with test/ and ends with .t.sol).
func isTestFileNode(nodeID string) bool {
	if !strings.HasPrefix(nodeID, "sol:") {
		return false
	}
	path := strings.TrimPrefix(nodeID, "sol:")
	return strings.HasPrefix(path, "test/") && strings.HasSuffix(path, ".t.sol")
}

// hasTestedByEdge reports whether nodeID has a tested_by edge to
// checkNodeID. Used to filter scope candidates to checks that actually
// apply to a given test file's language.
func hasTestedByEdge(g *graph.Graph, nodeID, checkNodeID string) bool {
	for _, edge := range g.EdgesFrom(nodeID) {
		if edge.Kind == graph.EdgeTestedBy && edge.To == checkNodeID {
			return true
		}
	}
	return false
}

// profileFromEdge extracts the profile name from a coverage/correlation
// edge's properties, or "" if unset.
func profileFromEdge(edge *graph.Edge) string {
	if edge.Properties == nil {
		return ""
	}
	if p, ok := edge.Properties["profile"].(string); ok {
		return p
	}
	return ""
}

// nodeIDToTestPath returns a specific Solidity test file scope
// (./test/L1/X.t.sol) for a sol: test-file node, or "" for anything
// else. Preferred over the directory-glob derivation because it
// scopes to a single file.
func nodeIDToTestPath(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		path := strings.TrimPrefix(nodeID, "sol:")
		if strings.HasPrefix(path, "test/") && strings.HasSuffix(path, ".t.sol") {
			return "./" + path
		}
	}
	return ""
}

// countLineHits returns how many lines from line_ranges fall inside
// the changedLines set. Used by coverage aggregation to compute the
// hit fraction.
func countLineHits(lineRanges any, changedLines map[int]bool) int {
	hits := 0
	diff.WalkLineRanges(lineRanges, func(line int) {
		if changedLines[line] {
			hits++
		}
	})
	return hits
}

// nodeIDToSourceFile extracts the source file path component from a
// sol:/go: node ID. Used when matching against changedLines.
func nodeIDToSourceFile(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		return strings.TrimPrefix(nodeID, "sol:")
	}
	if strings.HasPrefix(nodeID, "go:") {
		return strings.TrimPrefix(nodeID, "go:")
	}
	return nodeID
}

// nodeIDToScope converts a source node ID to a command-line scope
// argument based on the check's scope_type. Solidity src/L1/X.sol
// becomes ./test/L1/*; a Go package node becomes ./pkg/....
func nodeIDToScope(nodeID, scopeType string) string {
	switch scopeType {
	case "packages":
		if strings.HasPrefix(nodeID, "go:") {
			path := strings.TrimPrefix(nodeID, "go:")
			parts := strings.SplitN(path, "/", 4)
			if len(parts) >= 4 {
				return "./" + parts[3] + "/..."
			}
			return "./" + path + "/..."
		}
	case "paths":
		if strings.HasPrefix(nodeID, "sol:") {
			path := strings.TrimPrefix(nodeID, "sol:")
			if strings.HasPrefix(path, "src/") {
				dir := strings.TrimPrefix(path, "src/")
				dir = strings.Split(dir, "/")[0]
				return "./test/" + dir + "/*"
			}
			if strings.HasPrefix(path, "test/") {
				dir := strings.TrimPrefix(path, "test/")
				dir = strings.Split(dir, "/")[0]
				return "./test/" + dir + "/*"
			}
		}
	}
	return ""
}
