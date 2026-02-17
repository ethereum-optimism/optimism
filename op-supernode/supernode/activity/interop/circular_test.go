package interop

/*
TEST PLAN: Same-Timestamp Cycle Verification

KEY INSIGHT: executingMessageBefore semantics
  - Dependencies are only between Executing Messages (EMs), not all logs.
  - When EM references log index X, we depend on executingMessageBefore(chain, X):
    the latest EM with logIndex <= X on that chain.
  - If no such EM exists, NO cross-chain edge is added.
  - Therefore: If Chain B has no EMs at or before the referenced log index,
    no dependency is created even if A references B.

Tests are organized by component:

1. Graph Construction Tests
   - Test node creation with correct chainID and logIndex
   - Test edge creation updates both dependsOn and dependedOnBy
   - executingMessageBefore: empty chain, no match, single match, latest match

2. Kahn's Algorithm Tests (checkCycle function)
   - Empty graph → returns nil (no cycle)
   - Single node, no deps → resolves successfully
   - Linear chain (A→B→C) → resolves successfully (acyclic)
   - Simple cycle (A↔B) → returns error (cycle detected)
   - Complex cycle (A→B→C→A) → returns error (cycle detected)
   - Diamond pattern (A→B, A→C, B→D, C→D) → resolves successfully (acyclic)
   - Mixed: some nodes cyclic, some not → returns error

3. buildCycleGraph Tests
   - Mutual EMs referencing each other's exact log index → CYCLE
   - Mutual EMs where one references before the other's EM → depends on setup
   - Triangle patterns

4. Integration Tests (verifyCycleMessages)
   - No same-timestamp exec msgs → valid result
   - Same-timestamp exec msgs, no cycle → valid result
   - Same-timestamp exec msgs with cycle → invalid heads returned

5. Edge Cases
   - Self-referential message (A:L0 exec A:L0) → error
   - Unknown source chain → error
   - Log index out of bounds → error
*/

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Graph Construction Tests
// =============================================================================

func TestDependencyGraph_AddNode(t *testing.T) {
	t.Parallel()

	g := &dependencyGraph{}
	node := &dependencyNode{
		chainID:  eth.ChainIDFromUInt64(10),
		logIndex: 0,
	}

	g.addNode(node)

	require.Len(t, *g, 1)
	require.Equal(t, node, (*g)[0])
}

func TestDependencyGraph_AddEdge(t *testing.T) {
	t.Parallel()

	g := &dependencyGraph{}
	nodeA := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0}
	nodeB := &dependencyNode{chainID: eth.ChainIDFromUInt64(8453), logIndex: 0}

	g.addNode(nodeA)
	g.addNode(nodeB)

	// A depends on B (B must resolve before A)
	g.addEdge(nodeA, nodeB)

	require.Len(t, nodeA.dependsOn, 1)
	require.Equal(t, nodeB, nodeA.dependsOn[0])
	require.Len(t, nodeB.dependedOnBy, 1)
	require.Equal(t, nodeA, nodeB.dependedOnBy[0])
}

// =============================================================================
// executingMessageBefore Tests
// =============================================================================

func TestExecutingMessageBefore(t *testing.T) {
	t.Parallel()

	chainA := eth.ChainIDFromUInt64(10)

	tests := []struct {
		name           string
		chainEMs       []*dependencyNode // EMs on the chain, sorted by logIndex
		targetLogIdx   uint32
		expectNode     bool
		expectLogIndex uint32 // only checked if expectNode is true
	}{
		{
			name:         "empty chain returns nil",
			chainEMs:     nil,
			targetLogIdx: 5,
			expectNode:   false,
		},
		{
			name: "no EM at or before target returns nil",
			chainEMs: []*dependencyNode{
				{chainID: chainA, logIndex: 5},
				{chainID: chainA, logIndex: 10},
			},
			targetLogIdx: 3, // all EMs are > 3
			expectNode:   false,
		},
		{
			name: "exact match returns that EM",
			chainEMs: []*dependencyNode{
				{chainID: chainA, logIndex: 2},
				{chainID: chainA, logIndex: 5},
			},
			targetLogIdx:   5, // EM at exactly index 5
			expectLogIndex: 5,
			expectNode:     true,
		},
		{
			name: "returns latest EM at or before target",
			chainEMs: []*dependencyNode{
				{chainID: chainA, logIndex: 1},
				{chainID: chainA, logIndex: 3},
				{chainID: chainA, logIndex: 7},
			},
			targetLogIdx:   5, // EMs at 1 and 3 are <= 5, should return 3
			expectLogIndex: 3,
			expectNode:     true,
		},
		{
			name: "target at index 0 with EM at 0 returns that EM",
			chainEMs: []*dependencyNode{
				{chainID: chainA, logIndex: 0},
				{chainID: chainA, logIndex: 5},
			},
			targetLogIdx:   0, // EM at exactly 0
			expectLogIndex: 0,
			expectNode:     true,
		},
		{
			name: "target at index 0 with no EM at 0 returns nil",
			chainEMs: []*dependencyNode{
				{chainID: chainA, logIndex: 1},
				{chainID: chainA, logIndex: 5},
			},
			targetLogIdx: 0, // no EM at or before 0
			expectNode:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := executingMessageBefore(tc.chainEMs, tc.targetLogIdx)
			if tc.expectNode {
				require.NotNil(t, result, "expected to find an EM at or before target")
				require.Equal(t, tc.expectLogIndex, result.logIndex)
			} else {
				require.Nil(t, result, "expected no EM at or before target")
			}
		})
	}
}

// =============================================================================
// Kahn's Algorithm Tests
// =============================================================================

func TestCheckCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		buildGraph  func() *dependencyGraph
		expectCycle bool
	}{
		{
			name: "empty graph has no cycle",
			buildGraph: func() *dependencyGraph {
				return &dependencyGraph{}
			},
			expectCycle: false,
		},
		{
			name: "single node no deps resolves",
			buildGraph: func() *dependencyGraph {
				g := &dependencyGraph{}
				g.addNode(&dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0})
				return g
			},
			expectCycle: false,
		},
		{
			name: "linear chain A->B->C resolves (acyclic)",
			buildGraph: func() *dependencyGraph {
				g := &dependencyGraph{}
				a := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0}
				b := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 1}
				c := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 2}
				g.addNode(a)
				g.addNode(b)
				g.addNode(c)
				// c depends on b, b depends on a
				g.addEdge(c, b)
				g.addEdge(b, a)
				return g
			},
			expectCycle: false,
		},
		{
			name: "simple cycle A<->B detected",
			buildGraph: func() *dependencyGraph {
				g := &dependencyGraph{}
				a := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0}
				b := &dependencyNode{chainID: eth.ChainIDFromUInt64(8453), logIndex: 0}
				g.addNode(a)
				g.addNode(b)
				// A depends on B, B depends on A (cycle!)
				g.addEdge(a, b)
				g.addEdge(b, a)
				return g
			},
			expectCycle: true,
		},
		{
			name: "triangle cycle A->B->C->A detected",
			buildGraph: func() *dependencyGraph {
				g := &dependencyGraph{}
				a := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0}
				b := &dependencyNode{chainID: eth.ChainIDFromUInt64(8453), logIndex: 0}
				c := &dependencyNode{chainID: eth.ChainIDFromUInt64(420), logIndex: 0}
				g.addNode(a)
				g.addNode(b)
				g.addNode(c)
				// A depends on C, C depends on B, B depends on A (cycle!)
				g.addEdge(a, c)
				g.addEdge(c, b)
				g.addEdge(b, a)
				return g
			},
			expectCycle: true,
		},
		{
			name: "diamond pattern A->B,C B,C->D resolves (acyclic)",
			buildGraph: func() *dependencyGraph {
				g := &dependencyGraph{}
				a := &dependencyNode{chainID: eth.ChainIDFromUInt64(10), logIndex: 0}
				b := &dependencyNode{chainID: eth.ChainIDFromUInt64(8453), logIndex: 0}
				c := &dependencyNode{chainID: eth.ChainIDFromUInt64(420), logIndex: 0}
				d := &dependencyNode{chainID: eth.ChainIDFromUInt64(999), logIndex: 0}
				g.addNode(a)
				g.addNode(b)
				g.addNode(c)
				g.addNode(d)
				// D depends on B and C, B and C depend on A
				g.addEdge(d, b)
				g.addEdge(d, c)
				g.addEdge(b, a)
				g.addEdge(c, a)
				return g
			},
			expectCycle: false,
		},
		{
			name: "intra-chain sequential logs resolve",
			buildGraph: func() *dependencyGraph {
				// Simulates a single chain with 3 logs where each depends on previous
				g := &dependencyGraph{}
				chain10 := eth.ChainIDFromUInt64(10)
				l0 := &dependencyNode{chainID: chain10, logIndex: 0}
				l1 := &dependencyNode{chainID: chain10, logIndex: 1}
				l2 := &dependencyNode{chainID: chain10, logIndex: 2}
				g.addNode(l0)
				g.addNode(l1)
				g.addNode(l2)
				// l1 depends on l0, l2 depends on l1
				g.addEdge(l1, l0)
				g.addEdge(l2, l1)
				return g
			},
			expectCycle: false,
		},
		{
			name: "cross-chain valid exec message resolves",
			buildGraph: func() *dependencyGraph {
				// Chain A: [L0, L1(exec B:L0)]
				// Chain B: [L0(init)]
				g := &dependencyGraph{}
				chainA := eth.ChainIDFromUInt64(10)
				chainB := eth.ChainIDFromUInt64(8453)

				aL0 := &dependencyNode{chainID: chainA, logIndex: 0}
				aL1 := &dependencyNode{chainID: chainA, logIndex: 1, execMsg: &suptypes.ExecutingMessage{
					ChainID: chainB, LogIdx: 0,
				}}
				bL0 := &dependencyNode{chainID: chainB, logIndex: 0}

				g.addNode(aL0)
				g.addNode(aL1)
				g.addNode(bL0)

				// aL1 depends on aL0 (sequential) and bL0 (exec->init)
				g.addEdge(aL1, aL0)
				g.addEdge(aL1, bL0)
				return g
			},
			expectCycle: false,
		},
		{
			name: "cross-chain mutual exec creates cycle",
			buildGraph: func() *dependencyGraph {
				// Chain A: [L0(exec B:L0)]
				// Chain B: [L0(exec A:L0)]
				g := &dependencyGraph{}
				chainA := eth.ChainIDFromUInt64(10)
				chainB := eth.ChainIDFromUInt64(8453)

				aL0 := &dependencyNode{chainID: chainA, logIndex: 0, execMsg: &suptypes.ExecutingMessage{
					ChainID: chainB, LogIdx: 0,
				}}
				bL0 := &dependencyNode{chainID: chainB, logIndex: 0, execMsg: &suptypes.ExecutingMessage{
					ChainID: chainA, LogIdx: 0,
				}}

				g.addNode(aL0)
				g.addNode(bL0)

				// aL0 depends on bL0, bL0 depends on aL0 (cycle!)
				g.addEdge(aL0, bL0)
				g.addEdge(bL0, aL0)
				return g
			},
			expectCycle: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := tc.buildGraph()
			err := checkCycle(g)
			if tc.expectCycle {
				require.Error(t, err, "expected cycle to be detected")
			} else {
				require.NoError(t, err, "expected no cycle")
			}
		})
	}
}

// =============================================================================
// buildCycleGraph Tests
// =============================================================================

func TestBuildCycleGraph(t *testing.T) {
	t.Parallel()

	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(8453)
	chainC := eth.ChainIDFromUInt64(420)
	ts := uint64(1000)

	tests := []struct {
		name        string
		chainEMs    map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage
		expectCycle bool
	}{
		{
			name:        "no same-timestamp EMs returns valid (empty graph)",
			chainEMs:    map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{},
			expectCycle: false,
		},
		{
			name: "single chain single EM referencing past timestamp - filtered out",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts - 100}, // past timestamp, not same-ts
				},
			},
			expectCycle: false,
		},
		{
			name: "single chain single same-ts EM referencing chain with no EMs - no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts}, // B has no EMs
				},
			},
			expectCycle: false,
		},
		{
			name: "two chains mutual same-ts EMs at same log index - CYCLE",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts}, // refs B:0
				},
				chainB: {
					0: {ChainID: chainA, LogIdx: 0, Timestamp: ts}, // refs A:0
				},
			},
			// A:0 refs B:0 → executingMessageBefore(B, 0) = B:0 (exact match)
			// B:0 refs A:0 → executingMessageBefore(A, 0) = A:0 (exact match)
			// Both depend on each other → CYCLE
			expectCycle: true,
		},
		{
			name: "chain A refs B log 5, B has EM at 3 - one-way dependency, no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 5, Timestamp: ts}, // refs B:5
				},
				chainB: {
					3: {ChainID: chainC, LogIdx: 0, Timestamp: ts}, // refs C:0, not A
				},
			},
			// A:0 → executingMessageBefore(B, 5) = B:3 (3 <= 5)
			// B:3 → executingMessageBefore(C, 0) = nil (C has no EMs)
			// No cycle: A:0 → B:3 → nothing
			expectCycle: false,
		},
		{
			name: "chain A refs B log 2, B has EM only at 3 - no dependency, no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 2, Timestamp: ts}, // refs B:2
				},
				chainB: {
					3: {ChainID: chainA, LogIdx: 0, Timestamp: ts}, // refs A:0
				},
			},
			// A:0 → executingMessageBefore(B, 2) = nil (B:3 > 2)
			// B:3 → executingMessageBefore(A, 0) = A:0 (exact match)
			// No cycle: A:0 has no deps, B:3 → A:0
			expectCycle: false,
		},
		{
			name: "sequential EMs on same chain - intra-chain deps, no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts},
					5: {ChainID: chainB, LogIdx: 3, Timestamp: ts},
				},
			},
			// A:5 → A:0 (intra-chain)
			// A:0 → executingMessageBefore(B, 0) = nil (B has no EMs)
			// A:5 → executingMessageBefore(B, 3) = nil (B has no EMs)
			// No cycle
			expectCycle: false,
		},
		{
			name: "triangle at same log index - all reference each other - CYCLE",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts},
				},
				chainB: {
					0: {ChainID: chainC, LogIdx: 0, Timestamp: ts},
				},
				chainC: {
					0: {ChainID: chainA, LogIdx: 0, Timestamp: ts},
				},
			},
			// A:0 → B:0, B:0 → C:0, C:0 → A:0 → CYCLE
			expectCycle: true,
		},
		{
			name: "triangle where one leg has no matching EM - no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					5: {ChainID: chainB, LogIdx: 3, Timestamp: ts}, // refs B:3
				},
				chainB: {
					5: {ChainID: chainC, LogIdx: 3, Timestamp: ts}, // refs C:3
				},
				chainC: {
					5: {ChainID: chainA, LogIdx: 3, Timestamp: ts}, // refs A:3, but A has no EM <= 3
				},
			},
			// A:5 → executingMessageBefore(B, 3) = nil (B:5 > 3)
			// B:5 → executingMessageBefore(C, 3) = nil (C:5 > 3)
			// C:5 → executingMessageBefore(A, 3) = nil (A:5 > 3)
			// All have no cross-chain deps → no cycle
			expectCycle: false,
		},
		{
			name: "two chains with prior EMs creating mutual dependency - CYCLE",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 5, Timestamp: ts}, // refs B:5, B has EM at 3
					3: {ChainID: chainB, LogIdx: 5, Timestamp: ts}, // refs B:5
				},
				chainB: {
					3: {ChainID: chainA, LogIdx: 5, Timestamp: ts}, // refs A:5, A has EM at 3
					5: {ChainID: chainA, LogIdx: 5, Timestamp: ts}, // refs A:5
				},
			},
			// A:0 → executingMessageBefore(B, 5) = B:5 (5 <= 5)
			// A:3 → A:0 (intra) and executingMessageBefore(B, 5) = B:5
			// B:3 → executingMessageBefore(A, 5) = A:3 (3 <= 5)
			// B:5 → B:3 (intra) and executingMessageBefore(A, 5) = A:3
			// Cycle: A:3 → B:5 → A:3 (via B:5's dep on A:3)
			expectCycle: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			graph := buildCycleGraph(ts, tc.chainEMs)
			err := checkCycle(graph)
			if tc.expectCycle {
				require.Error(t, err, "expected cycle to be detected")
			} else {
				require.NoError(t, err, "expected no cycle")
			}
		})
	}
}

// =============================================================================
// Cycle Participant Detection Tests
// =============================================================================

// TestVerifyCycleMessagesOnlyCycleParticipants verifies that verifyCycleMessages
// only returns InvalidHeads for chains that are actually part of a cycle,
// not bystander chains that happen to have same-timestamp EMs.
//
// This test should FAIL until we update verifyCycleMessages to be more precise.
func TestVerifyCycleMessagesOnlyCycleParticipants(t *testing.T) {
	t.Parallel()

	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(8453)
	chainC := eth.ChainIDFromUInt64(420)
	ts := uint64(1000)

	// Setup: A↔C cycle, B is bystander
	// A:0 refs C:0, C:0 refs A:0 = cycle
	// B:0 refs some chain with no EMs = no cycle involvement

	blockA := eth.BlockID{Hash: common.HexToHash("0xAAA"), Number: 100}
	blockB := eth.BlockID{Hash: common.HexToHash("0xBBB"), Number: 100}
	blockC := eth.BlockID{Hash: common.HexToHash("0xCCC"), Number: 100}

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		chainA: blockA,
		chainB: blockB,
		chainC: blockC,
	}

	// Mock logsDBs that return same-timestamp EMs using existing algoMockLogsDB
	dbA := &algoMockLogsDB{
		openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
			0: {ChainID: chainC, LogIdx: 0, Timestamp: ts}, // refs C:0
		},
	}
	dbB := &algoMockLogsDB{
		openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
			0: {ChainID: eth.ChainIDFromUInt64(9999), LogIdx: 0, Timestamp: ts}, // refs non-existent chain
		},
	}
	dbC := &algoMockLogsDB{
		openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
			0: {ChainID: chainA, LogIdx: 0, Timestamp: ts}, // refs A:0 - creates cycle
		},
	}

	// Create Interop with mock DBs
	i := &Interop{
		logsDBs: map[eth.ChainID]LogsDB{
			chainA: dbA,
			chainB: dbB,
			chainC: dbC,
		},
	}

	// Call verifyCycleMessages
	result, err := i.verifyCycleMessages(ts, blocksAtTimestamp)
	require.NoError(t, err)

	// ASSERTION: Only A and C should be in InvalidHeads (they form the cycle)
	// B should NOT be in InvalidHeads (it's a bystander)
	require.NotNil(t, result.InvalidHeads, "InvalidHeads should not be nil when cycle detected")
	require.Contains(t, result.InvalidHeads, chainA, "Chain A should be invalid (part of cycle)")
	require.Contains(t, result.InvalidHeads, chainC, "Chain C should be invalid (part of cycle)")
	require.NotContains(t, result.InvalidHeads, chainB,
		"Chain B should NOT be invalid - it's a bystander with same-ts EMs but not part of the cycle")
}

// TestCycleParticipants verifies that only chains actually participating in a cycle
// are identified, not bystander chains that happen to have same-timestamp EMs.
func TestCycleParticipants(t *testing.T) {
	t.Parallel()

	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(8453)
	chainC := eth.ChainIDFromUInt64(420)
	chainD := eth.ChainIDFromUInt64(999)
	ts := uint64(1000)

	tests := []struct {
		name                 string
		chainEMs             map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage
		expectCycleChains    []eth.ChainID // chains that should be in the cycle
		expectNonCycleChains []eth.ChainID // chains with EMs that should NOT be in cycle
	}{
		{
			name: "A-C cycle with B as bystander",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainC, LogIdx: 0, Timestamp: ts}, // A refs C:0
				},
				chainB: {
					0: {ChainID: chainD, LogIdx: 0, Timestamp: ts}, // B refs D:0 (D has no EMs)
				},
				chainC: {
					0: {ChainID: chainA, LogIdx: 0, Timestamp: ts}, // C refs A:0 - creates cycle
				},
			},
			// A:0 → C:0, C:0 → A:0 = CYCLE between A and C
			// B:0 → nothing (D has no EMs)
			// B should NOT be part of the cycle!
			expectCycleChains:    []eth.ChainID{chainA, chainC},
			expectNonCycleChains: []eth.ChainID{chainB},
		},
		{
			name: "triangle cycle A-B-C with D as bystander",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				chainA: {
					0: {ChainID: chainB, LogIdx: 0, Timestamp: ts},
				},
				chainB: {
					0: {ChainID: chainC, LogIdx: 0, Timestamp: ts},
				},
				chainC: {
					0: {ChainID: chainA, LogIdx: 0, Timestamp: ts},
				},
				chainD: {
					0: {ChainID: eth.ChainIDFromUInt64(12345), LogIdx: 0, Timestamp: ts}, // refs non-existent chain
				},
			},
			expectCycleChains:    []eth.ChainID{chainA, chainB, chainC},
			expectNonCycleChains: []eth.ChainID{chainD},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			graph := buildCycleGraph(ts, tc.chainEMs)
			err := checkCycle(graph)
			require.Error(t, err, "expected cycle to be detected")

			// Collect chains with unresolved nodes (actual cycle participants)
			cycleChains := make(map[eth.ChainID]bool)
			for _, node := range *graph {
				if !node.resolved {
					cycleChains[node.chainID] = true
				}
			}

			// Verify expected cycle chains are in the cycle
			for _, chainID := range tc.expectCycleChains {
				require.True(t, cycleChains[chainID],
					"chain %v should be part of the cycle", chainID)
			}

			// Verify bystander chains are NOT in the cycle
			for _, chainID := range tc.expectNonCycleChains {
				require.False(t, cycleChains[chainID],
					"chain %v should NOT be part of the cycle (bystander)", chainID)
			}
		})
	}
}
