package interop

import (
	"cmp"
	"errors"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// ErrCycle is returned when a cycle is detected in same-timestamp messages.
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
// it assumes all executing messages are included on blocks of the given timestamp
// For each EM, two types of edges are added:
// 1. Intra-chain: depends on the previous EM on the same chain (if exists)
// 2. Cross-chain: depends on executingMessageBefore(targetChain, targetLogIdx) (if exists)
func buildCycleGraph(ts uint64, chainEMs map[eth.ChainID]map[uint32]*types.ExecutingMessage) *dependencyGraph {
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
// using Kahn's topological sort algorithm.
//
// Returns a Result with InvalidHeads populated for chains participating in cycles.
func (i *Interop) verifyCycleMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	result := Result{
		Timestamp: ts,
		L2Heads:   blocksAtTimestamp,
	}

	// collect all EMs for the given blocks per chain
	chainEMs := make(map[eth.ChainID]map[uint32]*types.ExecutingMessage)
	for chainID, blockID := range blocksAtTimestamp {
		db, ok := i.logsDBs[chainID]
		if !ok {
			// Chain not in logsDBs - skip it for cycle verification
			continue
		}
		blockRef, _, execMsgs, err := db.OpenBlock(blockID.Number)
		if err != nil {
			// Can't open block - no EMs to add to the graph for this chain
			// This can happen if the logsDB is empty or the block hasn't been indexed
			continue
		}
		// Verify the block has the expected timestamp
		if blockRef.Time != ts {
			// Block timestamp mismatch - skip this chain for cycle verification
			continue
		}
		chainEMs[chainID] = execMsgs
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
