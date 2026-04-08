package graph

import "container/heap"

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
