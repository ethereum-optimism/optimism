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
	// Collect per-check candidates; also build a map from CheckID to its
	// scope candidates so profile triggers can reuse the same scopes
	// (same file → same tests, just under a different profile env).
	byCheck := make(map[string][]Candidate)
	for i := range cat.CheckTypes {
		ct := &cat.CheckTypes[i]
		cands := candidatesForCheck(g, ct, filePaths, changedIDs, changedLines, pol, fresh)
		out = append(out, cands...)
		if ct.Scopeable {
			byCheck[ct.ID] = cands
		}
	}
	out = append(out, profileTriggerCandidates(cat, filePaths, byCheck)...)
	return out
}

// mergeScopedCandidates unions two candidate sets by (scope, profile),
// keeping the higher-signal candidate when both sides produce one for
// the same key. Provenance from both sources is preserved on the kept
// candidate so `explain --why` shows how each match was derived.
func mergeScopedCandidates(a, b []Candidate) []Candidate {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	type key struct{ scope, profile string }
	byKey := make(map[key]*Candidate, len(a)+len(b))
	add := func(c Candidate) {
		k := key{c.Scope, c.Profile}
		cur, ok := byKey[k]
		if !ok {
			c2 := c
			byKey[k] = &c2
			return
		}
		if c.Signal > cur.Signal {
			cur.Signal = c.Signal
		}
		cur.Provenance = append(cur.Provenance, c.Provenance...)
	}
	for _, c := range a {
		add(c)
	}
	for _, c := range b {
		add(c)
	}
	out := make([]Candidate, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, *c)
	}
	return out
}

// profileTriggerCandidates emits per-profile candidates for every
// scopeable check whose profile-matrix should run against this diff.
//
// Scopes are cloned from the same check's main-profile candidates
// (computed from coverage/import walk): if only two test files import
// the changed source, only those two files run — under every triggered
// profile. Without this, profile triggers fire unscoped `:all`
// candidates that run the entire test suite under each feature flag,
// which wipes out the selector's savings on contracts diffs.
//
// If the main-profile scoping found nothing (e.g. the changed file has
// no importers and no coverage edges), we fall back to a single
// unscoped candidate per profile — there's no scope data to reuse but
// the profile trigger still means "run this feature matrix as a safety
// net". Callers that want to suppress that safety net can rely on the
// optimizer's should_run math (miss_cost × prior vs run_cost).
func profileTriggerCandidates(cat *catalog.Catalog, filePaths []string, byCheck map[string][]Candidate) []Candidate {
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
			mainCands := byCheck[ct.ID]
			// Mirror whatever the main-profile candidate set looks like:
			// if main is empty (no coverage, no importers, no trigger),
			// there's nothing to test — profile triggers should NOT
			// invent work. If main has scoped candidates, clone them.
			// If main has only an unscoped trigger candidate, clone
			// that as the per-profile safety net.
			var mainProfileCands []Candidate
			for _, c := range mainCands {
				if c.Profile == "" {
					mainProfileCands = append(mainProfileCands, c)
				}
			}
			if len(mainProfileCands) == 0 {
				continue
			}
			for _, c := range mainProfileCands {
				clone := c
				clone.Profile = p.Name
				clone.Provenance = []SignalContribution{{
					Source:       graph.SourceStatic,
					EdgeKind:     graph.EdgeKind(""),
					Contribution: c.Signal,
					Raw: map[string]any{
						"reason":          "profile_trigger",
						"profile":         p.Name,
						"triggers":        p.Triggers,
						"scope_from_main": true,
					},
				}}
				out = append(out, clone)
			}
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
	triggerHit := len(ct.Triggers) > 0 && ct.MatchesTriggers(filePaths)

	// Scopeable checks always try scoping first, even when a trigger hits.
	// A trigger-only unscoped candidate (e.g. `rust-test:all`) is a last
	// resort — if the graph can localize the work to specific packages or
	// crates via coverage/import walks, those scoped Candidates carry the
	// same signal=1.0 but let the optimizer split into tiers and emit
	// targeted commands like `just test -p kona-derive`.
	if ct.Scopeable {
		// Union coverage and import-walk candidates. Coverage gives
		// runtime evidence (which tests actually exercise the changed
		// lines); import-walk gives compile-time evidence (which tests
		// depend on the changed file for their build). Interface-only
		// tests live in the second bucket: a test `import "IFoo.sol"`
		// doesn't show up in coverage for src/Foo.sol unless it
		// exercises the impl at runtime, but its build breaks if the
		// interface shape shifts. The adapter emits src → interface
		// `generates` edges and transitiveConsumers follows them, so
		// import-walk surfaces these tests.
		cov := coverageCandidates(g, ct, changedIDs, changedLines, pol, fresh)
		imp := importScopeCandidates(g, ct, changedIDs)
		if merged := mergeScopedCandidates(cov, imp); len(merged) > 0 {
			return merged
		}
		// Fall through to trigger-based unscoped candidate if graph gave
		// us nothing — better to run the whole check than miss it.
	}

	if triggerHit {
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
		return nil
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
// for this check. Solidity: test files (test/…/*.t.sol). Go: package
// nodes only (not .go file nodes — file paths don't form valid
// `go test ./pkg/file.go/...` targets). Rust: crate nodes only (file
// nodes would scope to non-existent cargo targets).
func isCandidateTestNode(nodeID string, ct *catalog.CheckType) bool {
	switch ct.Language {
	case "solidity":
		return isTestFileNode(nodeID)
	case "go":
		if !strings.HasPrefix(nodeID, "go:") {
			return false
		}
		// Reject file nodes (IDs ending in `.go`).
		return !strings.HasSuffix(nodeID, ".go")
	case "rust":
		if !strings.HasPrefix(nodeID, "rs:") {
			return false
		}
		return !strings.Contains(nodeID[3:], "/")
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
// sol:/go:/rs: node ID. Used when matching against changedLines.
func nodeIDToSourceFile(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		return strings.TrimPrefix(nodeID, "sol:")
	}
	if strings.HasPrefix(nodeID, "go:") {
		return strings.TrimPrefix(nodeID, "go:")
	}
	if strings.HasPrefix(nodeID, "rs:") {
		return strings.TrimPrefix(nodeID, "rs:")
	}
	return nodeID
}

// nodeIDToScope converts a source node ID to a command-line scope
// argument based on the check's scope_type. Solidity src/L1/X.sol
// becomes ./test/L1/*; a Go package node becomes ./pkg/...; a Rust
// crate node becomes the bare crate name (passed as `-p <crate>`).
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
		if strings.HasPrefix(nodeID, "rs:") {
			path := strings.TrimPrefix(nodeID, "rs:")
			// Crate-level nodes only (no slash after rs:).
			if !strings.Contains(path, "/") {
				return path
			}
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
