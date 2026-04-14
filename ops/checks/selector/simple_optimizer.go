package selector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// SimpleOptimizer is the initial optimization algorithm.
// Given candidate nodes from Phase 1, it:
//   - Groups candidates by check type (scopeable vs binary)
//   - For scopeable checks: resolves affected scopes, maps (signal, stage) → config
//   - For binary checks: run/skip based on signal and stage cost
//   - Resolves prerequisites
//   - Computes parallel schedule
type SimpleOptimizer struct{}

// NewSimpleOptimizer creates a new SimpleOptimizer.
func NewSimpleOptimizer() *SimpleOptimizer { return &SimpleOptimizer{} }

// Optimize implements Optimizer.
func (o *SimpleOptimizer) Optimize(
	g *graph.Graph,
	candidates []NodeWithSignal,
	diffs []diff.FileDiff,
	stage Stage,
	cat *catalog.Catalog,
) (*Result, error) {
	result := &Result{Stage: stage.Name}

	if len(candidates) == 0 {
		return result, nil
	}

	// Build changed lines map from diffs: file path → set of changed line numbers
	changedLines := buildChangedLinesMap(diffs)

	// Build candidate lookup
	candidateSignal := make(map[string]float64)
	for _, c := range candidates {
		candidateSignal[c.NodeID] = c.Signal
	}

	// Get changed source node IDs from the diff — NOT from the graph.
	// Using the graph (inferChangedSourceNodes) returns every source node
	// connected to any candidate check, which is everything.
	changedIDs := diffToNodeIDs(g, diffs)

	// Process each candidate
	for _, c := range candidates {
		ctID := strings.TrimPrefix(c.NodeID, "check:")
		ct := cat.ByID(ctID)
		if ct == nil {
			continue
		}

		if ct.Scopeable {
			items := o.optimizeScopeable(g, changedIDs, ct, c.Signal, stage, changedLines)
			for _, item := range items {
				if item.SkipCost > item.RunCost {
					result.Items = append(result.Items, item)
				} else {
					result.Skipped = append(result.Skipped, item)
				}
			}
		} else {
			item := o.optimizeBinary(ct, c.Signal, stage)
			if item.SkipCost > item.RunCost {
				result.Items = append(result.Items, item)
			} else {
				result.Skipped = append(result.Skipped, item)
			}
		}
	}

	// Resolve prerequisites
	o.resolvePrerequisites(result, cat)

	// Sort by skip cost descending
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].SkipCost > result.Items[j].SkipCost
	})

	// Compute schedule
	result.Schedule = ComputeSchedule(result.Items, 0)
	result.WallClock = result.Schedule.WallClock
	result.TotalCPU = result.Schedule.TotalCPU

	return result, nil
}

// diffToNodeIDs maps changed file paths from diffs to graph node IDs.
func diffToNodeIDs(g *graph.Graph, diffs []diff.FileDiff) []string {
	var filePaths []string
	for _, d := range diffs {
		if d.Path != "" {
			filePaths = append(filePaths, d.Path)
		}
	}
	nodeIDs, _ := diff.FilesToNodeIDs(g, filePaths)
	return nodeIDs
}

// optimizeScopeable creates ExecutionItems for a scopeable check type.
func (o *SimpleOptimizer) optimizeScopeable(
	g *graph.Graph,
	changedIDs []string,
	ct *catalog.CheckType,
	signal float64,
	stage Stage,
	changedLines map[string]map[int]bool,
) []ExecutionItem {
	checkNodeID := "check:" + ct.ID
	scopes := affectedScopes(g, changedIDs, checkNodeID, ct.ScopeType, changedLines)

	// If signal=1.0 (blast radius or direct trigger) and no scopes found,
	// run everything (nil scope = unscoped)
	if len(scopes) == 0 && signal >= 1.0 {
		config := mapSignalToConfig(ct, 1.0, stage)
		cost := float64(ct.AvgDuration)
		return []ExecutionItem{{
			ID:            ct.ID + ":all",
			CheckTypeID:   ct.ID,
			Scope:         nil,
			Config:        config,
			Signal:        1.0,
			RunCost:       cost,
			SkipCost:      cost + 1, // always run
			Prerequisites: prereqItemIDs(ct),
		}}
	}

	if len(scopes) == 0 {
		return nil
	}

	// Group scopes by signal tier for different configs
	type tier struct {
		minSignal float64
		label     string
	}
	tiers := []tier{
		{0.6, "high"},
		{0.3, "med"},
		{0.1, "low"},
	}

	var items []ExecutionItem
	used := make(map[string]bool)

	for _, t := range tiers {
		var tierScopes []string
		var maxSignal float64
		for _, sw := range scopes {
			if sw.Signal >= t.minSignal && !used[sw.Scope] {
				tierScopes = append(tierScopes, sw.Scope)
				used[sw.Scope] = true
				if sw.Signal > maxSignal {
					maxSignal = sw.Signal
				}
			}
		}
		if len(tierScopes) == 0 {
			continue
		}

		config := mapSignalToConfig(ct, maxSignal, stage)
		cost := estimateScopedRunCost(ct, len(tierScopes), config)

		prior := checkTypePrior(ct.Kind)
		if maxSignal > 0.6 {
			prior = 0.7
		}
		pFail := maxSignal * prior

		skipCost := pFail * stage.MissCost
		// Direct changes always run their tests
		if maxSignal > 0.6 {
			skipCost = cost + 1
		}

		items = append(items, ExecutionItem{
			ID:            fmt.Sprintf("%s:%s", ct.ID, t.label),
			CheckTypeID:   ct.ID,
			Scope:         tierScopes,
			Config:        config,
			Signal:        maxSignal,
			RunCost:       cost,
			SkipCost:      skipCost,
			Prerequisites: prereqItemIDs(ct),
		})
	}

	return items
}

// optimizeBinary creates an ExecutionItem for a binary check.
func (o *SimpleOptimizer) optimizeBinary(
	ct *catalog.CheckType,
	signal float64,
	stage Stage,
) ExecutionItem {
	prior := checkTypePrior(ct.Kind)
	pFail := prior * signal

	return ExecutionItem{
		ID:            ct.ID,
		CheckTypeID:   ct.ID,
		Signal:        signal,
		RunCost:       float64(ct.AvgDuration),
		SkipCost:      pFail * stage.MissCost,
		Prerequisites: prereqItemIDs(ct),
	}
}

// resolvePrerequisites ensures prerequisite items are included.
func (o *SimpleOptimizer) resolvePrerequisites(result *Result, cat *catalog.Catalog) {
	selected := make(map[string]bool)
	for _, item := range result.Items {
		selected[item.ID] = true
	}

	changed := true
	for changed {
		changed = false
		for _, item := range result.Items {
			for _, prereqID := range item.Prerequisites {
				if selected[prereqID] {
					continue
				}
				ct := cat.ByID(prereqID)
				if ct == nil {
					continue
				}
				result.Items = append(result.Items, ExecutionItem{
					ID:          prereqID,
					CheckTypeID: prereqID,
					Signal:      0,
					RunCost:     float64(ct.AvgDuration),
					SkipCost:    0,
				})
				selected[prereqID] = true
				changed = true
			}
		}
	}
}

// --- Helpers (Phase 2 / optimization concerns) ---

type scopeWithSignal struct {
	Scope  string
	Signal float64
}

// affectedScopes finds which test scopes are affected by changed source nodes.
// Tries two strategies in order:
//   1. Coverage-based (precise): look for coverage edges pointing at changed source
//      nodes. These come from `checks coverage ingest` and tell us exactly which
//      test files exercise the changed code.
//   2. Import-based (broad): walk from changed nodes through import edges and
//      collect source nodes with tested_by edges to the check type. This is the
//      fallback when no coverage data exists.
func affectedScopes(
	g *graph.Graph,
	changedIDs []string,
	checkTypeNodeID string,
	scopeType string,
	changedLines map[string]map[int]bool,
) []scopeWithSignal {
	// Strategy 1: Coverage-based scoping with line-level intersection
	if results := coverageBasedScopes(g, changedIDs, scopeType, changedLines); len(results) > 0 {
		return results
	}

	// Strategy 2: Import-based scoping (fallback)
	return importBasedScopes(g, changedIDs, checkTypeNodeID, scopeType)
}

// coverageBasedScopes finds test files that cover the *specific lines* that changed.
// It intersects coverage line ranges (from coverage edges) with changed lines (from diff).
// A test is only included if it covers at least one changed line.
func coverageBasedScopes(
	g *graph.Graph,
	changedIDs []string,
	scopeType string,
	changedLines map[string]map[int]bool,
) []scopeWithSignal {
	if len(changedLines) == 0 {
		return nil
	}

	// For each changed source node, find test nodes with coverage edges
	// and check if their covered line ranges intersect with the changed lines.
	testSignals := make(map[string]float64)

	for _, changedID := range changedIDs {
		// Get the source file path from the node ID to look up changed lines
		sourceFile := nodeIDToSourceFile(changedID)
		fileChangedLines := changedLines[sourceFile]
		if len(fileChangedLines) == 0 {
			// No line-level info for this file — fall back to file-level match
			// (any test covering this file counts)
			for _, edge := range g.EdgesTo(changedID) {
				if edge.Source != graph.SourceCoverage {
					continue
				}
				signal := edge.Strength * edge.Confidence
				if signal > testSignals[edge.From] {
					testSignals[edge.From] = signal
				}
			}
			continue
		}

		// Line-level intersection: only include tests that cover changed lines
		for _, edge := range g.EdgesTo(changedID) {
			if edge.Source != graph.SourceCoverage {
				continue
			}

			// Get the line ranges this test covers for this source file
			lineRanges, ok := edge.Properties["line_ranges"]
			if !ok {
				// No line data on edge — treat as file-level match
				signal := edge.Strength * edge.Confidence
				if signal > testSignals[edge.From] {
					testSignals[edge.From] = signal
				}
				continue
			}

			// Check if any covered ranges intersect with changed lines
			hitCount := countLineHits(lineRanges, fileChangedLines)
			if hitCount == 0 {
				continue // this test doesn't cover any changed lines
			}

			// Signal: if a test covers ANY changed lines, it's relevant.
			// The fraction scales between 0.5 (covers some) and 1.0 (covers all).
			// Even 1 hit out of 100 changed lines means the test exercises changed code.
			totalChanged := len(fileChangedLines)
			hitFraction := float64(hitCount) / float64(totalChanged)
			signal := (0.5 + 0.5*hitFraction) * edge.Confidence
			if signal > testSignals[edge.From] {
				testSignals[edge.From] = signal
			}
		}
	}

	if len(testSignals) == 0 {
		return nil
	}

	var results []scopeWithSignal
	seen := make(map[string]bool)
	for testNodeID, signal := range testSignals {
		// Coverage gives us precise test file IDs — use the file path directly
		// instead of converting to directory globs via nodeIDToScope.
		scope := nodeIDToTestPath(testNodeID)
		if scope == "" {
			// Fallback to directory-level scope
			scope = nodeIDToScope(testNodeID, scopeType)
		}
		if scope != "" && !seen[scope] {
			results = append(results, scopeWithSignal{Scope: scope, Signal: signal})
			seen[scope] = true
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Signal > results[j].Signal
	})

	return results
}

// nodeIDToTestPath converts a test node ID to a specific test file path for scoping.
// Returns the file path for forge --match-path (e.g. "test/L1/OptimismPortal2.t.sol").
func nodeIDToTestPath(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		path := strings.TrimPrefix(nodeID, "sol:")
		if strings.HasPrefix(path, "test/") && strings.HasSuffix(path, ".t.sol") {
			return "./" + path
		}
	}
	return ""
}

// countLineHits counts how many changed lines fall within the coverage ranges.
func countLineHits(lineRanges any, changedLines map[int]bool) int {
	hits := 0

	// lineRanges can be []interface{} (from JSON) or [][2]int (from Go)
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

// nodeIDToSourceFile extracts the source file path from a graph node ID.
func nodeIDToSourceFile(nodeID string) string {
	if strings.HasPrefix(nodeID, "sol:") {
		return strings.TrimPrefix(nodeID, "sol:")
	}
	if strings.HasPrefix(nodeID, "go:") {
		return strings.TrimPrefix(nodeID, "go:")
	}
	return nodeID
}

// buildChangedLinesMap extracts changed line numbers from parsed diffs.
// Returns: file path → set of changed line numbers (new side).
func buildChangedLinesMap(diffs []diff.FileDiff) map[string]map[int]bool {
	result := make(map[string]map[int]bool)

	for _, d := range diffs {
		if d.Path == "" || len(d.Hunks) == 0 {
			continue
		}

		// Strip common prefix for Solidity files
		path := d.Path
		if strings.HasPrefix(path, "packages/contracts-bedrock/") {
			path = strings.TrimPrefix(path, "packages/contracts-bedrock/")
		}

		lines := make(map[int]bool)
		for _, h := range d.Hunks {
			// The new-side lines (added/modified)
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

// importBasedScopes is the fallback when no coverage data exists.
// Walks from changed nodes through import edges and collects source nodes
// that have tested_by edges to the check type.
func importBasedScopes(
	g *graph.Graph,
	changedIDs []string,
	checkTypeNodeID string,
	scopeType string,
) []scopeWithSignal {
	allReachable := graph.ReachableChecks(g, changedIDs, 0.01)

	checkReachable := false
	for _, r := range allReachable {
		if r.CheckID == checkTypeNodeID {
			checkReachable = true
			break
		}
	}
	if !checkReachable {
		return nil
	}

	type nodeSignal struct {
		id     string
		signal float64
	}

	bestSignal := make(map[string]float64)
	queue := make([]nodeSignal, 0, len(changedIDs))
	for _, id := range changedIDs {
		bestSignal[id] = 1.0
		queue = append(queue, nodeSignal{id, 1.0})
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.signal < bestSignal[curr.id] {
			continue
		}

		for _, edge := range g.EdgesFrom(curr.id) {
			if edge.Kind == graph.EdgeTestedBy || edge.Kind == graph.EdgePrerequisite {
				continue
			}
			newSignal := curr.signal * edge.Strength * edge.Confidence
			if newSignal < 0.01 {
				continue
			}
			if existing, ok := bestSignal[edge.To]; !ok || newSignal > existing {
				bestSignal[edge.To] = newSignal
				queue = append(queue, nodeSignal{edge.To, newSignal})
			}
		}
	}

	var results []scopeWithSignal
	seen := make(map[string]bool)
	for nodeID, signal := range bestSignal {
		node := g.GetNode(nodeID)
		if node == nil || node.Kind != graph.KindSource {
			continue
		}
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind == graph.EdgeTestedBy && edge.To == checkTypeNodeID {
				scope := nodeIDToScope(nodeID, scopeType)
				if scope != "" && !seen[scope] {
					results = append(results, scopeWithSignal{Scope: scope, Signal: signal})
					seen[scope] = true
				}
				break
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Signal > results[j].Signal
	})

	return results
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

func mapSignalToConfig(ct *catalog.CheckType, signal float64, stage Stage) map[string]any {
	config := make(map[string]any)
	for _, knob := range ct.Knobs {
		switch knob.Name {
		case "fuzz_runs":
			config[knob.Name] = fuzzRunsForSignalStage(signal, stage)
		case "short":
			config[knob.Name] = signal < 0.8
		case "race":
			config[knob.Name] = signal > 0.8 && stage.MissCost >= StageOnPR.MissCost
		case "timeout":
			if signal > 0.8 {
				config[knob.Name] = "30m"
			} else {
				config[knob.Name] = "10m"
			}
		case "count":
			config[knob.Name] = 1
		default:
			config[knob.Name] = knob.Default
		}
	}
	return config
}

func fuzzRunsForSignalStage(signal float64, stage Stage) int {
	var high, med, low int
	switch {
	case stage.MissCost >= StageDevelop.MissCost:
		high, med, low = 20000, 1024, 128
	case stage.MissCost >= StageMergeQueue.MissCost:
		high, med, low = 1024, 128, 64
	case stage.MissCost >= StageOnPR.MissCost:
		high, med, low = 128, 64, 8
	case stage.MissCost >= StageOnCommit.MissCost:
		high, med, low = 64, 8, 1
	default:
		high, med, low = 1, 1, 1
	}

	switch {
	case signal > 0.6:
		return high
	case signal > 0.3:
		return med
	default:
		return low
	}
}

func checkTypePrior(kind string) float64 {
	switch kind {
	case "lint":
		return 0.8
	case "build":
		return 0.5
	case "test":
		return 0.3
	case "check":
		return 0.4
	default:
		return 0.3
	}
}

func estimateScopedRunCost(ct *catalog.CheckType, numScopes int, config map[string]any) float64 {
	if ct.PerUnitDuration <= 0 {
		return float64(ct.AvgDuration)
	}

	baseCost := float64(ct.PerUnitDuration)
	marginalCost := baseCost * 0.3
	if numScopes > 1 {
		baseCost += marginalCost * float64(numScopes-1)
	}

	if fuzzRuns, ok := config["fuzz_runs"]; ok {
		var fuzz float64
		switch v := fuzzRuns.(type) {
		case int:
			fuzz = float64(v)
		case float64:
			fuzz = v
		}
		if fuzz > 8 {
			fuzzMultiplier := 1.0 + 0.7*log2(fuzz/8)
			baseCost *= fuzzMultiplier
		}
	}

	return baseCost
}

func log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	result := 0.0
	for x >= 2 {
		x /= 2
		result++
	}
	if x > 1 {
		result += x - 1
	}
	return result
}

func prereqItemIDs(ct *catalog.CheckType) []string {
	return ct.Prerequisites
}
