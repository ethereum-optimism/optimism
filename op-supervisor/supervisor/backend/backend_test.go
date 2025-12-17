package backend

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const testChainIDOffset = 900

func fullConfigSet(t *testing.T, size int) depset.FullConfigSetMerged {
	staticDepSet := make(map[eth.ChainID]*depset.StaticConfigDependency, size)
	staticRollupCfgSet := make(map[eth.ChainID]*depset.StaticRollupConfig, size)
	zero := uint64(0)
	for i := 0; i < size; i++ {
		chainID := eth.ChainIDFromUInt64(testChainIDOffset + uint64(i))
		staticDepSet[chainID] = &depset.StaticConfigDependency{}
		staticRollupCfgSet[chainID] = &depset.StaticRollupConfig{
			InteropTime: &zero,
			BlockTime:   2,
		}
	}
	depSet, err := depset.NewStaticConfigDependencySet(staticDepSet)
	require.NoError(t, err)
	rollupCfgSet := depset.NewStaticRollupConfigSet(staticRollupCfgSet)
	fullCfgSet, err := depset.NewFullConfigSetMerged(rollupCfgSet, depSet)
	require.NoError(t, err)
	return fullCfgSet
}

func TestBackendLifetime_InteropAtGenesis(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	chainA := eth.ChainIDFromUInt64(testChainIDOffset)
	fullCfgSet := fullConfigSet(t, 2)
	rollupCfgSet := fullCfgSet.RollupConfigSet.(depset.StaticRollupConfigSet)

	anchor := eth.BlockRef{
		Hash:       common.Hash{0xff},
		Number:     0,
		ParentHash: common.Hash{}, // genesis has no parent hash
		Time:       10000,
	}

	rollupCfgSet[chainA].Genesis = depset.Genesis{
		L2: types.BlockSealFromRef(anchor),
	}

	cfg := &config.Config{
		Version:               "test",
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               dataDir,
	}

	ex := event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
	require.NoError(t, err)
	t.Log("initialized!")

	l1Src := &testutils.MockL1Source{}
	src := &MockProcessorSource{}

	blockX := eth.L2BlockRef{
		Hash:       common.Hash{0xaa},
		Number:     anchor.Number + 1,
		ParentHash: anchor.Hash,
		Time:       anchor.Time + 2,
	}
	blockY := eth.L2BlockRef{
		Hash:       common.Hash{0xbb},
		Number:     blockX.Number + 1,
		ParentHash: blockX.Hash,
		Time:       blockX.Time + 2,
	}

	b.AttachL1Source(l1Src)
	require.NoError(t, b.AttachProcessorSource(chainA, src))

	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "log.db"), "must have logs DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "log.db"), "must have logs DB 901")
	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "local_safe.db"), "must have local safe DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "local_safe.db"), "must have local safe DB 901")
	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "cross_safe.db"), "must have cross safe DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "cross_safe.db"), "must have cross safe DB 901")

	err = b.Start(context.Background())
	require.NoError(t, err)
	t.Log("started!")

	_, err = b.LocalUnsafe(context.Background(), chainA)
	require.ErrorIs(t, err, types.ErrFuture, "no data yet, need local-unsafe")

	require.NoError(t, ex.Drain())
	// The database is initialized from the genesis interop block at startup.
	xunsafe, err := b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, anchor.ID(), xunsafe)
	xsafe, err := b.CrossSafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, anchor.ID(), xsafe.Derived)

	// Receive unsafe block Y from node

	src.ExpectL2BlockRefByNumber(1, blockX, nil)
	src.ExpectFetchReceipts(blockX.Hash, nil, nil)
	src.ExpectL2BlockRefByNumber(2, blockY, nil)
	src.ExpectFetchReceipts(blockY.Hash, nil, nil)
	b.emitter.Emit(context.Background(), superevents.LocalUnsafeReceivedEvent{
		ChainID:        chainA,
		NewLocalUnsafe: blockY.BlockRef(),
	})
	require.NoError(t, ex.Drain())
	src.AssertExpectations(t)
	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockY.ID(), xunsafe)
	// cross-safe still at anchor
	xsafe, err = b.CrossSafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, anchor.ID(), xsafe.Derived)

	// Revert cross-unafe back to block X
	err = b.chainDBs.UpdateCrossUnsafe(chainA, types.BlockSealFromRef(blockX.BlockRef()))
	require.NoError(t, err)

	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockX.ID(), xunsafe)

	// Receive derived block X from node

	b.emitter.Emit(context.Background(), superevents.LocalDerivedEvent{
		ChainID: chainA,
		Derived: types.DerivedBlockRefPair{
			Derived: blockX.BlockRef(),
		},
	})
	require.NoError(t, ex.Drain())
	src.AssertExpectations(t)
	// cross-unsafe still at block X
	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockY.ID(), xunsafe)
	// cross-safe now at block X
	xsafe, err = b.CrossSafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockX.ID(), xsafe.Derived)

	err = b.Stop(context.Background())
	require.NoError(t, err)
	t.Log("stopped!")
}

func TestBackendLifetime_InteropPostGenesis(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	chainA := eth.ChainIDFromUInt64(testChainIDOffset)
	fullCfgSet := fullConfigSet(t, 2)
	rollupCfgSet := fullCfgSet.RollupConfigSet.(depset.StaticRollupConfigSet)

	block0 := eth.BlockRef{
		Hash:       common.Hash{0xff},
		Number:     0,
		ParentHash: common.Hash{}, // genesis has no parent hash
		Time:       10000,
	}
	blockX := eth.BlockRef{
		Hash:       common.Hash{0xaa},
		Number:     block0.Number + 1,
		ParentHash: block0.Hash,
		Time:       block0.Time + 2,
	}

	rollupCfgSet[chainA].InteropTime = &blockX.Time
	rollupCfgSet[chainA].Genesis = depset.Genesis{
		L2: types.BlockSealFromRef(block0),
	}

	cfg := &config.Config{
		Version:               "test",
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               dataDir,
	}

	ex := event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
	require.NoError(t, err)
	t.Log("initialized!")

	l1Src := &testutils.MockL1Source{}
	src := &MockProcessorSource{}

	blockY := eth.L2BlockRef{
		Hash:       common.Hash{0xbb},
		Number:     blockX.Number + 1,
		ParentHash: blockX.Hash,
		Time:       blockX.Time + 2,
	}

	b.AttachL1Source(l1Src)
	require.NoError(t, b.AttachProcessorSource(chainA, src))

	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "log.db"), "must have logs DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "log.db"), "must have logs DB 901")
	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "local_safe.db"), "must have local safe DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "local_safe.db"), "must have local safe DB 901")
	require.FileExists(t, filepath.Join(cfg.Datadir, "900", "cross_safe.db"), "must have cross safe DB 900")
	require.FileExists(t, filepath.Join(cfg.Datadir, "901", "cross_safe.db"), "must have cross safe DB 901")

	err = b.Start(context.Background())
	require.NoError(t, err)
	t.Log("started!")

	_, err = b.LocalUnsafe(context.Background(), chainA)
	require.ErrorIs(t, err, types.ErrFuture, "no data yet, need local-unsafe")

	require.NoError(t, ex.Drain())
	// The database is not initialized from non-Interop genesis
	xunsafe, err := b.CrossUnsafe(context.Background(), chainA)
	require.ErrorIs(t, err, types.ErrFuture, "got xunsafe %v", xunsafe)
	xsafe, err := b.CrossSafe(context.Background(), chainA)
	require.ErrorIs(t, err, types.ErrFuture, "got xsafe %v", xsafe)

	// Receive unsafe block X, interop activation block, from node

	// src.ExpectL2BlockRefByNumber(1, blockX, nil)
	// src.ExpectFetchReceipts(blockX.Hash, nil, nil)
	b.emitter.Emit(context.Background(), superevents.LocalUnsafeReceivedEvent{
		ChainID:        chainA,
		NewLocalUnsafe: blockX,
	})
	require.NoError(t, ex.Drain())
	src.AssertExpectations(t)
	unsafe, err := b.LocalUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockX.ID(), unsafe)
	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockX.ID(), xunsafe)
	// cross-safe still undefined
	_, err = b.CrossSafe(context.Background(), chainA)
	require.ErrorIs(t, err, types.ErrFuture, err)

	// Receive unsafe block Y from node

	src.ExpectL2BlockRefByNumber(blockY.Number, blockY, nil)
	src.ExpectFetchReceipts(blockY.Hash, nil, nil)
	b.emitter.Emit(context.Background(), superevents.LocalUnsafeReceivedEvent{
		ChainID:        chainA,
		NewLocalUnsafe: blockY.BlockRef(),
	})
	require.NoError(t, ex.Drain())
	src.AssertExpectations(t)
	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockY.ID(), xunsafe)

	// Receive derived block X from node

	b.emitter.Emit(context.Background(), superevents.LocalDerivedEvent{
		ChainID: chainA,
		Derived: types.DerivedBlockRefPair{
			Derived: blockX,
		},
	})
	require.NoError(t, ex.Drain())
	src.AssertExpectations(t)
	// cross-unsafe still at block Y
	xunsafe, err = b.CrossUnsafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockY.ID(), xunsafe)
	// cross-safe now at block X
	xsafe, err = b.CrossSafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockX.ID(), xsafe.Derived)

	// Receive derived block Y from node

	b.emitter.Emit(context.Background(), superevents.LocalDerivedEvent{
		ChainID: chainA,
		Derived: types.DerivedBlockRefPair{
			Derived: blockY.BlockRef(),
		},
	})
	require.NoError(t, ex.Drain())
	// cross-safe now at block Y
	xsafe, err = b.CrossSafe(context.Background(), chainA)
	require.NoError(t, err)
	require.Equal(t, blockY.ID(), xsafe.Derived)

	err = b.Stop(context.Background())
	require.NoError(t, err)
	t.Log("stopped!")
}

func TestBackendCallsMetrics(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	mockMetrics := &MockMetrics{}
	dataDir := t.TempDir()
	chainA := eth.ChainIDFromUInt64(testChainIDOffset)

	// Set up mock metrics
	mockMetrics.Mock.On("RecordDBEntryCount", chainA, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).Return()
	mockMetrics.Mock.On("RecordCrossUnsafe", chainA, mock.MatchedBy(func(_ types.BlockSeal) bool { return true })).Return()
	mockMetrics.Mock.On("RecordCrossSafe", chainA, mock.MatchedBy(func(_ types.BlockSeal) bool { return true })).Return()
	mockMetrics.Mock.On("RecordLocalSafe", chainA, mock.MatchedBy(func(_ types.BlockSeal) bool { return true })).Return()
	mockMetrics.Mock.On("RecordLocalUnsafe", chainA, mock.MatchedBy(func(_ types.BlockSeal) bool { return true })).Return()

	fullCfgSet := fullConfigSet(t, 1)
	cfg := &config.Config{
		Version:       "test",
		LogConfig:     oplog.CLIConfig{},
		MetricsConfig: opmetrics.CLIConfig{},
		PprofConfig:   oppprof.CLIConfig{},
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               dataDir,
	}

	ex := event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, mockMetrics, cfg, ex)
	require.NoError(t, err)

	// Assert that the metrics are called at initialization
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "log", int64(0))
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "local_derived", int64(0))
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "cross_derived", int64(0))

	// Start the backend
	err = b.Start(context.Background())
	require.NoError(t, err)

	// Create a test block
	block := eth.BlockRef{
		Hash:       common.Hash{0xaa},
		Number:     42,
		ParentHash: common.Hash{0xbb},
		Time:       10000,
	}
	safe := types.DerivedBlockRefPair{
		Source:  block, // dummy value
		Derived: block,
	}
	// update local unsafe/safe, cross unsafe/safe
	b.chainDBs.OnEvent(context.Background(), superevents.SafeActivationBlockEvent{
		Safe:    safe,
		ChainID: chainA,
	})
	// Assert that metrics are called on safety level updates
	mockMetrics.Mock.AssertCalled(t, "RecordLocalUnsafe", chainA, mock.MatchedBy(func(ref types.BlockSeal) bool {
		return ref.Hash == block.Hash && ref.Number == block.Number && ref.Timestamp == block.Time
	}))
	mockMetrics.Mock.AssertCalled(t, "RecordLocalSafe", chainA, mock.MatchedBy(func(ref types.BlockSeal) bool {
		return ref.Hash == block.Hash && ref.Number == block.Number && ref.Timestamp == block.Time
	}))
	mockMetrics.Mock.AssertCalled(t, "RecordCrossUnsafe", chainA, mock.MatchedBy(func(ref types.BlockSeal) bool {
		return ref.Hash == block.Hash && ref.Number == block.Number && ref.Timestamp == block.Time
	}))
	mockMetrics.Mock.AssertCalled(t, "RecordCrossSafe", chainA, mock.MatchedBy(func(ref types.BlockSeal) bool {
		return ref.Hash == block.Hash && ref.Number == block.Number && ref.Timestamp == block.Time
	}))
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "cross_derived", int64(1))
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "local_derived", int64(1))
	// db entry: searchCheckpoint, canonicalHash
	mockMetrics.Mock.AssertCalled(t, "RecordDBEntryCount", chainA, "log", int64(2))

	// Stop the backend
	err = b.Stop(context.Background())
	require.NoError(t, err)
}

type MockMetrics struct {
	mock.Mock
	event.NoopMetrics
	opmetrics.NoopRPCMetrics
}

var _ Metrics = (*MockMetrics)(nil)

func (m *MockMetrics) CacheAdd(chainID eth.ChainID, label string, cacheSize int, evicted bool) {
	m.Mock.Called(chainID, label, cacheSize, evicted)
}

func (m *MockMetrics) CacheGet(chainID eth.ChainID, label string, hit bool) {
	m.Mock.Called(chainID, label, hit)
}

func (m *MockMetrics) RecordCrossUnsafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.Mock.Called(chainID, seal)
}

func (m *MockMetrics) RecordCrossSafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.Mock.Called(chainID, seal)
}

func (m *MockMetrics) RecordLocalSafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.Mock.Called(chainID, seal)
}

func (m *MockMetrics) RecordLocalUnsafe(chainID eth.ChainID, seal types.BlockSeal) {
	m.Mock.Called(chainID, seal)
}

func (m *MockMetrics) RecordDBEntryCount(chainID eth.ChainID, kind string, count int64) {
	m.Mock.Called(chainID, kind, count)
}

func (m *MockMetrics) RecordDBSearchEntriesRead(chainID eth.ChainID, count int64) {
	m.Mock.Called(chainID, count)
}

func (m *MockMetrics) RecordAccessListVerifyFailure(chainID eth.ChainID) {
	m.Mock.Called(chainID)
}

type MockProcessorSource struct {
	mock.Mock
}

var _ processors.Source = (*MockProcessorSource)(nil)

func (m *MockProcessorSource) FetchReceipts(ctx context.Context, blockHash common.Hash) (types2.Receipts, error) {
	out := m.Mock.Called(blockHash)
	return out.Get(0).(types2.Receipts), out.Error(1)
}

func (m *MockProcessorSource) ExpectFetchReceipts(hash common.Hash, receipts types2.Receipts, err error) {
	m.Mock.On("FetchReceipts", hash).Once().Return(receipts, err)
}

func (m *MockProcessorSource) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	out := m.Mock.Called(num)
	return out.Get(0).(eth.L2BlockRef), out.Error(1)
}

func (m *MockProcessorSource) ExpectL2BlockRefByNumber(num uint64, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByNumber", num).Return(ref, err)
}

// fakeSyncSource implements syncnode.SyncSource for testing asyncVerifyAccessWithRPC.
type fakeSyncSource struct {
	chainID eth.ChainID
	seal    types.BlockSeal
	err     error
}

func (f *fakeSyncSource) Contains(_ context.Context, _ types.ContainsQuery) (types.BlockSeal, error) {
	return f.seal, f.err
}

func (f *fakeSyncSource) ChainID(_ context.Context) (eth.ChainID, error) {
	return f.chainID, nil
}

func (f *fakeSyncSource) L2BlockRefByNumber(_ context.Context, _ uint64) (eth.L2BlockRef, error) {
	panic("should not be called")
}

func (f *fakeSyncSource) FetchReceipts(_ context.Context, _ common.Hash) (types2.Receipts, error) {
	panic("should not be called")
}

func (f *fakeSyncSource) OutputV0AtTimestamp(_ context.Context, _ uint64) (*eth.OutputV0, error) {
	panic("should not be called")
}

func (f *fakeSyncSource) PendingOutputV0AtTimestamp(_ context.Context, _ uint64) (*eth.OutputV0, error) {
	panic("should not be called")
}

func (f *fakeSyncSource) L2BlockRefByTimestamp(_ context.Context, _ uint64) (eth.L2BlockRef, error) {
	panic("should not be called")
}

func (f *fakeSyncSource) String() string {
	return "fakeSyncSource"
}

// TestAsyncVerifyAccessWithRPC exercises the asyncVerifyAccessWithRPC method against various RPC error and block match/mismatch scenarios.
// The method is responsible for asynchronously verifying RPC access checks (checksum and block ID matching),
// and recording metrics when discrepancies are found.
//
// The test checks four key scenarios:
// 1. ErrConflict error + block ID mismatch: Should record 2 failures (one for checksum, one for mismatch)
// 2. ErrConflict error + matching block ID: Still records a failure for the checksum error
// 3. Other error (e.g. ErrFuture) + mismatch: Should record failure only for the block mismatch
// 4. No error + matching block ID: Should record no failures
func TestAsyncVerifyAccessWithRPC(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	// Setup a single-chain dependency set
	chainID := eth.ChainIDFromUInt64(testChainIDOffset)
	fullCfgSet := fullConfigSet(t, 1)

	// Create and set up mock metrics
	mockMetrics := &MockMetrics{}
	// Set up the required method calls that happen during initialization
	mockMetrics.Mock.On("RecordDBEntryCount", chainID, "log", int64(0)).Return()
	mockMetrics.Mock.On("RecordDBEntryCount", chainID, "local_derived", int64(0)).Return()
	mockMetrics.Mock.On("RecordDBEntryCount", chainID, "cross_derived", int64(0)).Return()

	// Initialize backend with mock metrics
	cfg := &config.Config{
		Version:       "test",
		LogConfig:     oplog.CLIConfig{},
		MetricsConfig: opmetrics.CLIConfig{},
		PprofConfig:   oppprof.CLIConfig{},
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               t.TempDir(),
	}
	ex := event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, mockMetrics, cfg, ex)
	require.NoError(t, err)

	// Prepare the access object (only ChainID matters for metrics)
	acc := types.Access{ChainID: chainID}

	// Helper to run a scenario and assert metrics calls
	runScenario := func(name string, stubSeal types.BlockSeal, stubErr error, dbBlock eth.BlockID) {
		t.Run(name, func(t *testing.T) {
			// Reset recorded calls
			mockMetrics.Mock = mock.Mock{}

			// Based on the log output, we observe:
			// 1. When err=ErrConflict: Logs "RPC access checksum failed" and calls RecordAccessListVerifyFailure
			// 2. When err!=ErrConflict: Logs "RPC access check failed mechanically" but doesn't record a metric
			// 3. When seal.ID() != dbBlock: Logs "DB access check result did not match" and calls RecordAccessListVerifyFailure

			// Set expectations for the actual behavior observed
			if errors.Is(stubErr, types.ErrConflict) {
				// Error for checksum failure
				mockMetrics.Mock.On("RecordAccessListVerifyFailure", chainID).Return()
			}

			// Block ID mismatch will always trigger a metrics call
			if seal := stubSeal.ID(); seal != dbBlock {
				mockMetrics.Mock.On("RecordAccessListVerifyFailure", chainID).Return()
			}

			// Override the sync source to return our stubbed result
			b.syncSources.Set(chainID, &fakeSyncSource{chainID: chainID, seal: stubSeal, err: stubErr})

			// Invoke the async verification
			b.asyncVerifyAccessWithRPC(context.Background(), acc, dbBlock)

			// Verify that our expectations were met
			mockMetrics.Mock.AssertExpectations(t)
		})
	}

	// Define a couple of block seals for match vs mismatch
	sealA := types.BlockSeal{Hash: common.HexToHash("0x1"), Number: 10, Timestamp: 100}
	idA := sealA.ID()
	sealB := types.BlockSeal{Hash: common.HexToHash("0x2"), Number: 20, Timestamp: 200}
	idB := sealB.ID()

	// ErrConflict + mismatch => 2 failures (checksum + mismatch)
	runScenario("ErrConflict_mismatch", sealA, types.ErrConflict, idB)
	// ErrConflict + match    => 1 failure  (checksum only)
	runScenario("ErrConflict_match", sealA, types.ErrConflict, idA)
	// Other non-conflict error + mismatch => 1 failure (mismatch only)
	runScenario("OtherErr_mismatch", sealA, types.ErrFuture, idB)
	// No error + match         => 0 failures
	runScenario("NoErr_match", sealA, nil, idA)
}

func TestFailsafeEnabled(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	fullCfgSet := fullConfigSet(t, 1)

	cfg := &config.Config{
		Version:               "test",
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               dataDir,
	}

	ex := event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
	require.NoError(t, err)

	// Test initial state - failsafe should be disabled by default
	enabled, err := b.GetFailsafeEnabled(context.Background())
	require.NoError(t, err)
	require.False(t, enabled, "failsafe should be disabled by default")

	// Test that CheckAccessList works normally in initial state
	err = b.CheckAccessList(context.Background(), []common.Hash{}, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.NoError(t, err, "CheckAccessList should work normally when failsafe is disabled")

	// Test setting failsafe to true
	err = b.SetFailsafeEnabled(context.Background(), true)
	require.NoError(t, err)
	enabled, err = b.GetFailsafeEnabled(context.Background())
	require.NoError(t, err)
	require.True(t, enabled, "failsafe should be enabled after setting to true")

	// Test that CheckAccessList returns ErrFailsafeEnabled when failsafe is enabled
	err = b.CheckAccessList(context.Background(), []common.Hash{}, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.ErrorIs(t, err, types.ErrFailsafeEnabled, "CheckAccessList should return ErrFailsafeEnabled when failsafe is enabled")

	// Test setting failsafe to false
	err = b.SetFailsafeEnabled(context.Background(), false)
	require.NoError(t, err)
	enabled, err = b.GetFailsafeEnabled(context.Background())
	require.NoError(t, err)
	require.False(t, enabled, "failsafe should be disabled after setting to false")

	// Test that CheckAccessList works normally when failsafe is disabled
	err = b.CheckAccessList(context.Background(), []common.Hash{}, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.NoError(t, err, "CheckAccessList should work normally when failsafe is disabled")
}

// TestFailsafeEnabledConfigInitialization confirms the configured failsafe state is correctly initialized
func TestFailsafeEnabledConfigInitialization(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	fullCfgSet := fullConfigSet(t, 1)

	testCases := []struct {
		name            string
		failsafeEnabled bool
	}{
		{
			name:            "FailsafeEnabled",
			failsafeEnabled: true,
		},
		{
			name:            "FailsafeDisabled",
			failsafeEnabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Version:               "test",
				FullConfigSetSource:   fullCfgSet,
				SynchronousProcessors: true,
				MockRun:               false,
				SyncSources:           &syncnode.CLISyncNodes{},
				Datadir:               dataDir,
				FailsafeEnabled:       tc.failsafeEnabled,
			}

			ex := event.NewGlobalSynchronous(context.Background())
			b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
			require.NoError(t, err)

			// Verify that failsafe state matches config after initialization
			enabled, err := b.GetFailsafeEnabled(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.failsafeEnabled, enabled, "failsafe state should match config setting")
		})
	}
}

func TestFailsafeOnInvalidation(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	fullCfgSet := fullConfigSet(t, 1)

	testCases := []struct {
		name                   string
		failsafeOnInvalidation bool
		expectFailsafeEnabled  bool
	}{
		{
			name:                   "FailsafeOnInvalidationEnabled",
			failsafeOnInvalidation: true,
		},
		{
			name:                   "FailsafeOnInvalidationDisabled",
			failsafeOnInvalidation: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Version:                "test",
				FullConfigSetSource:    fullCfgSet,
				SynchronousProcessors:  true,
				MockRun:                false,
				SyncSources:            &syncnode.CLISyncNodes{},
				Datadir:                dataDir,
				FailsafeEnabled:        false, // Start with failsafe disabled
				FailsafeOnInvalidation: tc.failsafeOnInvalidation,
			}

			ex := event.NewGlobalSynchronous(context.Background())
			b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
			require.NoError(t, err)

			// Verify that failsafe starts disabled
			enabled, err := b.GetFailsafeEnabled(context.Background())
			require.NoError(t, err)
			require.False(t, enabled, "failsafe should start disabled")

			// Emit InvalidateLocalSafeEvent
			b.OnEvent(context.Background(), superevents.InvalidateLocalSafeEvent{
				ChainID: eth.ChainIDFromUInt64(testChainIDOffset),
			})

			// Verify that failsafe state matches expectation
			enabled, err = b.GetFailsafeEnabled(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.failsafeOnInvalidation, enabled, "failsafe state should match FailsafeOnInvalidation setting")
		})
	}
}
