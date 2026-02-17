package interop

/*
SPECIFICATION: Same-Timestamp Cycle Verification

This file implements Kahn's topological sort algorithm to detect circular dependencies
in same-timestamp executing messages. When multiple chains have blocks at the same
timestamp, executing messages between them can form cycles that must be validated.

## Problem Statement

At timestamp T, we may have:
- Chain A block with logs [L0, L1, L2] where L2 is an exec msg referencing Chain B
- Chain B block with logs [L0, L1] where L1 is an exec msg referencing Chain A

This creates a dependency graph that must be acyclic for the messages to be valid.

## Algorithm: Kahn's Topological Sort

### 1. Graph Construction (verifyCycleMessages)

- Pull logs from chains that emitted messages at this exact timestamp
- Create map of ChainID -> []ExecutingMessage (only same-timestamp exec msgs)
- Construct dependencyNode slice where each node represents a log entry

**Edges** (dependencies):
- Intra-chain: Each log depends on the previous log in the same block
  This ensures logs within a block are processed in order.
- Cross-chain: Each same-timestamp executing message depends on its initiating message.
  exec(chainA, logIdx=2) referencing init(chainB, logIdx=0) creates: A:2 dependsOn B:0

### 2. Cycle Detection (Kahn's Algorithm)

Two-part cycle repeated until termination:

Part 1: For each node with empty dependedOnBy (nothing depends on it):
  - Add to removeSet
  - Set resolved = true

Part 2: For each node:
  - Remove items in removeSet from its dependsOn slice

Termination conditions:
  - SUCCESS: All nodes are resolved → No cycle, messages are valid
  - FAILURE: Unresolved nodes exist AND removeSet is empty → Cycle detected

### 3. Result

- If acyclic: Return valid Result (no InvalidHeads)
- If cyclic: Return Result with all participating chains in InvalidHeads

## Example: Valid (Acyclic)

Chain A at T=1000: [L0, L1(exec B:L0)]
Chain B at T=1000: [L0(init)]

Graph:
  A:L0 (no dependedOnBy after A:L1 is removed first)
  A:L1 dependsOn [A:L0, B:L0], dependedOnBy []
  B:L0 dependsOn [], dependedOnBy [A:L1]

Process:
  1. A:L1 has no dependedOnBy → add to removeSet, resolved=true
  2. Remove A:L1 from dependedOnBy of A:L0 and B:L0
  3. Now A:L0 and B:L0 have no dependedOnBy → add to removeSet, resolved=true
  4. All resolved → VALID

## Example: Invalid (Cyclic)

Chain A at T=1000: [L0(exec B:L0)]
Chain B at T=1000: [L0(exec A:L0)]

Graph:
  A:L0 dependsOn [B:L0], dependedOnBy [B:L0]
  B:L0 dependsOn [A:L0], dependedOnBy [A:L0]

Process:
  1. No nodes have empty dependedOnBy
  2. removeSet is empty, but nodes remain unresolved
  → CYCLE DETECTED → INVALID
*/

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// ErrCycle is returned when a circular dependency is detected in same-timestamp messages.
var ErrCycle = errors.New("cycle detected in same-timestamp messages")

// dependencyNode represents a log entry in the dependency graph.
// It tracks what this node depends on, and what depends on this node.
type dependencyNode struct {
	chainID  eth.ChainID
	logIndex uint32
	execMsg  *types.ExecutingMessage // nil if not an executing message

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

// checkCycle runs Kahn's topological sort algorithm to detect cycles.
// Returns nil if the graph is acyclic (valid), ErrCycle if a cycle is detected.
//
// Algorithm:
// 1. Find nodes with no dependedOnBy (nothing depends on them) → add to removeSet, mark resolved
// 2. Remove items in removeSet from dependedOnBy of all nodes
// 3. Repeat until either:
//   - All nodes resolved → acyclic (valid)
//   - No progress (removeSet empty but unresolved nodes remain) → cycle detected
func checkCycle(g *dependencyGraph) error {
	if len(*g) == 0 {
		return nil
	}

	for {
		// Part 1: Find nodes with no dependedOnBy and mark them resolved
		var removeSet []*dependencyNode
		for _, node := range *g {
			if !node.resolved && len(node.dependedOnBy) == 0 {
				node.resolved = true
				removeSet = append(removeSet, node)
			}
		}

		// If no nodes can be removed, check termination
		if len(removeSet) == 0 {
			// Check if all nodes are resolved
			for _, node := range *g {
				if !node.resolved {
					// Unresolved nodes remain but no progress → cycle detected
					return ErrCycle
				}
			}
			// All nodes resolved → acyclic
			return nil
		}

		// Part 2: Remove items in removeSet from dependedOnBy of all nodes
		for _, removed := range removeSet {
			for _, dependent := range removed.dependedOnBy {
				// This shouldn't happen since removed nodes have empty dependedOnBy,
				// but we clear it anyway for completeness
				dependent.dependsOn = removeFromSlice(dependent.dependsOn, removed)
			}
			// Remove this node from dependedOnBy of nodes it depends on
			for _, dependency := range removed.dependsOn {
				dependency.dependedOnBy = removeFromSlice(dependency.dependedOnBy, removed)
			}
		}
	}
}

// removeFromSlice removes a node from a slice of nodes.
func removeFromSlice(slice []*dependencyNode, toRemove *dependencyNode) []*dependencyNode {
	result := make([]*dependencyNode, 0, len(slice))
	for _, n := range slice {
		if n != toRemove {
			result = append(result, n)
		}
	}
	return result
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
// Only same-timestamp EMs are included in the graph.
//
// For each EM, two types of edges are added:
// 1. Intra-chain: depends on the previous EM on the same chain (if exists)
// 2. Cross-chain: depends on executingMessageBefore(targetChain, targetLogIdx) (if exists)
func buildCycleGraph(ts uint64, chainEMs map[eth.ChainID]map[uint32]*types.ExecutingMessage) *dependencyGraph {
	graph := &dependencyGraph{}

	// First pass: create nodes for all same-timestamp EMs, organized by chain
	chainNodes := make(map[eth.ChainID][]*dependencyNode)
	nodeByLocation := make(map[eth.ChainID]map[uint32]*dependencyNode)

	for chainID, emsMap := range chainEMs {
		nodeByLocation[chainID] = make(map[uint32]*dependencyNode)

		// Collect and sort log indices
		var logIndices []uint32
		for logIdx, em := range emsMap {
			if em != nil && em.Timestamp == ts {
				logIndices = append(logIndices, logIdx)
			}
		}
		// Sort log indices
		sortUint32s(logIndices)

		// Create nodes in order
		for _, logIdx := range logIndices {
			em := emsMap[logIdx]
			node := &dependencyNode{
				chainID:  chainID,
				logIndex: logIdx,
				execMsg:  em,
			}
			graph.addNode(node)
			chainNodes[chainID] = append(chainNodes[chainID], node)
			nodeByLocation[chainID][logIdx] = node
		}
	}

	// Second pass: add edges
	for chainID, nodes := range chainNodes {
		for i, node := range nodes {
			// Intra-chain edge: depends on previous EM on same chain
			if i > 0 {
				prevNode := nodes[i-1]
				graph.addEdge(node, prevNode)
			}

			// Cross-chain edge: depends on executingMessageBefore on target chain
			if node.execMsg != nil {
				targetChain := node.execMsg.ChainID
				targetLogIdx := node.execMsg.LogIdx

				// Skip if referencing same chain (would be handled by intra-chain)
				if targetChain == chainID {
					continue
				}

				targetChainNodes := chainNodes[targetChain]
				if depNode := executingMessageBefore(targetChainNodes, targetLogIdx); depNode != nil {
					graph.addEdge(node, depNode)
				}
			}
		}
	}

	return graph
}

// sortUint32s sorts a slice of uint32 in ascending order.
func sortUint32s(s []uint32) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// verifyCycleMessages is the cycle verification function for same-timestamp interop.
// It verifies that same-timestamp executing messages form valid dependency relationships
// using Kahn's topological sort algorithm.
//
// Returns a Result with InvalidHeads populated for chains participating in cycles.
func (i *Interop) verifyCycleMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	result := Result{
		Timestamp: ts,
		L2Heads:   blocksAtTimestamp,
	}

	// Collect all same-timestamp executing messages from each chain
	chainEMs := make(map[eth.ChainID]map[uint32]*types.ExecutingMessage)
	for chainID, blockID := range blocksAtTimestamp {
		db, ok := i.logsDBs[chainID]
		if !ok {
			continue
		}

		_, _, execMsgs, err := db.OpenBlock(blockID.Number)
		if err != nil {
			// Skip blocks we can't open - they may be handled elsewhere
			continue
		}

		// Filter to only same-timestamp EMs
		sameTS := make(map[uint32]*types.ExecutingMessage)
		for logIdx, em := range execMsgs {
			if em != nil && em.Timestamp == ts {
				sameTS[logIdx] = em
			}
		}
		if len(sameTS) > 0 {
			chainEMs[chainID] = sameTS
		}
	}

	// Build dependency graph and check for cycles
	graph := buildCycleGraph(ts, chainEMs)
	if err := checkCycle(graph); err != nil {
		// Cycle detected - mark only chains with unresolved nodes as invalid
		// (bystander chains that have same-ts EMs but aren't part of the cycle are spared)
		cycleChains := collectCycleParticipants(graph)
		if len(cycleChains) > 0 {
			result.InvalidHeads = make(map[eth.ChainID]eth.BlockID)
			for chainID := range cycleChains {
				result.InvalidHeads[chainID] = blocksAtTimestamp[chainID]
			}
		}
	}

	return result, nil
}

// collectCycleParticipants returns the set of chains that have unresolved nodes
// after running checkCycle. These are the chains actually participating in a cycle.
func collectCycleParticipants(graph *dependencyGraph) map[eth.ChainID]bool {
	cycleChains := make(map[eth.ChainID]bool)
	for _, node := range *graph {
		if !node.resolved {
			cycleChains[node.chainID] = true
		}
	}
	return cycleChains
}
