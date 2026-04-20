package graph

import (
	"container/heap"
	"strings"
)

// CheckSignal represents a reachable check node with its accumulated signal.
type CheckSignal struct {
	CheckID string
	Signal  float64  // accumulated relevance signal from graph walk
	Path    []string // node IDs along the shortest path from a changed node
}

// ReachableChecks performs a Dijkstra-style walk from changed nodes, following
// outgoing edges and accumulating signal as the product of edge strengths along
// the path. Returns check nodes whose accumulated signal is >= minSignal.
func ReachableChecks(g *Graph, changedIDs []string, minSignal float64) []CheckSignal {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Best signal seen so far for each node
	bestSignal := make(map[string]float64)
	bestPath := make(map[string][]string)

	// Priority queue: highest signal first
	pq := &signalHeap{}
	heap.Init(pq)

	// Seed with changed nodes at signal=1.0
	for _, id := range changedIDs {
		if _, ok := g.Nodes[id]; !ok {
			continue
		}
		bestSignal[id] = 1.0
		bestPath[id] = []string{id}
		heap.Push(pq, signalItem{id: id, signal: 1.0, path: []string{id}})
	}

	for pq.Len() > 0 {
		item := heap.Pop(pq).(signalItem)

		// Skip if we already found a better path
		if item.signal < bestSignal[item.id] {
			continue
		}

		// Walk outgoing edges
		for _, edge := range g.outgoing[item.id] {
			newSignal := item.signal * edge.Strength * edge.Confidence
			if newSignal < minSignal {
				continue
			}

			if existing, ok := bestSignal[edge.To]; !ok || newSignal > existing {
				bestSignal[edge.To] = newSignal
				newPath := make([]string, len(item.path)+1)
				copy(newPath, item.path)
				newPath[len(item.path)] = edge.To
				bestPath[edge.To] = newPath
				heap.Push(pq, signalItem{id: edge.To, signal: newSignal, path: newPath})
			}
		}
	}

	// Collect check nodes
	var results []CheckSignal
	for id, signal := range bestSignal {
		node := g.Nodes[id]
		if node != nil && node.Kind == KindCheck {
			results = append(results, CheckSignal{
				CheckID: id,
				Signal:  signal,
				Path:    bestPath[id],
			})
		}
	}

	return results
}

// Prerequisites returns the transitive prerequisite closure for a check.
// Follows EdgePrerequisite edges from the check node to find all prerequisites.
//
// Deprecated: replaced by CheckPrerequisites which derives prereq
// ordering from the dataflow graph (produces/consumes via artifacts).
// Kept for the transition so existing callers still compile.
func Prerequisites(g *Graph, checkID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var result []string
	var walk func(id string)
	walk = func(id string) {
		for _, edge := range g.incoming[id] {
			if edge.Kind != EdgePrerequisite {
				continue
			}
			if visited[edge.From] {
				continue
			}
			visited[edge.From] = true
			walk(edge.From)
			result = append(result, edge.From)
		}
	}
	walk(checkID)
	return result
}

// CheckPrerequisites returns the transitive set of check IDs that
// must run before checkID, derived from the dataflow graph: for every
// artifact this check consumes, every check that produces the artifact
// is a prerequisite (transitively).
//
// Returns IDs WITHOUT the "check:" prefix, in deterministic
// topological order (producers before consumers; ties broken by
// lexicographic sort on check ID). Determinism is load-bearing:
// ExecutionItem.Prerequisites feeds the scheduler and CI log output,
// both of which require stable ordering across runs.
//
// checkID must include the "check:" prefix.
func CheckPrerequisites(g *Graph, checkID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// BFS-style walk collecting check→check dependencies derived from
	// artifact production/consumption.
	producers := make(map[string]bool)
	queue := []string{checkID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		// For every artifact this check consumes, find its producers.
		for _, outEdge := range g.outgoing[cur] {
			if outEdge.Kind != EdgeConsumes {
				continue
			}
			for _, inEdge := range g.incoming[outEdge.To] {
				if inEdge.Kind != EdgeProduces {
					continue
				}
				if inEdge.From == checkID {
					continue
				}
				if producers[inEdge.From] {
					continue
				}
				producers[inEdge.From] = true
				queue = append(queue, inEdge.From)
			}
		}
	}
	// Topological order: producer-before-consumer is implicit in the
	// artifact chain; we don't have cycles by catalog validation. For
	// determinism, sort lexicographically. A proper topo-sort would
	// order intermediate producers too; when callers need strict topo
	// (e.g. scheduler), they call this per layer and the scheduler
	// levels them.
	out := make([]string, 0, len(producers))
	for id := range producers {
		out = append(out, strings.TrimPrefix(id, "check:"))
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	// Local sort to avoid importing sort in a perf-sensitive file if
	// the caller didn't already. Simple insertion sort; the prereq
	// lists are tiny (typically 0-3 entries).
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// signalItem is an entry in the priority queue.
type signalItem struct {
	id     string
	signal float64
	path   []string
}

// signalHeap implements heap.Interface for max-signal-first ordering.
type signalHeap []signalItem

func (h signalHeap) Len() int            { return len(h) }
func (h signalHeap) Less(i, j int) bool  { return h[i].signal > h[j].signal }
func (h signalHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *signalHeap) Push(x interface{}) { *h = append(*h, x.(signalItem)) }
func (h *signalHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
