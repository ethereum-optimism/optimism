package selector

import (
	"sort"
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
	return out
}

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
			// workspace member → rs:; otherwise external → mod:
			if g.GetNode("rs:" + dep) != nil {
				ids = append(ids, "rs:"+dep)
			} else {
				ids = append(ids, "mod:"+dep)
			}
		}
		// Also add the crate that owns this Cargo.toml, if any.
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
	// cargoTomlPath is repo-relative; crate node `dir` is absolute.
	// We compare by suffix to tolerate either form.
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

func extractPaths(diffs []diff.FileDiff) []string {
	var paths []string
	for _, d := range diffs {
		if d.Path != "" {
			paths = append(paths, d.Path)
		}
	}
	return paths
}

// blastRadiusCandidates emits one unscoped Candidate per check type,
// with signal=1.0 and provenance pointing at the triggering blast-radius
// files. Blast radius is the "something upstream of everything changed"
// escape hatch — CI config, toolchain, go.mod, foundry.toml, etc.
func blastRadiusCandidates(cat *catalog.Catalog, files []string) []Candidate {
	out := make([]Candidate, 0, len(cat.CheckTypes))
	for _, ct := range cat.CheckTypes {
		out = append(out, Candidate{
			CheckID: ct.ID,
			Signal:  1.0,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeKind(""),
				Contribution: 1.0,
				Raw: map[string]any{
					"reason": "blast_radius",
					"files":  files,
				},
			}},
		})
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

// mergeBinaryCandidates combines at most one import-based and one
// correlation-based candidate for the same check, taking max signal
// and unioning provenance so both sources show up in `explain`.
func mergeBinaryCandidates(imports, correlation []Candidate) []Candidate {
	switch {
	case len(imports) == 0 && len(correlation) == 0:
		return nil
	case len(imports) == 0:
		return correlation
	case len(correlation) == 0:
		return imports
	}
	primary, secondary := imports[0], correlation[0]
	if secondary.Signal > primary.Signal {
		primary, secondary = secondary, primary
	}
	primary.Provenance = append(primary.Provenance, secondary.Provenance...)
	return []Candidate{primary}
}

// coverageCandidates emits one Candidate per (test, profile) whose
// coverage line ranges intersect the diff's changed lines. Signal comes
// from the fraction of changed lines the test touches, floored at
// policy.Coverage.SignalFloor for any hit (rewards hit/no-hit
// discrimination over fine-grained fractions — see 76967a8b5f).
func coverageCandidates(
	g *graph.Graph,
	ct *catalog.CheckType,
	changedIDs []string,
	changedLines map[string]map[int]bool,
	pol *policy.Policy,
	fresh freshness.Checker,
) []Candidate {
	floor := pol.Coverage.SignalFloor
	if len(changedLines) == 0 {
		return nil
	}

	type key struct {
		testNode string
		profile  string
	}
	type entry struct {
		signal     float64
		hitLines   int
		totalLines int
		freshness  float64 // multiplier actually applied (1.0 = as-generated)
	}
	best := make(map[key]entry)

	// update retains the highest-signal entry per (test, profile).
	update := func(k key, raw entry) {
		raw.signal *= raw.freshness
		if e, ok := best[k]; !ok || raw.signal > e.signal {
			best[k] = raw
		}
	}

	for _, changedID := range changedIDs {
		sourceFile := nodeIDToSourceFile(changedID)
		fileChanged := changedLines[sourceFile]

		for _, edge := range g.EdgesTo(changedID) {
			if edge.Source != graph.SourceCoverage {
				continue
			}
			profile := profileFromEdge(edge)
			fr := fresh.Assess(edge)

			if len(fileChanged) == 0 {
				// No line-level info from the diff — treat as a file-level match.
				update(key{edge.From, profile}, entry{
					signal:    edge.Strength * edge.Confidence,
					freshness: fr,
				})
				continue
			}

			lineRanges, ok := edge.Properties["line_ranges"]
			if !ok {
				update(key{edge.From, profile}, entry{
					signal:    edge.Strength * edge.Confidence,
					freshness: fr,
				})
				continue
			}

			hitCount := countLineHits(lineRanges, fileChanged)
			if hitCount == 0 {
				continue
			}

			totalChanged := len(fileChanged)
			hitFraction := float64(hitCount) / float64(totalChanged)
			// Signal rises from `floor` (any hit) to `floor + (1-floor)*hitFraction`
			// when every changed line is covered.
			signal := (floor + (1-floor)*hitFraction) * edge.Confidence
			update(key{edge.From, profile}, entry{
				signal:     signal,
				hitLines:   hitCount,
				totalLines: totalChanged,
				freshness:  fr,
			})
		}
	}

	if len(best) == 0 {
		return nil
	}

	var out []Candidate
	for k, e := range best {
		scope := nodeIDToTestPath(k.testNode)
		if scope == "" {
			scope = nodeIDToScope(k.testNode, ct.ScopeType)
		}
		if scope == "" {
			continue
		}
		raw := map[string]any{
			"test_node": k.testNode,
		}
		if e.hitLines > 0 {
			raw["hit_lines"] = e.hitLines
			raw["total_changed"] = e.totalLines
		}
		if e.freshness != 0 && e.freshness < 1.0 {
			raw["freshness"] = e.freshness
		}
		out = append(out, Candidate{
			CheckID: ct.ID,
			Scope:   scope,
			Profile: k.profile,
			Signal:  e.signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceCoverage,
				EdgeKind:     graph.EdgeTestedBy,
				Contribution: e.signal,
				Raw:          raw,
			}},
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Signal > out[j].Signal })
	return out
}

// importScopeCandidates is the fallback for scopeable checks when no
// coverage edges point at the changed files. Reverse-BFS walks *incoming*
// import edges from the changed nodes to find everything that
// transitively imports them — the set of files whose compilation or
// behavior could break when the changed file changes.
//
// This catches cases coverage misses:
//   - Test helper changes (test/helpers/Foo.sol has no tested_by edge
//     pointing at it, so coverage finds nothing; but test files that
//     import it must re-run because their compilation can break).
//   - Source changes in modules where coverage hasn't been collected.
//
// Edge direction: the Solidity adapter writes `importer → imported`,
// so to find consumers of a changed node we walk `EdgesTo(cur)` and
// propagate from edge.From (the importer).
//
// Only consumers that are themselves test files, and that have a
// tested_by edge to this check, become scope candidates; the scope is
// the specific test file path (e.g. `./test/L1/X.t.sol`), not a
// directory glob.
func importScopeCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string) []Candidate {
	checkNodeID := "check:" + ct.ID

	consumers := transitiveConsumers(g, changedIDs, 0.01)
	if len(consumers) == 0 {
		return nil
	}

	type emit struct {
		nodeID string
		signal float64
	}
	var hits []emit
	for nodeID, signal := range consumers {
		if !isCandidateTestNode(nodeID, ct) {
			continue
		}
		if !hasTestedByEdge(g, nodeID, checkNodeID) {
			continue
		}
		hits = append(hits, emit{nodeID, signal})
	}
	if len(hits) == 0 {
		return nil
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].signal > hits[j].signal })

	seen := make(map[string]bool)
	out := make([]Candidate, 0, len(hits))
	for _, h := range hits {
		scope := scopeForCandidate(h.nodeID, ct)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, Candidate{
			CheckID: ct.ID,
			Scope:   scope,
			Signal:  h.signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeImports,
				Contribution: h.signal,
				Raw: map[string]any{
					"via_node": h.nodeID,
					"reason":   "transitive_importer",
				},
			}},
		})
	}
	return out
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
// Prefers the Solidity-specific test-path mapping; falls back to the
// scope_type-driven derivation (Go packages, etc.).
func scopeForCandidate(nodeID string, ct *catalog.CheckType) string {
	if s := nodeIDToTestPath(nodeID); s != "" {
		return s
	}
	return nodeIDToScope(nodeID, ct.ScopeType)
}

// transitiveConsumers performs reverse-BFS on incoming `imports` edges
// from each changed node. Returns node ID → accumulated signal for
// every node that (transitively) imports a changed node. Signal
// attenuates by edge.Strength * edge.Confidence per hop.
func transitiveConsumers(g *graph.Graph, changedIDs []string, minSignal float64) map[string]float64 {
	best := make(map[string]float64)
	type item struct {
		id     string
		signal float64
	}
	queue := make([]item, 0, len(changedIDs))
	for _, id := range changedIDs {
		if g.GetNode(id) == nil {
			continue
		}
		best[id] = 1.0
		queue = append(queue, item{id, 1.0})
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.signal < best[cur.id] {
			continue
		}
		for _, edge := range g.EdgesTo(cur.id) {
			if edge.Kind != graph.EdgeImports {
				continue
			}
			s := cur.signal * edge.Strength * edge.Confidence
			if s < minSignal {
				continue
			}
			importer := edge.From
			if existing, ok := best[importer]; !ok || s > existing {
				best[importer] = s
				queue = append(queue, item{importer, s})
			}
		}
	}
	return best
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

// importBinaryCandidates emits a single Candidate for a non-scopeable
// check if it's reachable via the import graph.
func importBinaryCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string) []Candidate {
	checkNodeID := "check:" + ct.ID
	for _, r := range graph.ReachableChecks(g, changedIDs, 0.01) {
		if r.CheckID != checkNodeID {
			continue
		}
		return []Candidate{{
			CheckID: ct.ID,
			Signal:  r.Signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeTestedBy,
				Contribution: r.Signal,
				Raw: map[string]any{
					"path": r.Path,
				},
			}},
		}}
	}
	return nil
}

// correlationCandidates emits a Candidate whose signal comes from
// EdgeObservedCorrelation edges written by CI-history ingestion. For
// each changed node, the strongest outgoing correlation edge to this
// check's node becomes the candidate's signal; freshness is applied
// like it is for coverage.
func correlationCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string, fresh freshness.Checker) []Candidate {
	checkNodeID := "check:" + ct.ID
	var bestSignal float64
	var bestProps map[string]any
	var bestFreshness float64

	for _, nodeID := range changedIDs {
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind != graph.EdgeObservedCorrelation || edge.To != checkNodeID {
				continue
			}
			fr := fresh.Assess(edge)
			signal := edge.Strength * edge.Confidence * fr
			if signal > bestSignal {
				bestSignal = signal
				bestProps = edge.Properties
				bestFreshness = fr
			}
		}
	}

	if bestSignal == 0 {
		return nil
	}

	raw := map[string]any{}
	if bestProps != nil {
		if v, ok := bestProps["observations"]; ok {
			raw["observations"] = v
		}
		if v, ok := bestProps["failures"]; ok {
			raw["failures"] = v
		}
		if v, ok := bestProps["precision"]; ok {
			raw["precision"] = v
		}
	}
	if bestFreshness < 1.0 {
		raw["freshness"] = bestFreshness
	}

	return []Candidate{{
		CheckID: ct.ID,
		Signal:  bestSignal,
		Provenance: []SignalContribution{{
			Source:       graph.SourceCIHistory,
			EdgeKind:     graph.EdgeObservedCorrelation,
			Contribution: bestSignal,
			Raw:          raw,
		}},
	}}
}

// --- helpers moved from simple_optimizer.go ---

func profileFromEdge(edge *graph.Edge) string {
	if edge.Properties == nil {
		return ""
	}
	if p, ok := edge.Properties["profile"].(string); ok {
		return p
	}
	return ""
}

func nodeIDToTestPath(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		path := strings.TrimPrefix(nodeID, "sol:")
		if strings.HasPrefix(path, "test/") && strings.HasSuffix(path, ".t.sol") {
			return "./" + path
		}
	}
	return ""
}

func countLineHits(lineRanges any, changedLines map[int]bool) int {
	hits := 0
	switch ranges := lineRanges.(type) {
	case [][2]int:
		for _, r := range ranges {
			for line := r[0]; line <= r[1]; line++ {
				if changedLines[line] {
					hits++
				}
			}
		}
	case []interface{}:
		for _, r := range ranges {
			rSlice, ok := r.([]interface{})
			if !ok || len(rSlice) != 2 {
				continue
			}
			start, ok1 := toInt(rSlice[0])
			end, ok2 := toInt(rSlice[1])
			if !ok1 || !ok2 {
				continue
			}
			for line := start; line <= end; line++ {
				if changedLines[line] {
					hits++
				}
			}
		}
	}
	return hits
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

func nodeIDToSourceFile(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		return strings.TrimPrefix(nodeID, "sol:")
	}
	if strings.HasPrefix(nodeID, "go:") {
		return strings.TrimPrefix(nodeID, "go:")
	}
	return nodeID
}

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
