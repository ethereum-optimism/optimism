package interop

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrCycle is returned when a cycle is detected in same-timestamp messages.
var ErrCycle = errors.New("cycle detected in same-timestamp messages")

// dependencyNode represents a log entry in the dependency graph.
// It tracks what this node depends on, and what depends on this node.
type dependencyNode struct {
	chainID  eth.ChainID
	logIndex uint32
	execMsg  *messages.ExecutingMessage // nil if not an executing message

	resolved     bool
	dependsOn    []*dependencyNode
	dependedOnBy []*dependencyNode
}

// dependencyGraph is a collection of dependency nodes for cycle checking.
type dependencyGraph []*dependencyNode

// addNode adds a node to the graph.
func (g *dependencyGraph) addNode(n *dependencyNode) {
	*g = append(*g, n)
}

// addEdge adds a directed dependency: "from" depends on "to".
// This means "to" must be resolved before "from" can be resolved.
func (g *dependencyGraph) addEdge(from, to *dependencyNode) {
	from.dependsOn = append(from.dependsOn, to)
	to.dependedOnBy = append(to.dependedOnBy, from)
}

// checkCycle finds strongly connected components and marks only cycle nodes unresolved.
// It returns ErrCycle when the graph contains a directed cycle.
func checkCycle(g *dependencyGraph) error {
	graphNodes := make(map[*dependencyNode]struct{}, len(*g))
	for _, node := range *g {
		node.resolved = false
		graphNodes[node] = struct{}{}
	}

	type dfsFrame struct {
		node     *dependencyNode
		nextEdge int
	}

	// The first pass records finish order in the dependency graph.
	visited := make(map[*dependencyNode]bool, len(*g))
	finishOrder := make([]*dependencyNode, 0, len(*g))
	for _, start := range *g {
		if visited[start] {
			continue
		}
		visited[start] = true
		stack := []dfsFrame{{node: start}}
		for len(stack) > 0 {
			frame := &stack[len(stack)-1]
			if frame.nextEdge < len(frame.node.dependsOn) {
				next := frame.node.dependsOn[frame.nextEdge]
				frame.nextEdge++
				if _, ok := graphNodes[next]; !ok || visited[next] {
					continue
				}
				visited[next] = true
				stack = append(stack, dfsFrame{node: next})
				continue
			}
			finishOrder = append(finishOrder, frame.node)
			stack = stack[:len(stack)-1]
		}
	}

	// The second pass collects components in the reversed graph.
	visited = make(map[*dependencyNode]bool, len(*g))
	hasCycle := false
	for i := len(finishOrder) - 1; i >= 0; i-- {
		start := finishOrder[i]
		if visited[start] {
			continue
		}

		visited[start] = true
		stack := []*dependencyNode{start}
		component := make([]*dependencyNode, 0, 1)
		for len(stack) > 0 {
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			component = append(component, node)
			for _, next := range node.dependedOnBy {
				if _, ok := graphNodes[next]; !ok || visited[next] {
					continue
				}
				visited[next] = true
				stack = append(stack, next)
			}
		}

		// A multi-node component always contains a directed cycle.
		componentHasCycle := len(component) > 1
		if len(component) == 1 {
			for _, dependency := range component[0].dependsOn {
				if dependency == component[0] {
					componentHasCycle = true
					break
				}
			}
		}
		for _, node := range component {
			node.resolved = !componentHasCycle
		}
		hasCycle = hasCycle || componentHasCycle
	}

	if hasCycle {
		return ErrCycle
	}
	return nil
}

// executingMessageBefore finds the latest EM in the slice with logIndex <= targetLogIdx.
// The slice must be sorted by logIndex ascending.
// Returns nil if no such EM exists.
func executingMessageBefore(chainEMs []*dependencyNode, targetLogIdx uint32) *dependencyNode {
	var result *dependencyNode
	for _, em := range chainEMs {
		if em.logIndex <= targetLogIdx {
			result = em // keep updating to get the latest one at or before target
		} else {
			break // since sorted, no need to continue
		}
	}
	return result
}

// buildCycleGraph constructs a dependency graph from executing messages at the given timestamp.
// it assumes all executing messages are included on blocks of the given timestamp
// For each EM, two types of edges are added:
// 1. Intra-chain: depends on the previous EM on the same chain (if exists)
// 2. Cross-chain: depends on executingMessageBefore(targetChain, targetLogIdx) (if exists)
func buildCycleGraph(ts uint64, chainEMs map[eth.ChainID]map[uint32]*messages.ExecutingMessage) *dependencyGraph {
	graph := &dependencyGraph{}
	orderedExecutingMessages := make(map[eth.ChainID][]*dependencyNode)

	// First pass: create nodes for all same-timestamp EMs
	for chainID, emsMap := range chainEMs {
		for logIdx, em := range emsMap {
			if em != nil && em.Timestamp == ts {
				node := &dependencyNode{
					chainID:  chainID,
					logIndex: logIdx,
					execMsg:  em,
				}
				graph.addNode(node)
				orderedExecutingMessages[chainID] = append(orderedExecutingMessages[chainID], node)
			}
		}
	}

	// Sort each chain's nodes by logIndex (map iteration order is non-deterministic)
	for _, nodes := range orderedExecutingMessages {
		slices.SortFunc(nodes, func(a, b *dependencyNode) int {
			return cmp.Compare(a.logIndex, b.logIndex)
		})
	}

	// Second pass: add edges
	for _, nodes := range orderedExecutingMessages {
		for i, node := range nodes {
			// all nodes point back to the previous node on the same chain
			if i > 0 {
				graph.addEdge(node, nodes[i-1])
			}

			// all nodes also point to their target
			targetChainEMs := orderedExecutingMessages[node.execMsg.ChainID]
			target := executingMessageBefore(targetChainEMs, node.execMsg.LogIdx)
			if target != nil {
				graph.addEdge(node, target)
			}
		}
	}

	return graph
}

// verifyCycleMessages is the cycle verification function for same-timestamp interop.
// It verifies that same-timestamp executing messages form valid dependency relationships
// with strongly connected component detection.

// Returns a Result with InvalidHeads populated for chains participating in cycles.
func (i *Interop) verifyCycleMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID, view *frontierVerificationView) (Result, error) {
	result := Result{
		Timestamp: ts,
		L2Heads:   blocksAtTimestamp,
	}

	// collect all EMs for the given blocks per chain
	chainEMs := make(map[eth.ChainID]map[uint32]*messages.ExecutingMessage)
	for chainID, blockID := range blocksAtTimestamp {
		if frontierBlock, ok := view.block(chainID); ok {
			if frontierBlock.ref.Time == ts {
				chainEMs[chainID] = frontierBlock.execMsgs
			}
			continue
		}

		db, ok := i.logsDBs[chainID]
		if !ok {
			// Chain not registered with the interop activity - out of scope for
			// cycle verification, consistent with verifyInteropMessages.
			continue
		}
		blockRef, _, execMsgs, err := db.OpenBlock(blockID.Number)
		if err != nil {
			// The block is expected to be available: chain-readiness gating and
			// the frontier view guarantee it. A read failure means the data the
			// cycle check needs is missing, not that the chain has no
			// same-timestamp messages. Surface it so the round retries rather
			// than silently dropping the chain - and with it every cross-chain
			// edge that targets it, which could hide a real cycle.
			return Result{}, fmt.Errorf("chain %s: failed to open block %d for cycle verification: %w", chainID, blockID.Number, err)
		}
		// A block at a different timestamp legitimately contributes no
		// same-timestamp executing messages; skip it without error.
		if blockRef.Time != ts {
			continue
		}
		chainEMs[chainID] = execMsgs
	}

	// Build the dependency graph and check for cycles.
	graph := buildCycleGraph(ts, chainEMs)
	if err := checkCycle(graph); err != nil {
		// Mark only chains with cycle nodes as invalid.
		cycleChains := collectCycleParticipants(graph)
		if len(cycleChains) > 0 {
			result.InvalidHeads = make(map[eth.ChainID]InvalidHead)
			for chainID := range cycleChains {
				invalid, err := i.newInvalidHead(chainID, blocksAtTimestamp[chainID])
				if err != nil {
					return Result{}, fmt.Errorf("chain %s: %w", chainID, err)
				}
				result.InvalidHeads[chainID] = invalid
			}
		}
	}

	return result, nil
}

// collectCycleParticipants returns chains with directed-cycle nodes.
// checkCycle marks only those nodes unresolved.
func collectCycleParticipants(graph *dependencyGraph) map[eth.ChainID]bool {
	cycleChains := make(map[eth.ChainID]bool)
	for _, node := range *graph {
		if !node.resolved {
			cycleChains[node.chainID] = true
		}
	}
	return cycleChains
}
