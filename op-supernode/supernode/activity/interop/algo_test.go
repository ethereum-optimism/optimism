package interop

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// TestVerifyInteropMessages - Table-Driven Tests
// =============================================================================

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

				mockDB := &algoMockLogsDB{
					openBlockRef:     eth.BlockRef{Hash: blockHash, Number: 100, Time: 1000},
					openBlockExecMsg: nil,
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
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
			name: "InvalidBlocks/TimestampViolation",
			setup: func() (*Interop, uint64, map[eth.ChainID]eth.BlockID) {
				sourceChainID := eth.ChainIDFromUInt64(10)
				destChainID := eth.ChainIDFromUInt64(8453)

				destBlockHash := common.HexToHash("0xDest")
				destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

				execMsg := &suptypes.ExecutingMessage{
					ChainID:   sourceChainID,
					BlockNum:  50,
					LogIdx:    0,
					Timestamp: 1001, // Future timestamp - INVALID!
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

				mockDB := &algoMockLogsDB{
					openBlockErr: errors.New("database error"),
				}

				interop := &Interop{
					log:     gethlog.New(),
					logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
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
