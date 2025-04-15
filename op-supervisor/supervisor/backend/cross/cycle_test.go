package cross

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const testDefaultTimestamp = 100

var errUnexpectedChain = errors.New("unexpected chain")

type testDepSet struct {
	mapping map[types.ChainIndex]eth.ChainID
}

func (t testDepSet) ChainIDFromIndex(index types.ChainIndex) (eth.ChainID, error) {
	v, ok := t.mapping[index]
	if !ok {
		return eth.ChainID{}, types.ErrUnknownChain
	}
	return v, nil
}

var _ depset.ChainIDFromIndex = (*testDepSet)(nil)

type mockCycleCheckDeps struct {
	openBlockFn func(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error)
}

func (m *mockCycleCheckDeps) OpenBlock(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
	return m.openBlockFn(chainID, blockNum)
}

type chainBlockDef struct {
	logCount uint32
	messages map[uint32]*types.ExecutingMessage
	error    error
}

type hazardCycleChecksTestCase struct {
	name              string
	chainBlocks       map[string]chainBlockDef
	expectErr         error
	expectErrContains string
	msg               string

	// Optional overrides
	hazards     map[types.ChainIndex]types.BlockSeal
	openBlockFn func(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error)
}

func runHazardCycleChecksTestCaseGroup(t *testing.T, group string, tests []hazardCycleChecksTestCase) {
	for _, tc := range tests {
		t.Run(group+"/"+tc.name, func(t *testing.T) {
			runHazardCycleChecksTestCase(t, tc)
		})
	}
}

func runHazardCycleChecksTestCase(t *testing.T, tc hazardCycleChecksTestCase) {
	// Create mocked dependencies
	deps := &mockCycleCheckDeps{
		openBlockFn: func(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
			// Use override if provided
			if tc.openBlockFn != nil {
				return tc.openBlockFn(chainID, blockNum)
			}

			// Default behavior
			chainStr := chainID.String()
			def, ok := tc.chainBlocks[chainStr]
			if !ok {
				return eth.BlockRef{}, 0, nil, errUnexpectedChain
			}
			if def.error != nil {
				return eth.BlockRef{}, 0, nil, def.error
			}
			return eth.BlockRef{Number: blockNum}, def.logCount, def.messages, nil
		},
	}

	// Generate hazards map automatically if not explicitly provided
	var hazards map[types.ChainIndex]types.BlockSeal
	if tc.hazards != nil {
		hazards = tc.hazards
	} else {
		hazards = make(map[types.ChainIndex]types.BlockSeal)
		for chainStr := range tc.chainBlocks {
			hazards[chainIndex(chainStr)] = types.BlockSeal{Number: 1}
		}
	}

	depSet := &testDepSet{
		mapping: make(map[types.ChainIndex]eth.ChainID),
	}
	for chainStr := range tc.chainBlocks {
		index := chainIndex(chainStr)
		depSet.mapping[index] = eth.ChainIDFromUInt64(uint64(index))
	}
	// Run the test
	err := HazardCycleChecks(depSet, deps, testDefaultTimestamp, NewHazardSetFromEntries(hazards))

	if tc.expectErr != nil && tc.expectErrContains != "" {
		require.Fail(t, "expectErr and expectErrContains cannot both be set in a test case")
		return
	}

	switch {
	case tc.expectErr == nil && tc.expectErrContains == "":
		require.NoError(t, err, tc.msg)
	case tc.expectErr != nil:
		require.ErrorIs(t, err, tc.expectErr, tc.msg)
	case tc.expectErrContains != "":
		require.Error(t, err, tc.msg)
		require.Contains(t, err.Error(), tc.expectErrContains, tc.msg)
	}
}

func chainIndex(s string) types.ChainIndex {
	id, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("invalid chain index in test: %v", err))
	}
	return types.ChainIndex(id)
}

func execMsg(chain string, logIdx uint32) *types.ExecutingMessage {
	return execMsgWithTimestamp(chain, logIdx, testDefaultTimestamp)
}

func execMsgWithTimestamp(chain string, logIdx uint32, timestamp uint64) *types.ExecutingMessage {
	return &types.ExecutingMessage{
		Chain:     chainIndex(chain),
		LogIdx:    logIdx,
		Timestamp: timestamp,
	}
}

var emptyChainBlocks = map[string]chainBlockDef{
	"1": {
		logCount: 0,
		messages: map[uint32]*types.ExecutingMessage{},
	},
}

var (
	errTestOpenBlock = errors.New("test OpenBlock error")
)

func TestHazardCycleChecksFailures(t *testing.T) {
	tests := []hazardCycleChecksTestCase{
		{
			name:        "empty hazards",
			chainBlocks: emptyChainBlocks,
			hazards:     make(map[types.ChainIndex]types.BlockSeal),
			expectErr:   nil,
			msg:         "expected no error when there are no hazards",
		},
		{
			name:        "nil hazards",
			chainBlocks: emptyChainBlocks,
			hazards:     nil,
			expectErr:   nil,
			msg:         "expected no error when there are nil hazards",
		},
		{
			name:        "nil blocks",
			chainBlocks: nil,
			hazards:     nil,
			expectErr:   nil,
			msg:         "expected no error when there are nil blocks and hazards",
		},
		{
			name:        "failed to open block error",
			chainBlocks: emptyChainBlocks,
			openBlockFn: func(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
				return eth.BlockRef{}, 0, nil, errTestOpenBlock
			},
			expectErrContains: "failed to open block",
			msg:               "expected error when OpenBlock fails",
		},
		{
			name:        "block mismatch error",
			chainBlocks: emptyChainBlocks,
			// openBlockFn returns a block number that doesn't match the expected block number.
			openBlockFn: func(chainID eth.ChainID, blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
				return eth.BlockRef{Number: blockNum + 1}, 0, make(map[uint32]*types.ExecutingMessage), nil
			},
			expectErr: errInconsistentBlockSeal,
			msg:       "expected error due to block mismatch",
		},
		{
			name: "multiple blocks with messages",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("2", 0),
						1: execMsg("2", 1),
					},
				},
				"2": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 0),
						1: execMsg("1", 1),
					},
				},
			},
			hazards: map[types.ChainIndex]types.BlockSeal{
				1: {Number: 1},
				2: {Number: 1},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle error with multiple blocks and messages",
		},
		{
			name: "invalid log index error",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						5: execMsg("1", 0), // Invalid index >= logCount.
					},
				},
			},
			expectErr: ErrExecMsgHasInvalidIndex,
			msg:       "expected invalid log index error",
		},
		{
			name: "self reference detected error",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 1,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 0), // Points at itself.
					},
				},
			},
			expectErr: types.ErrConflict,
			msg:       "expected self reference detection error",
		},
		{
			name: "unknown chain",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("2", 0), // References chain 2 which isn't in hazards.
					},
				},
			},
			hazards: map[types.ChainIndex]types.BlockSeal{
				1: {Number: 1}, // Only include chain 1.
			},
			expectErr: ErrExecMsgUnknownChain,
			msg:       "expected unknown chain error",
		},
	}
	runHazardCycleChecksTestCaseGroup(t, "Failure", tests)
}

func TestHazardCycleChecksNoCycle(t *testing.T) {
	tests := []hazardCycleChecksTestCase{
		{
			name:        "no logs",
			chainBlocks: emptyChainBlocks,
			expectErr:   nil,
			msg:         "expected no cycle found for block with no logs",
		},
		{
			name: "one basic log",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 1,
					messages: map[uint32]*types.ExecutingMessage{},
				},
			},
			msg: "expected no cycle found for single basic log",
		},
		{
			name: "one exec log",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 0),
					},
				},
			},
			msg: "expected no cycle found for single exec log",
		},
		{
			name: "two basic logs",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{},
				},
			},
			msg: "expected no cycle found for two basic logs",
		},
		{
			name: "two exec logs to same target",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 0),
						2: execMsg("1", 0),
					},
				},
			},
			msg: "expected no cycle found for two exec logs pointing at the same log",
		},
		{
			name: "two exec logs to different targets",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 0),
						2: execMsg("1", 1),
					},
				},
			},
			msg: "expected no cycle found for two exec logs pointing at the different logs",
		},
		{
			name: "one basic log one exec log",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 0),
					},
				},
			},
			msg: "expected no cycle found for one basic and one exec log",
		},
		{
			name: "first log is exec",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 1,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("2", 0),
					},
				},
				"2": {
					logCount: 1,
					messages: nil,
				},
			},
			msg: "expected no cycle found first log is exec",
		},
		{
			name: "no cycle using different timestamp",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsgWithTimestamp("1", 1, testDefaultTimestamp+1),
					},
				},
			},
			msg: "expected no cycle for different timestamp",
		},
	}
	runHazardCycleChecksTestCaseGroup(t, "NoCycle", tests)
}

func TestHazardCycleChecksCycle(t *testing.T) {
	// Comment cycle notation: `executing message -> corresponding initiating message`
	// The index of the log itself is used as name of the message.
	// For different chains, "A" or "B", etc. may be prefixed, to identify the chain with the corresponding chain index. (A=0, B=1, etc.)
	tests := []hazardCycleChecksTestCase{
		{
			// 0->2->1->0  - executing message pointing to the future, cycle completed by regular log ordering
			name: "3-cycle in single chain",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 2),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// 0->2->0   - both the executing messages
			// 0->2->1->0  - first executing message combined with regular log-ordering dependencies
			name: "3-cycle in single chain, 2-cycle in single chain with first log",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 2),
						2: execMsg("1", 0),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// 0->1->0
			name: "2-cycle in single chain with first log, adjacent",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 1),
						1: execMsg("1", 0),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// 1->2->1
			name: "2-cycle in single chain, not first, adjacent",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 2),
						2: execMsg("1", 1),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// 1->3->1  - two executing messages
			// 1->3->2->1  - one executing message and regular log ordering
			name: "2,3-cycle in single chain, not first, not adjacent",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 4,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 3),
						3: execMsg("1", 1),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// A1->B0->A1
			name: "2-cycle across chains",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("2", 0),
					},
				},
				"2": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 1),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error for cycle through executing messages",
		},
		{
			// 1->2->1  - 1 executes the next log, forming a small cycle
			// 2->3->2  - 2 executes the next log, forming a small cycle
			// 3->1->2->3  - (3->1) is a valid executing message, but part of a larger cycle where 2 and 1 depend on the next future log.
			// 3->2->1->2->3  - we have the regular order of logs, and then multiple logs pointing to the future logs, making an even larger cycle.
			name: "2,2,3-cycle in single chain",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 4,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 2), // Points to log 2
						2: execMsg("1", 3), // Points to log 3
						3: execMsg("1", 1), // Points back to log 1
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error for 3-node cycle",
		},
		{
			// 1->5->4->3->2->1  - executing message, and multiple regular log ordering steps
			// 1->5->2->1 - two executing messages and single regular log ordering dependency
			name: "cycle through adjacency dependency",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 10,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 5), // Points to log 5
						5: execMsg("1", 2), // Points back to log 2 which is adjacent to log 1
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error for when cycle goes through adjacency dependency",
		},
		{
			// A1->B1->A1
			name: "2-cycle across chains with 3 hazard chains",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("2", 1),
					},
				},
				"2": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						1: execMsg("1", 1),
					},
				},
				"3": {},
			},
			expectErr: ErrCycle,
			hazards: map[types.ChainIndex]types.BlockSeal{
				1: {Number: 1},
				2: {Number: 1},
				3: {Number: 1},
			},
			msg: "expected cycle detection error for cycle through executing messages",
		},
		{
			// 0->1->0
			name: "cycle through single chain, exec message prior to init and adjacent",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 1),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// 0->2->1->0
			name: "cycle through single chain, exec message prior to init and not adjacent",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 3,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 2),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
		{
			// A0->B0->A1->A0  - A may not depend on a log of B that depends on the future of A
			name: "3-cycle across chains",
			chainBlocks: map[string]chainBlockDef{
				"1": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("2", 0),
					},
				},
				"2": {
					logCount: 2,
					messages: map[uint32]*types.ExecutingMessage{
						0: execMsg("1", 1),
					},
				},
			},
			expectErr: ErrCycle,
			msg:       "expected cycle detection error",
		},
	}
	runHazardCycleChecksTestCaseGroup(t, "Cycle", tests)
}

const (
	largeGraphChains       = 10
	largeGraphLogsPerChain = 10000
)

func TestHazardCycleChecksLargeGraphNoCycle(t *testing.T) {
	// Create a large but acyclic graph
	chainBlocks := make(map[string]chainBlockDef)
	for i := 1; i <= largeGraphChains; i++ {
		msgs := make(map[uint32]*types.ExecutingMessage)
		// Create a chain of dependencies across chains
		if i > 1 {
			for j := uint32(0); j < largeGraphLogsPerChain; j++ {
				// Point to previous chain, same log index
				msgs[j] = execMsg(strconv.Itoa(i-1), j)
			}
		}
		chainBlocks[strconv.Itoa(i)] = chainBlockDef{
			logCount: largeGraphLogsPerChain,
			messages: msgs,
		}
	}

	tc := hazardCycleChecksTestCase{
		name:        "Large graph without cycles",
		chainBlocks: chainBlocks,
		expectErr:   nil,
		msg:         "expected no cycle in large acyclic graph",
	}
	runHazardCycleChecksTestCase(t, tc)
}

func TestHazardCycleChecksLargeGraphCycle(t *testing.T) {
	// Create a large graph with a cycle hidden in it
	const cycleChain = 3
	const cycleLogIndex = 5678

	chainBlocks := make(map[string]chainBlockDef)
	for i := 1; i <= largeGraphChains; i++ {
		msgs := make(map[uint32]*types.ExecutingMessage)
		// Create a chain of dependencies across chains
		if i > 1 {
			for j := uint32(0); j < largeGraphLogsPerChain; j++ {
				if i == cycleChain && j == cycleLogIndex {
					// Create a cycle by pointing back to chain 1
					msgs[j] = execMsg("1", cycleLogIndex+1)
				} else {
					// Normal case: point to previous chain, same log index
					msgs[j] = execMsg(strconv.Itoa(i-1), j)
				}
			}
		} else {
			// In chain 1, create the other side of the cycle
			msgs[cycleLogIndex+1] = execMsg(strconv.Itoa(cycleChain), cycleLogIndex)
		}

		chainBlocks[strconv.Itoa(i)] = chainBlockDef{
			logCount: largeGraphLogsPerChain,
			messages: msgs,
		}
	}

	tc := hazardCycleChecksTestCase{
		name:        "Large graph with cycle",
		chainBlocks: chainBlocks,
		expectErr:   ErrCycle,
		msg:         "expected to detect cycle in large cyclic graph",
	}
	runHazardCycleChecksTestCase(t, tc)
}
