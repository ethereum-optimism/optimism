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
func (m *loadLogsTestChainContainer) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
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

func TestVerifyPreviousTimestampSealed_NonActivation_WrongTimestamp_Error(t *testing.T) {
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
			Timestamp: 999, // Wrong! Expected 1000 for ts=1001
		},
	}

	// For non-activation timestamp with wrong previous timestamp, should return error
	hash, err := interop.verifyPreviousTimestampSealed(chainID, db, 1001)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPreviousTimestampNotSealed)
	require.Nil(t, hash)
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
func (m *statefulMockChainContainer) BlockAtTimestamp(ctx context.Context, ts uint64, label eth.BlockLabel) (eth.L2BlockRef, error) {
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
