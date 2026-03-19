package interop

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// TestVerifyInteropMessages - Table-Driven Tests
// =============================================================================

// newMockChainWithL1 creates a mock chain with the specified L1 block for OptimisticAt
func newMockChainWithL1(chainID eth.ChainID, l1Block eth.BlockID) *algoMockChain {
	return &algoMockChain{
		id:           chainID,
		optimisticL1: l1Block,
	}
}

// verifyInteropTestCase defines a single test case for verifyInteropMessages
type verifyInteropTestCase struct {
	name        string
	setup       func() (*Interop, uint64, map[eth.ChainID]eth.BlockID)
	expectError bool
	errorMsg    string
	validate    func(t *testing.T, result Result)
}

func runVerifyInteropTest(t *testing.T, tc verifyInteropTestCase) {
	t.Parallel()
	interop, timestamp, blocks := tc.setup()
	result, err := interop.verifyInteropMessages(timestamp, blocks)

	if tc.expectError {
		require.Error(t, err)
		if tc.errorMsg != "" {
			require.Contains(t, err.Error(), tc.errorMsg)
		}
	} else {
		require.NoError(t, err)
	}

	if tc.validate != nil {
		tc.validate(t, result)
	}
}

func TestL1Inclusion(t *testing.T) {
	t.Parallel()

	type l1InclusionTestCase struct {
		name        string
		setup       func() (*Interop, uint64, map[eth.ChainID]eth.BlockID)
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, l1 eth.BlockID)
	}

	tests := []l1InclusionTestCase{
		{
			name: "SingleChain",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)
				expectedBlock := eth.BlockID{Number: 100, Hash: common.HexToHash("0x123")}
				l1Block := eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1")}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains:  map[eth.ChainID]cc.ChainContainer{chainID: &algoMockChain{id: chainID, optimisticL1: l1Block}},
				}
				return interop, 1000, map[eth.ChainID]eth.BlockID{chainID: expectedBlock}
			},
			validate: func(t *testing.T, l1 eth.BlockID) {
				require.Equal(t, eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1")}, l1)
			},
		},
		{
			name: "MultipleChains_HighestL1Selected",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chain1ID := eth.ChainIDFromUInt64(10)
				chain2ID := eth.ChainIDFromUInt64(8453)
				chain3ID := eth.ChainIDFromUInt64(420)

				// Chain 1 has L1 at 60 (highest - should be selected)
				// Chain 2 has L1 at 45 (earliest)
				// Chain 3 has L1 at 50 (middle)
				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains: map[eth.ChainID]cc.ChainContainer{
						chain1ID: &algoMockChain{id: chain1ID, optimisticL1: eth.BlockID{Number: 60, Hash: common.HexToHash("0xL1_1")}},
						chain2ID: &algoMockChain{id: chain2ID, optimisticL1: eth.BlockID{Number: 45, Hash: common.HexToHash("0xL1_2")}},
						chain3ID: &algoMockChain{id: chain3ID, optimisticL1: eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1_3")}},
					},
				}
				return interop, 1000, map[eth.ChainID]eth.BlockID{
					chain1ID: {Number: 100, Hash: common.HexToHash("0x1")},
					chain2ID: {Number: 200, Hash: common.HexToHash("0x2")},
					chain3ID: {Number: 150, Hash: common.HexToHash("0x3")},
				}
			},
			validate: func(t *testing.T, l1 eth.BlockID) {
				require.Equal(t, eth.BlockID{Number: 60, Hash: common.HexToHash("0xL1_1")}, l1)
			},
		},
		{
			name: "ChainNotInChainsMap_Skipped",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chain1ID := eth.ChainIDFromUInt64(10)
				chain2ID := eth.ChainIDFromUInt64(8453) // Not in chains map

				l1Block1 := eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1_1")}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains: map[eth.ChainID]cc.ChainContainer{
						chain1ID: &algoMockChain{id: chain1ID, optimisticL1: l1Block1},
						// chain2ID NOT in chains map
					},
				}
				return interop, 1000, map[eth.ChainID]eth.BlockID{
					chain1ID: {Number: 100, Hash: common.HexToHash("0x1")},
					chain2ID: {Number: 200, Hash: common.HexToHash("0x2")},
				}
			},
			validate: func(t *testing.T, l1 eth.BlockID) {
				require.Equal(t, eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1_1")}, l1)
			},
		},
		{
			name: "OptimisticAtError_ReturnsError",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains: map[eth.ChainID]cc.ChainContainer{
						chainID: &algoMockChain{id: chainID, optimisticAtErr: errors.New("optimistic at error")},
					},
				}
				return interop, 1000, map[eth.ChainID]eth.BlockID{
					chainID: {Number: 100, Hash: common.HexToHash("0x123")},
				}
			},
			expectError: true,
			errorMsg:    "failed to get L1 inclusion",
		},
		{
			name: "NoChains_ReturnsEmpty",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains:  map[eth.ChainID]cc.ChainContainer{},
				}
				return interop, 1000, map[eth.ChainID]eth.BlockID{}
			},
			validate: func(t *testing.T, l1 eth.BlockID) {
				require.Equal(t, eth.BlockID{}, l1)
			},
		},
		{
			name: "GenesisBlock_NoError",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)
				// L1 genesis block at number 0
				l1Block := eth.BlockID{Number: 0, Hash: common.HexToHash("0xGenesisL1")}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{},
					chains:  map[eth.ChainID]cc.ChainContainer{chainID: &algoMockChain{id: chainID, optimisticL1: l1Block}},
				}
				return interop, 0, map[eth.ChainID]eth.BlockID{
					chainID: {Number: 0, Hash: common.HexToHash("0x123")},
				}
			},
			// Genesis blocks included at L1 block number 0 must not cause an error.
			validate: func(t *testing.T, l1 eth.BlockID) {
				require.Equal(t, eth.BlockID{Number: 0, Hash: common.HexToHash("0xGenesisL1")}, l1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			interop, ts, blocks := tc.setup()
			l1, err := interop.l1Inclusion(ts, blocks)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}

			if tc.validate != nil {
				tc.validate(t, l1)
			}
		})
	}
}

func TestVerifyInteropMessages(t *testing.T) {
	t.Parallel()

	tests := []verifyInteropTestCase{
		// Valid block cases
		{
			name: "ValidBlocks/NoExecutingMessages",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)
				blockHash := common.HexToHash("0x123")
				expectedBlock := eth.BlockID{Number: 100, Hash: blockHash}
				l1Block := eth.BlockID{Number: 50, Hash: common.HexToHash("0xL1")}

				mockDB := &algoMockLogsDB{
					openBlockRef:     eth.BlockRef{Hash: blockHash, Number: 100, Time: 1000},
					openBlockExecMsg: nil,
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
					chains:  map[eth.ChainID]cc.ChainContainer{chainID: newMockChainWithL1(chainID, l1Block)},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{chainID: expectedBlock}
			},
			validate: func(t *testing.T, result Result) {
				chainID := eth.ChainIDFromUInt64(10)
				expectedBlock := eth.BlockID{Number: 100, Hash: common.HexToHash("0x123")}
				require.True(t, result.IsValid())
				require.Empty(t, result.InvalidHeads)
				require.Equal(t, expectedBlock, result.L2Heads[chainID])
			},
		},
		{
			name: "ValidBlocks/ValidExecutingMessage",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				sourceBlockHash := common.HexToHash("0xSource")
				destBlockHash := common.HexToHash("0xDest")

				sourceBlock := eth.BlockID{Number: 50, Hash: sourceBlockHash}
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}
				l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 500, // Source timestamp < dest timestamp (1000)
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: sourceBlockHash, Number: 50, Time: 500},
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 500},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: 1000},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, l1Block),
						destChainID:   newMockChainWithL1(destChainID, l1Block),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{
					sourceChainID: sourceBlock,
					destChainID:   destBlock,
				}
			},
			validate: func(t *testing.T, result Result) {
				require.True(t, result.IsValid())
				require.Empty(t, result.InvalidHeads)
			},
		},
		{
			name: "ValidBlocks/MessageAtExpiryBoundary",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				sourceBlockHash := common.HexToHash("0xSource")
				destBlockHash := common.HexToHash("0xDest")

				// Message is exactly at the expiry boundary (should pass)
				execTimestamp := uint64(1000000)
				initTimestamp := execTimestamp - ExpiryTime // Exactly at boundary

				sourceBlock := eth.BlockID{Number: 50, Hash: sourceBlockHash}
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}
				l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: initTimestamp, // Exactly at expiry boundary
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: sourceBlockHash, Number: 50, Time: initTimestamp},
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: initTimestamp},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: execTimestamp},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, l1Block),
						destChainID:   newMockChainWithL1(destChainID, l1Block),
					},
				}

				return interop, execTimestamp, map[eth.ChainID]eth.BlockID{
					sourceChainID: sourceBlock,
					destChainID:   destBlock,
				}
			},
			validate: func(t *testing.T, result Result) {
				require.True(t, result.IsValid())
				require.Empty(t, result.InvalidHeads)
			},
		},
		{
			name: "ValidBlocks/SameTimestampMessage",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				// Same-timestamp interop: executing message references an initiating message
				// from the SAME timestamp.
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				sourceBlockHash := common.HexToHash("0xSource")
				destBlockHash := common.HexToHash("0xDest")

				// Both blocks at the SAME timestamp
				sharedTimestamp := uint64(1000)

				sourceBlock := eth.BlockID{Number: 50, Hash: sourceBlockHash}
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: sharedTimestamp, // SAME as executing timestamp - should be VALID
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: sourceBlockHash, Number: 50, Time: sharedTimestamp},
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: sharedTimestamp},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: sharedTimestamp},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, l1Block),
						destChainID:   newMockChainWithL1(destChainID, l1Block),
					},
				}

				return interop, sharedTimestamp, map[eth.ChainID]eth.BlockID{
					sourceChainID: sourceBlock,
					destChainID:   destBlock,
				}
			},
			validate: func(t *testing.T, result Result) {
				// Same-timestamp messages should now be VALID
				require.True(t, result.IsValid(), "same-timestamp messages should be valid")
				require.Empty(t, result.InvalidHeads, "no blocks should be invalid")
			},
		},
		{
			// Interop verification *never* expects to be given chain data for chains that are not part of the supernode,
			// so this test is not helpful except to demonstrate the specified behavior: if chain data is available
			// but is not part of the chains map for some reason, it should not be used at all, as it is unrelated to the
			// superchain's interop verification.
			name: "ValidBlocks/UnregisteredChainsSkipped",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				registeredChain := eth.ChainIDFromUInt64(10)
				unregisteredChain := eth.ChainIDFromUInt64(9999)

				mockDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: common.HexToHash("0x1"), Number: 100, Time: 1000},
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{registeredChain: mockDB},
					chains: map[eth.ChainID]cc.ChainContainer{
						registeredChain: newMockChainWithL1(registeredChain, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{
					registeredChain:   {Number: 100, Hash: common.HexToHash("0x1")},
					unregisteredChain: {Number: 200, Hash: common.HexToHash("0x2")},
				}
			},
			validate: func(t *testing.T, result Result) {
				registeredChain := eth.ChainIDFromUInt64(10)
				unregisteredChain := eth.ChainIDFromUInt64(9999)
				require.True(t, result.IsValid())
				require.Contains(t, result.L2Heads, registeredChain)
				require.NotContains(t, result.L2Heads, unregisteredChain)
			},
		},
		// Invalid block cases
		{
			name: "InvalidBlocks/BlockHashMismatch",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)
				expectedBlock := eth.BlockID{Number: 100, Hash: common.HexToHash("0xExpected")}
				l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}

				mockDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{
						Hash:   common.HexToHash("0xActual"), // Different from expected
						Number: 100,
						Time:   1000,
					},
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
					chains:  map[eth.ChainID]cc.ChainContainer{chainID: newMockChainWithL1(chainID, l1Block)},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{chainID: expectedBlock}
			},
			validate: func(t *testing.T, result Result) {
				chainID := eth.ChainIDFromUInt64(10)
				expectedBlock := eth.BlockID{Number: 100, Hash: common.HexToHash("0xExpected")}
				require.False(t, result.IsValid())
				require.Contains(t, result.InvalidHeads, chainID)
				require.Equal(t, expectedBlock, result.InvalidHeads[chainID])
			},
		},
		{
			name: "InvalidBlocks/InitiatingMessageNotFound",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				destBlockHash := common.HexToHash("0xDest")
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 500,
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					containsErr: suptypes.ErrConflict, // Message not found
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: 1000},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						destChainID:   newMockChainWithL1(destChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{destChainID: destBlock}
			},
			validate: func(t *testing.T, result Result) {
				destChainID := eth.ChainIDFromUInt64(8453)
				require.False(t, result.IsValid())
				require.Contains(t, result.InvalidHeads, destChainID)
			},
		},
		{
			name: "InvalidBlocks/FutureTimestamp",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				// Future timestamp: initiating message timestamp > executing timestamp.
				// This is INVALID (you can't execute a message that hasn't been initiated yet).
				// Note: Same-timestamp (==) is ALLOWED, only strictly greater (>) is invalid.
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				destBlockHash := common.HexToHash("0xDest")
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 1001, // FUTURE timestamp (> 1000) - INVALID!
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 1001},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: 1000},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						destChainID:   newMockChainWithL1(destChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{destChainID: destBlock}
			},
			validate: func(t *testing.T, result Result) {
				destChainID := eth.ChainIDFromUInt64(8453)
				require.False(t, result.IsValid(), "future timestamp messages should be invalid")
				require.Contains(t, result.InvalidHeads, destChainID)
			},
		},
		{
			name: "InvalidBlocks/UnknownSourceChain",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				unknownSourceChain := eth.ChainIDFromUInt64(9999)
				destChainID := eth.ChainIDFromUInt64(8453)

				destBlockHash := common.HexToHash("0xDest")
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   unknownSourceChain, // Not registered
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 500,
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: 1000},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						destChainID: destDB,
						// Note: unknownSourceChain NOT in logsDBs
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						unknownSourceChain: newMockChainWithL1(unknownSourceChain, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						destChainID:        newMockChainWithL1(destChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{destChainID: destBlock}
			},
			validate: func(t *testing.T, result Result) {
				destChainID := eth.ChainIDFromUInt64(8453)
				require.False(t, result.IsValid())
				require.Contains(t, result.InvalidHeads, destChainID)
			},
		},
		{
			name: "InvalidBlocks/ExpiredMessage",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				destBlockHash := common.HexToHash("0xDest")
				// Executing block is at timestamp 1000000 (well after expiry)
				execTimestamp := uint64(1000000)
				// Initiating message timestamp is more than ExpiryTime (604800) before executing timestamp
				initTimestamp := execTimestamp - ExpiryTime - 1 // 1 second past expiry

				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: initTimestamp, // Expired!
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: initTimestamp},
				}

				destDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: destBlockHash, Number: 100, Time: execTimestamp},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: execMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID: sourceDB,
						destChainID:   destDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						sourceChainID: newMockChainWithL1(sourceChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						destChainID:   newMockChainWithL1(destChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, execTimestamp, map[eth.ChainID]eth.BlockID{destChainID: destBlock}
			},
			validate: func(t *testing.T, result Result) {
				destChainID := eth.ChainIDFromUInt64(8453)
				require.False(t, result.IsValid())
				require.Contains(t, result.InvalidHeads, destChainID)
			},
		},
		{
			name: "InvalidBlocks/MultipleChainsOneInvalid",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				validChainID := eth.ChainIDFromUInt64(8453)
				invalidChainID := eth.ChainIDFromUInt64(420)

				validBlockHash := common.HexToHash("0xValid")
				invalidBlockHash := common.HexToHash("0xInvalid")

				validBlock := eth.BlockID{Number: 100, Hash: validBlockHash}
				invalidBlock := eth.BlockID{Number: 200, Hash: invalidBlockHash}

				badExecMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 1001, // Future timestamp - INVALID
					Checksum:  suptypes.MessageChecksum{0x01},
				}

				sourceDB := &algoMockLogsDB{
					containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 1001},
				}

				validDB := &algoMockLogsDB{
					openBlockRef:     eth.BlockRef{Hash: validBlockHash, Number: 100, Time: 1000},
					openBlockExecMsg: nil, // No executing messages - valid
				}

				invalidDB := &algoMockLogsDB{
					openBlockRef: eth.BlockRef{Hash: invalidBlockHash, Number: 200, Time: 1000},
					openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
						0: badExecMsg,
					},
				}

				interop := &Interop{
					log: gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{
						sourceChainID:  sourceDB,
						validChainID:   validDB,
						invalidChainID: invalidDB,
					},
					chains: map[eth.ChainID]cc.ChainContainer{
						invalidChainID: newMockChainWithL1(invalidChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						sourceChainID:  newMockChainWithL1(sourceChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
						validChainID:   newMockChainWithL1(validChainID, eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}),
					},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{
					validChainID:   validBlock,
					invalidChainID: invalidBlock,
				}
			},
			validate: func(t *testing.T, result Result) {
				validChainID := eth.ChainIDFromUInt64(8453)
				invalidChainID := eth.ChainIDFromUInt64(420)
				require.False(t, result.IsValid())
				// Both chains in L2Heads
				require.Contains(t, result.L2Heads, validChainID)
				require.Contains(t, result.L2Heads, invalidChainID)
				// Only invalid in InvalidHeads
				require.NotContains(t, result.InvalidHeads, validChainID)
				require.Contains(t, result.InvalidHeads, invalidChainID)
			},
		},
		// Error cases
		{
			name: "Errors/OpenBlockError",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				chainID := eth.ChainIDFromUInt64(10)
				block := eth.BlockID{Number: 100, Hash: common.HexToHash("0x123")}
				l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0xL1")}

				mockDB := &algoMockLogsDB{
					openBlockErr: errors.New("database error"),
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
					chains:  map[eth.ChainID]cc.ChainContainer{chainID: newMockChainWithL1(chainID, l1Block)},
				}

				return interop, 1000, map[eth.ChainID]eth.BlockID{chainID: block}
			},
			expectError: true,
			errorMsg:    "database error",
			validate: func(t *testing.T, result Result) {
				require.True(t, result.IsEmpty())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runVerifyInteropTest(t, tc)
		})
	}
}

// =============================================================================
// Mock Types for Algorithm Tests
// =============================================================================

// algoMockLogsDB is a mock LogsDB for algorithm tests
type algoMockLogsDB struct {
	openBlockRef     eth.BlockRef
	openBlockLogCnt  uint32
	openBlockExecMsg map[uint32]*suptypes.ExecutingMessage
	openBlockErr     error

	firstSealedBlock    suptypes.BlockSeal
	firstSealedBlockErr error

	containsSeal suptypes.BlockSeal
	containsErr  error
}

func (m *algoMockLogsDB) LatestSealedBlock() (eth.BlockID, bool) { return eth.BlockID{}, false }
func (m *algoMockLogsDB) FirstSealedBlock() (suptypes.BlockSeal, error) {
	if m.firstSealedBlockErr != nil {
		return suptypes.BlockSeal{}, m.firstSealedBlockErr
	}
	return m.firstSealedBlock, nil
}
func (m *algoMockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	return suptypes.BlockSeal{}, nil
}
func (m *algoMockLogsDB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	if m.openBlockErr != nil {
		return eth.BlockRef{}, 0, nil, m.openBlockErr
	}
	return m.openBlockRef, m.openBlockLogCnt, m.openBlockExecMsg, nil
}
func (m *algoMockLogsDB) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if m.containsErr != nil {
		return suptypes.BlockSeal{}, m.containsErr
	}
	return m.containsSeal, nil
}
func (m *algoMockLogsDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	return nil
}
func (m *algoMockLogsDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	return nil
}
func (m *algoMockLogsDB) Rewind(inv reads.Invalidator, newHead eth.BlockID) error { return nil }
func (m *algoMockLogsDB) Clear(inv reads.Invalidator) error                       { return nil }
func (m *algoMockLogsDB) Close() error                                            { return nil }

var _ LogsDB = (*algoMockLogsDB)(nil)

// testBlockInfo implements eth.BlockInfo for testing
type testBlockInfo struct {
	hash       common.Hash
	parentHash common.Hash
	number     uint64
	timestamp  uint64
}

func (m *testBlockInfo) Hash() common.Hash                                    { return m.hash }
func (m *testBlockInfo) ParentHash() common.Hash                              { return m.parentHash }
func (m *testBlockInfo) Coinbase() common.Address                             { return common.Address{} }
func (m *testBlockInfo) Root() common.Hash                                    { return common.Hash{} }
func (m *testBlockInfo) NumberU64() uint64                                    { return m.number }
func (m *testBlockInfo) Time() uint64                                         { return m.timestamp }
func (m *testBlockInfo) MixDigest() common.Hash                               { return common.Hash{} }
func (m *testBlockInfo) BaseFee() *big.Int                                    { return big.NewInt(1) }
func (m *testBlockInfo) BlobBaseFee(chainConfig *params.ChainConfig) *big.Int { return big.NewInt(1) }
func (m *testBlockInfo) ExcessBlobGas() *uint64                               { return nil }
func (m *testBlockInfo) ReceiptHash() common.Hash                             { return common.Hash{} }
func (m *testBlockInfo) GasUsed() uint64                                      { return 0 }
func (m *testBlockInfo) GasLimit() uint64                                     { return 30000000 }
func (m *testBlockInfo) BlobGasUsed() *uint64                                 { return nil }
func (m *testBlockInfo) ParentBeaconRoot() *common.Hash                       { return nil }
func (m *testBlockInfo) WithdrawalsRoot() *common.Hash                        { return nil }
func (m *testBlockInfo) HeaderRLP() ([]byte, error)                           { return nil, nil }
func (m *testBlockInfo) Header() *types.Header                                { return nil }
func (m *testBlockInfo) ID() eth.BlockID                                      { return eth.BlockID{Hash: m.hash, Number: m.number} }

var _ eth.BlockInfo = (*testBlockInfo)(nil)

// =============================================================================
// Mock Chain Container for Algo Tests
// =============================================================================

// algoMockChain is a simplified mock chain container for algo tests
type algoMockChain struct {
	id              eth.ChainID
	optimisticL2    eth.BlockID
	optimisticL1    eth.BlockID
	optimisticAtErr error
}

func (m *algoMockChain) ID() eth.ChainID                                  { return m.id }
func (m *algoMockChain) Start(ctx context.Context) error                  { return nil }
func (m *algoMockChain) Stop(ctx context.Context) error                   { return nil }
func (m *algoMockChain) Pause(ctx context.Context) error                  { return nil }
func (m *algoMockChain) Resume(ctx context.Context) error                 { return nil }
func (m *algoMockChain) PauseAndStopVN(ctx context.Context) error         { return nil }
func (m *algoMockChain) RegisterVerifier(v activity.VerificationActivity) {}
func (m *algoMockChain) VerifierCurrentL1s() []eth.BlockID                { return nil }
func (m *algoMockChain) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *algoMockChain) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *algoMockChain) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *algoMockChain) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.optimisticAtErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticAtErr
	}
	return m.optimisticL2, m.optimisticL1, nil
}
func (m *algoMockChain) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *algoMockChain) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *algoMockChain) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return nil, types.Receipts{}, nil
}
func (m *algoMockChain) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{}, nil
}
func (m *algoMockChain) RewindEngine(ctx context.Context, timestamp uint64, invalidatedBlock eth.BlockRef) error {
	return nil
}
func (m *algoMockChain) BlockTime() uint64 { return 1 }
func (m *algoMockChain) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64) (bool, error) {
	return false, nil
}
func (m *algoMockChain) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	return nil, nil
}
func (m *algoMockChain) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}
func (m *algoMockChain) SetResetCallback(cb cc.ResetCallback) {}

var _ cc.ChainContainer = (*algoMockChain)(nil)
