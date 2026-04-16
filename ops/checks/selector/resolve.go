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

	if isBlast, files := diff.BlastRadiusFiles(filePaths, pol.BlastRadius); isBlast {
		return blastRadiusCandidates(cat, files)
	}

	changedLines := buildChangedLinesMap(diffs)
	changedIDs, _ := diff.FilesToNodeIDs(g, filePaths)

	var out []Candidate
	for i := range cat.CheckTypes {
		ct := &cat.CheckTypes[i]
		out = append(out, candidatesForCheck(g, ct, filePaths, changedIDs, changedLines, pol, fresh)...)
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

	// Binary check: reachable via the import graph?
	return importBinaryCandidates(g, ct, changedIDs)
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
// coverage edges point at the changed files. It walks from changed
// nodes through non-test, non-prereq edges (i.e. through imports) and
// collects source nodes that have a tested_by edge to this check type.
func importScopeCandidates(g *graph.Graph, ct *catalog.CheckType, changedIDs []string) []Candidate {
	checkNodeID := "check:" + ct.ID

	// Cheap check: if Dijkstra doesn't reach this check at all, skip the walk.
	reachable := false
	for _, r := range graph.ReachableChecks(g, changedIDs, 0.01) {
		if r.CheckID == checkNodeID {
			reachable = true
			break
		}
	}
	if !reachable {
		return nil
	}

	type nodeSignal struct {
		id     string
		signal float64
	}
	best := make(map[string]float64)
	queue := make([]nodeSignal, 0, len(changedIDs))
	for _, id := range changedIDs {
		best[id] = 1.0
		queue = append(queue, nodeSignal{id, 1.0})
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.signal < best[cur.id] {
			continue
		}
		for _, edge := range g.EdgesFrom(cur.id) {
			if edge.Kind == graph.EdgeTestedBy || edge.Kind == graph.EdgePrerequisite {
				continue
			}
			s := cur.signal * edge.Strength * edge.Confidence
			if s < 0.01 {
				continue
			}
			if existing, ok := best[edge.To]; !ok || s > existing {
				best[edge.To] = s
				queue = append(queue, nodeSignal{edge.To, s})
			}
		}
	}

	seen := make(map[string]bool)
	var out []Candidate
	for nodeID, signal := range best {
		node := g.GetNode(nodeID)
		if node == nil || node.Kind != graph.KindSource {
			continue
		}
		tested := false
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind == graph.EdgeTestedBy && edge.To == checkNodeID {
				tested = true
				break
			}
		}
		if !tested {
			continue
		}
		scope := nodeIDToScope(nodeID, ct.ScopeType)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, Candidate{
			CheckID: ct.ID,
			Scope:   scope,
			Signal:  signal,
			Provenance: []SignalContribution{{
				Source:       graph.SourceStatic,
				EdgeKind:     graph.EdgeImports,
				Contribution: signal,
				Raw: map[string]any{
					"via_node": nodeID,
				},
			}},
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Signal > out[j].Signal })
	return out
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
