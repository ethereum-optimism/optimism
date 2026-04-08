package selector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// SimpleStrategy is the initial algorithm implementation.
type SimpleStrategy struct{}

// NewSimpleStrategy creates a new SimpleStrategy.
func NewSimpleStrategy() *SimpleStrategy { return &SimpleStrategy{} }

// Plan computes an execution plan.
func (s *SimpleStrategy) Plan(
	g *graph.Graph,
	diffs []diff.FileDiff,
	stage Stage,
	cat *catalog.Catalog,
) (*Result, error) {
	result := &Result{Stage: stage.Name}

	// Extract changed file paths
	var filePaths []string
	for _, d := range diffs {
		if d.Path != "" {
			filePaths = append(filePaths, d.Path)
		}
	}

	if len(filePaths) == 0 {
		return result, nil
	}

	// Check blast radius
	isBlast, _ := diff.BlastRadiusFiles(filePaths)

	// Map files to graph node IDs
	changedIDs, _ := diff.FilesToNodeIDs(g, filePaths)

	// Process each check type
	for _, ct := range cat.CheckTypes {
		if ct.Scopeable {
			items := s.planScopeableCheck(g, changedIDs, &ct, stage, isBlast)
			for _, item := range items {
				if item.SkipCost > item.RunCost {
					result.Items = append(result.Items, item)
				} else {
					result.Skipped = append(result.Skipped, item)
				}
			}
		} else {
			item := s.planBinaryCheck(g, changedIDs, filePaths, &ct, stage, isBlast)
			if item == nil {
				continue
			}
			if item.SkipCost > item.RunCost {
				result.Items = append(result.Items, *item)
			} else {
				result.Skipped = append(result.Skipped, *item)
			}
		}
	}

	// Resolve prerequisites: if an item is selected, its prereqs must be too
	s.resolvePrerequisites(result, cat, stage)

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

// planScopeableCheck creates ExecutionItems for a scopeable check (forge-test, go-test).
// Groups affected scopes by signal tier and creates one item per tier.
func (s *SimpleStrategy) planScopeableCheck(
	g *graph.Graph,
	changedIDs []string,
	ct *catalog.CheckType,
	stage Stage,
	isBlast bool,
) []ExecutionItem {
	checkNodeID := "check:" + ct.ID
	scopes := affectedScopes(g, changedIDs, checkNodeID, ct.ScopeType)

	if isBlast {
		// Blast radius: everything at max config
		config := mapSignalToConfig(ct, 1.0, stage)
		cost := float64(ct.AvgDuration)
		return []ExecutionItem{{
			ID:          ct.ID + ":all",
			CheckTypeID: ct.ID,
			Scope:       nil, // nil scope = run everything
			Config:      config,
			Signal:      1.0,
			RunCost:     cost,
			SkipCost:    1.0 * stage.MissCost,
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
		cost := float64(ct.PerUnitDuration) * float64(len(tierScopes))
		if cost == 0 {
			cost = float64(ct.AvgDuration)
		}

		// P(fail) heuristic: signal × check type prior
		prior := checkTypePrior(ct.Kind)
		pFail := maxSignal * prior

		items = append(items, ExecutionItem{
			ID:          fmt.Sprintf("%s:%s", ct.ID, t.label),
			CheckTypeID: ct.ID,
			Scope:       tierScopes,
			Config:      config,
			Signal:      maxSignal,
			RunCost:     cost,
			SkipCost:    pFail * stage.MissCost,
			Prerequisites: prereqItemIDs(ct),
		})
	}

	return items
}

// planBinaryCheck creates an ExecutionItem for a non-scopeable check.
func (s *SimpleStrategy) planBinaryCheck(
	g *graph.Graph,
	changedIDs []string,
	filePaths []string,
	ct *catalog.CheckType,
	stage Stage,
	isBlast bool,
) *ExecutionItem {
	triggered := isBlast

	if !triggered {
		// Check if any changed files match triggers
		if len(ct.Triggers) > 0 {
			triggered = ct.MatchesTriggers(filePaths)
		} else {
			// No triggers defined — check if any changed node reaches this check
			reachable := graph.ReachableChecks(g, changedIDs, 0.01)
			for _, r := range reachable {
				if r.CheckID == "check:"+ct.ID {
					triggered = true
					break
				}
			}
		}
	}

	if !triggered {
		return nil
	}

	prior := checkTypePrior(ct.Kind)
	pFail := prior // binary checks: if triggered, use the prior directly

	return &ExecutionItem{
		ID:            ct.ID,
		CheckTypeID:   ct.ID,
		Signal:        1.0,
		RunCost:       float64(ct.AvgDuration),
		SkipCost:      pFail * stage.MissCost,
		Prerequisites: prereqItemIDs(ct),
	}
}

// resolvePrerequisites ensures prerequisite items are included when dependents are selected.
func (s *SimpleStrategy) resolvePrerequisites(result *Result, cat *catalog.Catalog, stage Stage) {
	selected := make(map[string]bool)
	for _, item := range result.Items {
		selected[item.ID] = true
	}

	// Check if any selected item has unresolved prerequisites
	changed := true
	for changed {
		changed = false
		for _, item := range result.Items {
			for _, prereqID := range item.Prerequisites {
				if selected[prereqID] {
					continue
				}
				// Find the prerequisite check type
				ct := cat.ByID(prereqID)
				if ct == nil {
					continue
				}
				prereqItem := ExecutionItem{
					ID:          prereqID,
					CheckTypeID: prereqID,
					Signal:      0,
					RunCost:     float64(ct.AvgDuration),
					SkipCost:    0,
				}
				result.Items = append(result.Items, prereqItem)
				selected[prereqID] = true
				changed = true
			}
		}
	}
}

// affectedScopes walks the graph to find which source nodes are affected
// and maps them to scope units.
func affectedScopes(
	g *graph.Graph,
	changedIDs []string,
	checkTypeNodeID string,
	scopeType string,
) []scopeWithSignal {
	// Walk from changed nodes. For each source node we reach,
	// check if it has a tested_by edge to the check type node.
	// Reuse the ReachableChecks mechanism but collect source nodes.

	// First, find all source nodes with their signal
	allReachable := graph.ReachableChecks(g, changedIDs, 0.01)

	// If the check type node is reachable, collect the source nodes
	// that have tested_by edges to it.
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

	// Now walk from changed nodes and collect source nodes that connect to this check
	type nodeSignal struct {
		id     string
		signal float64
	}

	// BFS from changed nodes
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
				continue // don't walk into check nodes
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

	// Collect source nodes that have tested_by edges to the check type node
	var results []scopeWithSignal
	for nodeID, signal := range bestSignal {
		node := g.GetNode(nodeID)
		if node == nil || node.Kind != graph.KindSource {
			continue
		}
		// Check if this source has a tested_by edge to the target check
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind == graph.EdgeTestedBy && edge.To == checkTypeNodeID {
				scope := nodeIDToScope(nodeID, scopeType)
				if scope != "" {
					results = append(results, scopeWithSignal{
						Scope:  scope,
						Signal: signal,
					})
				}
				break
			}
		}
	}

	// Sort by signal descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Signal > results[j].Signal
	})

	return results
}

type scopeWithSignal struct {
	Scope  string
	Signal float64
}

// nodeIDToScope converts a graph node ID to a scope unit.
func nodeIDToScope(nodeID, scopeType string) string {
	switch scopeType {
	case "packages":
		if strings.HasPrefix(nodeID, "go:") {
			// Convert full import path to relative package path
			// e.g. "go:github.com/ethereum-optimism/optimism/op-node/rollup" → "./op-node/rollup/..."
			path := strings.TrimPrefix(nodeID, "go:")
			// Find the repo-relative part (after the module path)
			parts := strings.SplitN(path, "/", 4) // github.com/org/repo/rest
			if len(parts) >= 4 {
				return "./" + parts[3] + "/..."
			}
			return "./" + path + "/..."
		}
	case "paths":
		if strings.HasPrefix(nodeID, "sol:") {
			path := strings.TrimPrefix(nodeID, "sol:")
			// Convert source paths to test paths for forge
			// src/L1/Foo.sol → test/L1/*
			if strings.HasPrefix(path, "src/") {
				dir := strings.TrimPrefix(path, "src/")
				dir = strings.Split(dir, "/")[0] // get top-level dir (L1, L2, etc.)
				return "./test/" + dir + "/*"
			}
			// Test files are already in test/ paths
			if strings.HasPrefix(path, "test/") {
				dir := strings.TrimPrefix(path, "test/")
				dir = strings.Split(dir, "/")[0]
				return "./test/" + dir + "/*"
			}
		}
	}
	return ""
}

// mapSignalToConfig maps graph signal × stage to knob values.
func mapSignalToConfig(ct *catalog.CheckType, signal float64, stage Stage) map[string]any {
	config := make(map[string]any)

	for _, knob := range ct.Knobs {
		switch knob.Name {
		case "fuzz_runs":
			config[knob.Name] = fuzzRunsForSignalStage(signal, stage)
		case "short":
			// Direct dependency (high signal) → don't skip slow tests
			config[knob.Name] = signal < 0.8
		case "race":
			// Only enable race detector for direct dependencies at higher stages
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
	// Stage determines the base fuzz depth
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
	default: // save
		high, med, low = 1, 1, 1
	}

	// Signal selects which tier
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

func prereqItemIDs(ct *catalog.CheckType) []string {
	// For now, prerequisite item IDs = prerequisite check type IDs
	// (binary prereqs like forge-build have the same item ID as their check type ID)
	return ct.Prerequisites
}
