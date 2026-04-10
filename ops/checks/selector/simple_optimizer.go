package selector

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
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
	stage Stage,
	cat *catalog.Catalog,
) (*Result, error) {
	result := &Result{Stage: stage.Name}

	if len(candidates) == 0 {
		return result, nil
	}

	// Build candidate lookup
	candidateSignal := make(map[string]float64)
	for _, c := range candidates {
		candidateSignal[c.NodeID] = c.Signal
	}

	// Get changed source node IDs for scope resolution.
	// We reconstruct these from the graph — source nodes that have edges
	// leading to the candidate check nodes.
	changedIDs := inferChangedSourceNodes(g, candidates)

	// Process each candidate
	for _, c := range candidates {
		// Strip "check:" prefix to get check type ID
		ctID := strings.TrimPrefix(c.NodeID, "check:")
		ct := cat.ByID(ctID)
		if ct == nil {
			continue
		}

		if ct.Scopeable {
			items := o.optimizeScopeable(g, changedIDs, ct, c.Signal, stage)
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

// inferChangedSourceNodes finds source nodes that are "close" to the candidate
// check nodes. We walk backward from check nodes to find source nodes with
// tested_by edges, then trace which ones have high signal.
func inferChangedSourceNodes(g *graph.Graph, candidates []NodeWithSignal) []string {
	// Collect all source nodes that have tested_by edges to candidate check nodes
	candidateSet := make(map[string]bool)
	for _, c := range candidates {
		candidateSet[c.NodeID] = true
	}

	var sourceIDs []string
	seen := make(map[string]bool)
	for _, node := range g.NodesOfKind(graph.KindSource) {
		for _, edge := range g.EdgesFrom(node.ID) {
			if edge.Kind == graph.EdgeTestedBy && candidateSet[edge.To] {
				if !seen[node.ID] {
					sourceIDs = append(sourceIDs, node.ID)
					seen[node.ID] = true
				}
				break
			}
		}
	}
	return sourceIDs
}

// optimizeScopeable creates ExecutionItems for a scopeable check type.
func (o *SimpleOptimizer) optimizeScopeable(
	g *graph.Graph,
	changedIDs []string,
	ct *catalog.CheckType,
	signal float64,
	stage Stage,
) []ExecutionItem {
	checkNodeID := "check:" + ct.ID
	scopes := affectedScopes(g, changedIDs, checkNodeID, ct.ScopeType)

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

// affectedScopes walks the graph to find which source nodes are affected
// and maps them to scope units for a specific check type.
func affectedScopes(
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
	for nodeID, signal := range bestSignal {
		node := g.GetNode(nodeID)
		if node == nil || node.Kind != graph.KindSource {
			continue
		}
		for _, edge := range g.EdgesFrom(nodeID) {
			if edge.Kind == graph.EdgeTestedBy && edge.To == checkTypeNodeID {
				scope := nodeIDToScope(nodeID, scopeType)
				if scope != "" {
					results = append(results, scopeWithSignal{Scope: scope, Signal: signal})
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
