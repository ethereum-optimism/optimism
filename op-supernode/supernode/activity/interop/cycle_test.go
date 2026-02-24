package interop

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Test Helpers - Common Graph Patterns
// =============================================================================

var (
	testChainA = eth.ChainIDFromUInt64(10)
	testChainB = eth.ChainIDFromUInt64(8453)
	testChainC = eth.ChainIDFromUInt64(420)
	testChainD = eth.ChainIDFromUInt64(999)
	testTS     = uint64(1000)
)

// mutualCycle creates A↔B cycle at log index 0
func mutualCycle(a, b eth.ChainID) map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage {
	return map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
		a: {0: {ChainID: b, LogIdx: 0, Timestamp: testTS}},
		b: {0: {ChainID: a, LogIdx: 0, Timestamp: testTS}},
	}
}

// triangleCycle creates A→B→C→A cycle at log index 0
func triangleCycle(a, b, c eth.ChainID) map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage {
	return map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
		a: {0: {ChainID: b, LogIdx: 0, Timestamp: testTS}},
		b: {0: {ChainID: c, LogIdx: 0, Timestamp: testTS}},
		c: {0: {ChainID: a, LogIdx: 0, Timestamp: testTS}},
	}
}

// oneWayRef creates a one-way reference from chain 'from' to chain 'to'
func oneWayRef(from, to eth.ChainID, fromLogIdx, toLogIdx uint32) map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage {
	return map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
		from: {fromLogIdx: {ChainID: to, LogIdx: toLogIdx, Timestamp: testTS}},
	}
}

// mergeEMs merges multiple EM maps into one
func mergeEMs(maps ...map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage) map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage {
	result := make(map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage)
	for _, m := range maps {
		for chainID, ems := range m {
			if result[chainID] == nil {
				result[chainID] = make(map[uint32]*suptypes.ExecutingMessage)
			}
			for logIdx, em := range ems {
				result[chainID][logIdx] = em
			}
		}
	}
	return result
}

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

	tests := []struct {
		name             string
		chainEMs         map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage
		expectCycle      bool
		expectInCycle    []eth.ChainID // chains that should be in the cycle (only checked if expectCycle)
		expectNotInCycle []eth.ChainID // chains that should NOT be in cycle (bystanders)
	}{
		{
			name:        "empty graph - no cycle",
			chainEMs:    map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{},
			expectCycle: false,
		},
		{
			name: "past timestamp filtered out",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				testChainA: {0: {ChainID: testChainB, LogIdx: 0, Timestamp: testTS - 100}},
			},
			expectCycle: false,
		},
		{
			name:        "one-way ref to chain with no EMs - no cycle",
			chainEMs:    oneWayRef(testChainA, testChainB, 0, 0),
			expectCycle: false,
		},
		{
			name:          "mutual cycle A↔B",
			chainEMs:      mutualCycle(testChainA, testChainB),
			expectCycle:   true,
			expectInCycle: []eth.ChainID{testChainA, testChainB},
		},
		{
			name:          "triangle cycle A→B→C→A",
			chainEMs:      triangleCycle(testChainA, testChainB, testChainC),
			expectCycle:   true,
			expectInCycle: []eth.ChainID{testChainA, testChainB, testChainC},
		},
		{
			name: "A↔C cycle with B as bystander",
			chainEMs: mergeEMs(
				mutualCycle(testChainA, testChainC),
				oneWayRef(testChainB, testChainD, 0, 0), // B refs non-existent D
			),
			expectCycle:      true,
			expectInCycle:    []eth.ChainID{testChainA, testChainC},
			expectNotInCycle: []eth.ChainID{testChainB},
		},
		{
			name: "one-way dependency - no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				testChainA: {0: {ChainID: testChainB, LogIdx: 5, Timestamp: testTS}},
				testChainB: {3: {ChainID: testChainC, LogIdx: 0, Timestamp: testTS}},
			},
			expectCycle: false,
		},
		{
			name: "ref before target EM - no dependency, no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				testChainA: {0: {ChainID: testChainB, LogIdx: 2, Timestamp: testTS}}, // refs B:2
				testChainB: {3: {ChainID: testChainA, LogIdx: 0, Timestamp: testTS}}, // B:3 > 2, no match
			},
			expectCycle: false,
		},
		{
			name: "intra-chain sequential EMs - no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				testChainA: {
					0: {ChainID: testChainB, LogIdx: 0, Timestamp: testTS},
					5: {ChainID: testChainB, LogIdx: 3, Timestamp: testTS},
				},
			},
			expectCycle: false,
		},
		{
			name: "triangle with missing leg - no cycle",
			chainEMs: map[eth.ChainID]map[uint32]*suptypes.ExecutingMessage{
				testChainA: {5: {ChainID: testChainB, LogIdx: 3, Timestamp: testTS}},
				testChainB: {5: {ChainID: testChainC, LogIdx: 3, Timestamp: testTS}},
				testChainC: {5: {ChainID: testChainA, LogIdx: 3, Timestamp: testTS}}, // A:5 > 3
			},
			expectCycle: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			graph := buildCycleGraph(testTS, tc.chainEMs)
			err := checkCycle(graph)

			if tc.expectCycle {
				require.Error(t, err, "expected cycle")

				// Verify cycle participants
				cycleChains := make(map[eth.ChainID]bool)
				for _, node := range *graph {
					if !node.resolved {
						cycleChains[node.chainID] = true
					}
				}
				for _, c := range tc.expectInCycle {
					require.True(t, cycleChains[c], "chain %v should be in cycle", c)
				}
				for _, c := range tc.expectNotInCycle {
					require.False(t, cycleChains[c], "chain %v should NOT be in cycle", c)
				}
			} else {
				require.NoError(t, err, "expected no cycle")
			}
		})
	}
}
