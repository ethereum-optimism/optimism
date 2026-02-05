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
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Mock LogsDB for unit testing
// =============================================================================

type mockLogsDB struct {
	latestBlock   eth.BlockID
	hasBlocks     bool
	seal          suptypes.BlockSeal
	findSealErr   error
	addLogErr     error
	sealBlockErr  error
	addLogCalls   int
	sealBlockCall *sealBlockCall

	// OpenBlock mock fields
	openBlockRef     eth.BlockRef
	openBlockLogCnt  uint32
	openBlockExecMsg map[uint32]*suptypes.ExecutingMessage
	openBlockErr     error

	// Contains mock fields
	containsSeal suptypes.BlockSeal
	containsErr  error
}

type sealBlockCall struct {
	parentHash common.Hash
	block      eth.BlockID
	timestamp  uint64
}

func (m *mockLogsDB) LatestSealedBlock() (eth.BlockID, bool) {
	return m.latestBlock, m.hasBlocks
}

func (m *mockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	if m.findSealErr != nil {
		return suptypes.BlockSeal{}, m.findSealErr
	}
	return m.seal, nil
}

func (m *mockLogsDB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	if m.openBlockErr != nil {
		return eth.BlockRef{}, 0, nil, m.openBlockErr
	}
	return m.openBlockRef, m.openBlockLogCnt, m.openBlockExecMsg, nil
}

func (m *mockLogsDB) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if m.containsErr != nil {
		return suptypes.BlockSeal{}, m.containsErr
	}
	return m.containsSeal, nil
}

func (m *mockLogsDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	m.addLogCalls++
	return m.addLogErr
}

func (m *mockLogsDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.sealBlockCall = &sealBlockCall{
		parentHash: parentHash,
		block:      block,
		timestamp:  timestamp,
	}
	return m.sealBlockErr
}

func (m *mockLogsDB) Close() error {
	return nil
}

var _ LogsDB = (*mockLogsDB)(nil)

// =============================================================================
// Mock BlockInfo for unit testing
// =============================================================================

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
// Mock ChainContainer for loadLogs tests
// =============================================================================

type loadLogsTestChainContainer struct {
	id                  eth.ChainID
	blockAtTimestampRef eth.L2BlockRef
	blockAtTimestampErr error
	fetchReceiptsInfo   eth.BlockInfo
	fetchReceipts       types.Receipts
	fetchReceiptsErr    error
}

func (m *loadLogsTestChainContainer) ID() eth.ChainID                  { return m.id }
func (m *loadLogsTestChainContainer) Start(ctx context.Context) error  { return nil }
func (m *loadLogsTestChainContainer) Stop(ctx context.Context) error   { return nil }
func (m *loadLogsTestChainContainer) Pause(ctx context.Context) error  { return nil }
func (m *loadLogsTestChainContainer) Resume(ctx context.Context) error { return nil }
func (m *loadLogsTestChainContainer) RegisterVerifier(v activity.VerificationActivity) {
}
func (m *loadLogsTestChainContainer) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	if m.blockAtTimestampErr != nil {
		return eth.L2BlockRef{}, m.blockAtTimestampErr
	}
	return m.blockAtTimestampRef, nil
}
func (m *loadLogsTestChainContainer) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *loadLogsTestChainContainer) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *loadLogsTestChainContainer) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *loadLogsTestChainContainer) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *loadLogsTestChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *loadLogsTestChainContainer) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	if m.fetchReceiptsErr != nil {
		return nil, nil, m.fetchReceiptsErr
	}
	return m.fetchReceiptsInfo, m.fetchReceipts, nil
}
func (m *loadLogsTestChainContainer) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{}, nil
}

var _ cc.ChainContainer = (*loadLogsTestChainContainer)(nil)

// =============================================================================
// LogsDB Tests
// =============================================================================

func TestOpenLogsDB_NewAndClose(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	// Open a new logs DB
	db, err := openLogsDB(gethlog.New(), chainID, dataDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close it
	err = db.Close()
	require.NoError(t, err)
}

func TestOpenLogsDB_SealBlock(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	db, err := openLogsDB(gethlog.New(), chainID, dataDir)
	require.NoError(t, err)
	defer db.Close()

	// Seal a block
	parentHash := common.Hash{0x01}
	block := eth.BlockID{Hash: common.Hash{0x02}, Number: 100}
	timestamp := uint64(1000)

	err = db.SealBlock(parentHash, block, timestamp)
	require.NoError(t, err)

	// Verify the block was sealed
	latestBlock, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latestBlock.Number)
	require.Equal(t, block.Hash, latestBlock.Hash)
}

func TestOpenLogsDB_Persistence(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	// Create and populate a logsDB
	{
		db, err := openLogsDB(gethlog.New(), chainID, dataDir)
		require.NoError(t, err)

		// Seal parent block
		parentBlock := eth.BlockID{Hash: common.Hash{0x01}, Number: 99}
		err = db.SealBlock(common.Hash{}, parentBlock, 998)
		require.NoError(t, err)

		// Add a log
		logHash := common.Hash{0x02}
		err = db.AddLog(logHash, parentBlock, 0, nil)
		require.NoError(t, err)

		// Seal block 100
		block100 := eth.BlockID{Hash: common.Hash{0x03}, Number: 100}
		err = db.SealBlock(parentBlock.Hash, block100, 1000)
		require.NoError(t, err)

		err = db.Close()
		require.NoError(t, err)
	}

	// Reopen and verify persistence
	{
		db, err := openLogsDB(gethlog.New(), chainID, dataDir)
		require.NoError(t, err)
		defer db.Close()

		// Verify the block is still there
		latestBlock, ok := db.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, uint64(100), latestBlock.Number)
		require.Equal(t, common.Hash{0x03}, latestBlock.Hash)
	}
}

func TestOpenLogsDB_MultipleChains(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chainID1 := eth.ChainIDFromUInt64(10)
	chainID2 := eth.ChainIDFromUInt64(8453)

	// Create logsDBs for two chains
	db1, err := openLogsDB(gethlog.New(), chainID1, dataDir)
	require.NoError(t, err)
	defer db1.Close()

	db2, err := openLogsDB(gethlog.New(), chainID2, dataDir)
	require.NoError(t, err)
	defer db2.Close()

	// Seal blocks on chain 1
	parentBlock1 := eth.BlockID{Hash: common.Hash{0x01}, Number: 99}
	err = db1.SealBlock(common.Hash{}, parentBlock1, 998)
	require.NoError(t, err)

	block100_1 := eth.BlockID{Hash: common.Hash{0x02}, Number: 100}
	err = db1.SealBlock(parentBlock1.Hash, block100_1, 1000)
	require.NoError(t, err)

	// Seal blocks on chain 2
	parentBlock2 := eth.BlockID{Hash: common.Hash{0x11}, Number: 199}
	err = db2.SealBlock(common.Hash{}, parentBlock2, 1998)
	require.NoError(t, err)

	block200_2 := eth.BlockID{Hash: common.Hash{0x12}, Number: 200}
	err = db2.SealBlock(parentBlock2.Hash, block200_2, 2000)
	require.NoError(t, err)

	// Verify chain 1
	latestBlock1, ok := db1.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(100), latestBlock1.Number)

	// Verify chain 2
	latestBlock2, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(200), latestBlock2.Number)
}

// =============================================================================
// verifyPreviousTimestampSealed Tests
// =============================================================================

func TestVerifyPreviousTimestampSealed_ActivationTimestamp_EmptyDB(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	db := &mockLogsDB{hasBlocks: false}

	// For activation timestamp with empty DB, should return nil hash and nil error
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1000)
	require.NoError(t, err)
	require.Nil(t, hash)
}

func TestVerifyPreviousTimestampSealed_ActivationTimestamp_NonEmptyDB_Error(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	db := &mockLogsDB{
		hasBlocks:   true,
		latestBlock: eth.BlockID{Hash: common.Hash{0x01}, Number: 100},
	}

	// For activation timestamp with non-empty DB, should return error
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1000)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPreviousTimestampNotSealed)
	require.Nil(t, hash)
}

func TestVerifyPreviousTimestampSealed_NonActivation_EmptyDB_Error(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	db := &mockLogsDB{hasBlocks: false}

	// For non-activation timestamp with empty DB, should return error
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPreviousTimestampNotSealed)
	require.Nil(t, hash)
}

func TestVerifyPreviousTimestampSealed_NonActivation_TimestampNotBefore_Error(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	db := &mockLogsDB{
		hasBlocks:   true,
		latestBlock: eth.BlockID{Hash: common.Hash{0x01}, Number: 100},
		seal: suptypes.BlockSeal{
			Hash:      common.Hash{0x01},
			Number:    100,
			Timestamp: 1001, // Error: not before ts=1001
		},
	}

	// For non-activation timestamp where seal.Timestamp >= ts, should return error
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPreviousTimestampNotSealed)
	require.Nil(t, hash)
}

func TestVerifyPreviousTimestampSealed_NonActivation_OlderTimestamp_Success(t *testing.T) {
	t.Parallel()

	// Test that a block with timestamp < ts (but not exactly ts-1) is valid.
	// This handles the case where blocks span multiple seconds (block time > 1).
	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	expectedHash := common.Hash{0x01}
	db := &mockLogsDB{
		hasBlocks:   true,
		latestBlock: eth.BlockID{Hash: expectedHash, Number: 100},
		seal: suptypes.BlockSeal{
			Hash:      expectedHash,
			Number:    100,
			Timestamp: 998, // Older than ts-1, but still valid
		},
	}

	// For ts=1001, a block at timestamp 998 is valid (998 < 1001)
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.NoError(t, err)
	require.NotNil(t, hash)
	require.Equal(t, expectedHash, *hash)
}

func TestVerifyPreviousTimestampSealed_NonActivation_CorrectTimestamp_ReturnsHash(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	expectedHash := common.Hash{0x01}
	db := &mockLogsDB{
		hasBlocks:   true,
		latestBlock: eth.BlockID{Hash: expectedHash, Number: 100},
		seal: suptypes.BlockSeal{
			Hash:      expectedHash,
			Number:    100,
			Timestamp: 1000, // Correct! Expected 1000 for ts=1001
		},
	}

	// For non-activation timestamp with correct previous timestamp, should return hash
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.NoError(t, err)
	require.NotNil(t, hash)
	require.Equal(t, expectedHash, *hash)
}

func TestVerifyPreviousTimestampSealed_FindSealedBlock_Error(t *testing.T) {
	t.Parallel()

	interop := &Interop{
		log:                 gethlog.New(),
		activationTimestamp: 1000,
	}
	chainID := eth.ChainIDFromUInt64(10)
	db := &mockLogsDB{
		hasBlocks:   true,
		latestBlock: eth.BlockID{Hash: common.Hash{0x01}, Number: 100},
		findSealErr: errors.New("database error"),
	}

	// Should return error when FindSealedBlock fails
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database error")
	require.Nil(t, hash)
}

// =============================================================================
// processBlockLogs Tests
// =============================================================================

func TestProcessBlockLogs_EmptyReceipts(t *testing.T) {
	t.Parallel()

	interop := &Interop{log: gethlog.New()}
	db := &mockLogsDB{}
	blockInfo := &testBlockInfo{
		hash:       common.Hash{0x02},
		parentHash: common.Hash{0x01},
		number:     100,
		timestamp:  1000,
	}

	err := interop.processBlockLogs(db, blockInfo, types.Receipts{})
	require.NoError(t, err)

	// Should seal the block even with no logs
	require.NotNil(t, db.sealBlockCall)
	require.Equal(t, common.Hash{0x01}, db.sealBlockCall.parentHash)
	require.Equal(t, uint64(100), db.sealBlockCall.block.Number)
	require.Equal(t, uint64(1000), db.sealBlockCall.timestamp)
	require.Equal(t, 0, db.addLogCalls)
}

func TestProcessBlockLogs_WithLogs(t *testing.T) {
	t.Parallel()

	interop := &Interop{log: gethlog.New()}
	db := &mockLogsDB{}
	blockInfo := &testBlockInfo{
		hash:       common.Hash{0x02},
		parentHash: common.Hash{0x01},
		number:     100,
		timestamp:  1000,
	}

	// Create receipts with multiple logs
	receipts := types.Receipts{
		&types.Receipt{
			Logs: []*types.Log{
				{Address: common.Address{0xAA}, Data: []byte{0x01}},
				{Address: common.Address{0xBB}, Data: []byte{0x02}},
			},
		},
		&types.Receipt{
			Logs: []*types.Log{
				{Address: common.Address{0xCC}, Data: []byte{0x03}},
			},
		},
	}

	err := interop.processBlockLogs(db, blockInfo, receipts)
	require.NoError(t, err)

	// Should have called AddLog 3 times (2 logs in first receipt + 1 in second)
	require.Equal(t, 3, db.addLogCalls)

	// Should seal the block
	require.NotNil(t, db.sealBlockCall)
	require.Equal(t, uint64(100), db.sealBlockCall.block.Number)
}

func TestProcessBlockLogs_AddLogError(t *testing.T) {
	t.Parallel()

	interop := &Interop{log: gethlog.New()}
	db := &mockLogsDB{addLogErr: errors.New("add log failed")}
	blockInfo := &testBlockInfo{
		hash:       common.Hash{0x02},
		parentHash: common.Hash{0x01},
		number:     100,
		timestamp:  1000,
	}

	receipts := types.Receipts{
		&types.Receipt{
			Logs: []*types.Log{
				{Address: common.Address{0xAA}, Data: []byte{0x01}},
			},
		},
	}

	err := interop.processBlockLogs(db, blockInfo, receipts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "add log failed")
}

func TestProcessBlockLogs_SealBlockError(t *testing.T) {
	t.Parallel()

	interop := &Interop{log: gethlog.New()}
	db := &mockLogsDB{sealBlockErr: errors.New("seal failed")}
	blockInfo := &testBlockInfo{
		hash:       common.Hash{0x02},
		parentHash: common.Hash{0x01},
		number:     100,
		timestamp:  1000,
	}

	err := interop.processBlockLogs(db, blockInfo, types.Receipts{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "seal failed")
}

func TestProcessBlockLogs_GenesisBlock(t *testing.T) {
	t.Parallel()

	interop := &Interop{log: gethlog.New()}
	db := &mockLogsDB{}
	blockInfo := &testBlockInfo{
		hash:       common.Hash{0x01},
		parentHash: common.Hash{}, // Genesis has no parent
		number:     0,
		timestamp:  1000,
	}

	err := interop.processBlockLogs(db, blockInfo, types.Receipts{})
	require.NoError(t, err)

	// Should seal block 0 with empty parent block
	require.NotNil(t, db.sealBlockCall)
	require.Equal(t, uint64(0), db.sealBlockCall.block.Number)
}

// =============================================================================
// loadLogs Tests
// =============================================================================

func TestLoadLogs_ParentHashMismatch_Error(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()

	chainID := eth.ChainIDFromUInt64(10)

	// Use a stateful mock that changes behavior based on timestamp
	firstBlockHash := common.Hash{0x01}
	wrongParentHash := common.Hash{0xFF} // This won't match the logsDB

	callCount := 0
	mockChain := &statefulMockChainContainer{
		id: chainID,
		blockAtTimestampFn: func(ts uint64) (eth.L2BlockRef, error) {
			if ts == 1000 {
				return eth.L2BlockRef{
					Hash:   firstBlockHash,
					Number: 100,
					Time:   1000,
				}, nil
			}
			return eth.L2BlockRef{
				Hash:   common.Hash{0x02},
				Number: 101,
				Time:   1001,
			}, nil
		},
		fetchReceiptsFn: func(blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
			callCount++
			if callCount == 1 {
				// First call (timestamp 1000) - return block with correct data
				return &testBlockInfo{
					hash:       firstBlockHash,
					parentHash: common.Hash{}, // Genesis parent
					number:     100,
					timestamp:  1000,
				}, types.Receipts{}, nil
			}
			// Second call (timestamp 1001) - return block with WRONG parent hash
			return &testBlockInfo{
				hash:       common.Hash{0x02},
				parentHash: wrongParentHash, // Wrong parent! Should be firstBlockHash
				number:     101,
				timestamp:  1001,
			}, types.Receipts{}, nil
		},
	}

	chains := map[eth.ChainID]cc.ChainContainer{chainID: mockChain}
	interop := New(gethlog.New(), 1000, chains, dataDir)
	require.NotNil(t, interop)
	interop.ctx = context.Background()
	defer interop.Stop(context.Background())

	// First, load logs for activation timestamp (1000) to set up the logsDB
	err := interop.loadLogs(1000)
	require.NoError(t, err)

	// Now try to load logs for 1001 - should fail because parent hash doesn't match
	// The logsDB has the block from timestamp 1000 with hash firstBlockHash,
	// but the new block claims wrongParentHash as its parent
	err = interop.loadLogs(1001)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrParentHashMismatch)
}

// statefulMockChainContainer allows dynamic behavior based on test state
type statefulMockChainContainer struct {
	id                 eth.ChainID
	blockAtTimestampFn func(ts uint64) (eth.L2BlockRef, error)
	fetchReceiptsFn    func(blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error)
}

func (m *statefulMockChainContainer) ID() eth.ChainID                  { return m.id }
func (m *statefulMockChainContainer) Start(ctx context.Context) error  { return nil }
func (m *statefulMockChainContainer) Stop(ctx context.Context) error   { return nil }
func (m *statefulMockChainContainer) Pause(ctx context.Context) error  { return nil }
func (m *statefulMockChainContainer) Resume(ctx context.Context) error { return nil }
func (m *statefulMockChainContainer) RegisterVerifier(v activity.VerificationActivity) {
}
func (m *statefulMockChainContainer) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return m.blockAtTimestampFn(ts)
}
func (m *statefulMockChainContainer) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *statefulMockChainContainer) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *statefulMockChainContainer) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *statefulMockChainContainer) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *statefulMockChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *statefulMockChainContainer) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return m.fetchReceiptsFn(blockID)
}
func (m *statefulMockChainContainer) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{}, nil
}

var _ cc.ChainContainer = (*statefulMockChainContainer)(nil)

// =============================================================================
// verifyInteropMessages Tests
// =============================================================================

func TestVerifyInteropMessages_NoExecMessages_AllValid(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(10)
	blockHash := common.HexToHash("0x123")
	expectedBlock := eth.BlockID{Number: 100, Hash: blockHash}

	// Mock logsDB returns block with no executing messages
	mockDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   blockHash,
			Number: 100,
			Time:   1000,
		},
		openBlockExecMsg: nil, // No executing messages
	}

	interop := &Interop{
		log:     gethlog.New(),
		logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
	}

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		chainID: expectedBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.True(t, result.IsValid())
	require.Empty(t, result.InvalidHeads)
	require.Equal(t, expectedBlock, result.L2Heads[chainID])
}

func TestVerifyInteropMessages_BlockHashMismatch_Invalid(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(10)
	expectedBlock := eth.BlockID{Number: 100, Hash: common.HexToHash("0xExpected")}
	actualBlockHash := common.HexToHash("0xActual") // Different from expected

	// Mock logsDB returns block with different hash than expected
	mockDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   actualBlockHash, // Different from expectedBlock.Hash
			Number: 100,
			Time:   1000,
		},
	}

	interop := &Interop{
		log:     gethlog.New(),
		logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
	}

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		chainID: expectedBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.False(t, result.IsValid())
	require.Contains(t, result.InvalidHeads, chainID)
	require.Equal(t, expectedBlock, result.InvalidHeads[chainID])
}

func TestVerifyInteropMessages_ValidExecMessage_Success(t *testing.T) {
	t.Parallel()

	sourceChainID := eth.ChainIDFromUInt64(10)
	destChainID := eth.ChainIDFromUInt64(8453)

	sourceBlockHash := common.HexToHash("0xSource")
	destBlockHash := common.HexToHash("0xDest")

	sourceBlock := eth.BlockID{Number: 50, Hash: sourceBlockHash}
	destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

	// Executing message references initiating message on source chain
	execMsg := &suptypes.ExecutingMessage{
		ChainID:   sourceChainID,
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: 500, // Source timestamp < dest timestamp (1000)
		Checksum:  suptypes.MessageChecksum{0x01},
	}

	// Source chain logsDB - has the initiating message
	sourceDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   sourceBlockHash,
			Number: 50,
			Time:   500,
		},
		containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 500}, // Found!
	}

	// Dest chain logsDB - has the executing message
	destDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   destBlockHash,
			Number: 100,
			Time:   1000,
		},
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

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		sourceChainID: sourceBlock,
		destChainID:   destBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.True(t, result.IsValid())
	require.Empty(t, result.InvalidHeads)
}

func TestVerifyInteropMessages_InitiatingMessageNotFound_Invalid(t *testing.T) {
	t.Parallel()

	sourceChainID := eth.ChainIDFromUInt64(10)
	destChainID := eth.ChainIDFromUInt64(8453)

	destBlockHash := common.HexToHash("0xDest")
	destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

	// Executing message references non-existent initiating message
	execMsg := &suptypes.ExecutingMessage{
		ChainID:   sourceChainID,
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: 500,
		Checksum:  suptypes.MessageChecksum{0x01},
	}

	// Source chain logsDB - does NOT have the initiating message
	sourceDB := &mockLogsDB{
		containsErr: suptypes.ErrConflict, // Message not found
	}

	// Dest chain logsDB - has the executing message
	destDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   destBlockHash,
			Number: 100,
			Time:   1000,
		},
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

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		destChainID: destBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.False(t, result.IsValid())
	require.Contains(t, result.InvalidHeads, destChainID)
}

func TestVerifyInteropMessages_TimestampViolation_Invalid(t *testing.T) {
	t.Parallel()

	sourceChainID := eth.ChainIDFromUInt64(10)
	destChainID := eth.ChainIDFromUInt64(8453)

	destBlockHash := common.HexToHash("0xDest")
	destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

	// Executing message references initiating message with timestamp >= executing timestamp
	execMsg := &suptypes.ExecutingMessage{
		ChainID:   sourceChainID,
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: 1000, // Same as dest block timestamp - INVALID!
		Checksum:  suptypes.MessageChecksum{0x01},
	}

	// Source chain logsDB - has the initiating message
	sourceDB := &mockLogsDB{
		containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 1000}, // Found
	}

	// Dest chain logsDB - has the executing message
	destDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   destBlockHash,
			Number: 100,
			Time:   1000, // Same timestamp as initiating message - INVALID
		},
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

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		destChainID: destBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.False(t, result.IsValid())
	require.Contains(t, result.InvalidHeads, destChainID)
}

func TestVerifyInteropMessages_SourceChainNotFound_Invalid(t *testing.T) {
	t.Parallel()

	unknownSourceChain := eth.ChainIDFromUInt64(9999) // Not in logsDBs
	destChainID := eth.ChainIDFromUInt64(8453)

	destBlockHash := common.HexToHash("0xDest")
	destBlock := eth.BlockID{Number: 100, Hash: destBlockHash}

	// Executing message references unknown source chain
	execMsg := &suptypes.ExecutingMessage{
		ChainID:   unknownSourceChain, // This chain is not registered
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: 500,
		Checksum:  suptypes.MessageChecksum{0x01},
	}

	// Dest chain logsDB - has the executing message
	destDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   destBlockHash,
			Number: 100,
			Time:   1000,
		},
		openBlockExecMsg: map[uint32]*suptypes.ExecutingMessage{
			0: execMsg,
		},
	}

	interop := &Interop{
		log: gethlog.New(),
		logsDBs: map[eth.ChainID]LogsDB{
			destChainID: destDB,
			// Note: unknownSourceChain is NOT in logsDBs
		},
	}

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		destChainID: destBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.False(t, result.IsValid())
	require.Contains(t, result.InvalidHeads, destChainID)
}

func TestVerifyInteropMessages_MultipleChains_OneInvalid(t *testing.T) {
	t.Parallel()

	sourceChainID := eth.ChainIDFromUInt64(10)
	validChainID := eth.ChainIDFromUInt64(8453)
	invalidChainID := eth.ChainIDFromUInt64(420)

	validBlockHash := common.HexToHash("0xValid")
	invalidBlockHash := common.HexToHash("0xInvalid")

	validBlock := eth.BlockID{Number: 100, Hash: validBlockHash}
	invalidBlock := eth.BlockID{Number: 200, Hash: invalidBlockHash}

	// Invalid chain has an executing message with bad timestamp
	badExecMsg := &suptypes.ExecutingMessage{
		ChainID:   sourceChainID,
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: 1000, // Same as block timestamp - INVALID
		Checksum:  suptypes.MessageChecksum{0x01},
	}

	sourceDB := &mockLogsDB{
		containsSeal: suptypes.BlockSeal{Number: 50, Timestamp: 1000},
	}

	validDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   validBlockHash,
			Number: 100,
			Time:   1000,
		},
		openBlockExecMsg: nil, // No executing messages - valid
	}

	invalidDB := &mockLogsDB{
		openBlockRef: eth.BlockRef{
			Hash:   invalidBlockHash,
			Number: 200,
			Time:   1000,
		},
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

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		validChainID:   validBlock,
		invalidChainID: invalidBlock,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.NoError(t, err)
	require.False(t, result.IsValid())
	// Both chains should be in L2Heads
	require.Contains(t, result.L2Heads, validChainID)
	require.Contains(t, result.L2Heads, invalidChainID)
	// Only the invalid chain should be in InvalidHeads
	require.NotContains(t, result.InvalidHeads, validChainID)
	require.Contains(t, result.InvalidHeads, invalidChainID)
}

func TestVerifyInteropMessages_OpenBlockError(t *testing.T) {
	t.Parallel()

	chainID := eth.ChainIDFromUInt64(10)
	block := eth.BlockID{Number: 100, Hash: common.HexToHash("0x123")}

	mockDB := &mockLogsDB{
		openBlockErr: errors.New("database error"),
	}

	interop := &Interop{
		log:     gethlog.New(),
		logsDBs: map[eth.ChainID]LogsDB{chainID: mockDB},
	}

	blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
		chainID: block,
	}

	result, err := interop.verifyInteropMessages(1000, blocksAtTimestamp)

	require.Error(t, err)
	require.Contains(t, err.Error(), "database error")
	require.True(t, result.IsEmpty())
}
