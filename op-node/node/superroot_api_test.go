package node

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// superRootProvider mirrors the minimal interface that op-challenger
// (SuperNodeRootProvider) and op-dispute-mon (SuperRootProvider) require from any
// `superroot_atTimestamp` server. op-node MUST satisfy it.
type superRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
}

// superrootAPIQueryAdapter exposes superrootAPI through superRootProvider for
// compile-time and runtime interface conformance checking. It is NOT registered as
// an RPC service — placing the SuperRootAtTimestamp method on this adapter (rather
// than the live *superrootAPI) avoids leaking a `superroot_superRootAtTimestamp`
// wire method that go-ethereum would auto-register from any exported method on the
// live struct.
type superrootAPIQueryAdapter struct{ *superrootAPI }

func (a superrootAPIQueryAdapter) SuperRootAtTimestamp(ctx context.Context, ts uint64) (eth.SuperRootAtTimestampResponse, error) {
	return a.atTimestamp(ctx, ts)
}

var _ superRootProvider = superrootAPIQueryAdapter{}

const (
	testL2ChainID    = 420
	testGenesisL1Num = uint64(100)
	testBlockTime    = uint64(2)
	testGenesisL2Ts  = uint64(1000)
)

// fixture sets up a realistic single-chain SyncStatus and a deterministic OutputV0.
type fixture struct {
	cfg        *rollup.Config
	dr         *mockDriverClient
	l2Client   *testutils.MockL2Client
	safeDB     *mockSafeDBReader
	api        *superrootAPI
	syncStatus *eth.SyncStatus
	chainID    eth.ChainID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	logger := testlog.Logger(t, log.LevelError)
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(testL2ChainID),
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Number: testGenesisL1Num, Hash: common.Hash{0xa0}},
			L2:     eth.BlockID{Number: 0, Hash: common.Hash{0xb0}},
			L2Time: testGenesisL2Ts,
		},
		BlockTime: testBlockTime,
	}

	syncStatus := &eth.SyncStatus{
		CurrentL1: eth.L1BlockRef{Number: 200, Hash: common.Hash{0xc0}},
		UnsafeL2: eth.L2BlockRef{
			Hash:   common.Hash{0xd0},
			Number: 60,
			Time:   testGenesisL2Ts + 60*testBlockTime,
		},
		SafeL2: eth.L2BlockRef{
			Hash:   common.Hash{0xe0},
			Number: 50,
			Time:   testGenesisL2Ts + 50*testBlockTime,
		},
		LocalSafeL2: eth.L2BlockRef{
			Hash:   common.Hash{0xe1},
			Number: 50,
			Time:   testGenesisL2Ts + 50*testBlockTime,
		},
		FinalizedL2: eth.L2BlockRef{
			Hash:   common.Hash{0xf0},
			Number: 40,
			Time:   testGenesisL2Ts + 40*testBlockTime,
		},
	}

	dr := &mockDriverClient{}
	l2Client := &testutils.MockL2Client{}
	safeDB := &mockSafeDBReader{}
	api := NewSuperrootAPI(cfg, l2Client, dr, safeDB, logger)

	return &fixture{
		cfg:        cfg,
		dr:         dr,
		l2Client:   l2Client,
		safeDB:     safeDB,
		api:        api,
		syncStatus: syncStatus,
		chainID:    eth.ChainIDFromBig(cfg.L2ChainID),
	}
}

func (f *fixture) expectSyncStatus() {
	f.dr.Mock.On("SyncStatus").Return(f.syncStatus)
}

func (f *fixture) expectBlockRef(blockNum uint64, hash common.Hash, l1Origin eth.BlockID) eth.L2BlockRef {
	ref := eth.L2BlockRef{
		Hash:     hash,
		Number:   blockNum,
		Time:     testGenesisL2Ts + blockNum*testBlockTime,
		L1Origin: l1Origin,
	}
	f.dr.ExpectBlockRefWithStatus(blockNum, ref, f.syncStatus, nil)
	return ref
}

func (f *fixture) expectOutputV0(hash common.Hash) *eth.OutputV0 {
	output := &eth.OutputV0{
		StateRoot:                eth.Bytes32{0x11},
		MessagePasserStorageRoot: eth.Bytes32{0x22},
		BlockHash:                hash,
	}
	f.l2Client.ExpectOutputV0AtBlock(hash, output, nil)
	return output
}

func TestSuperrootAPI_PreGenesis(t *testing.T) {
	f := newFixture(t)
	f.expectSyncStatus()

	// Timestamp before genesis.
	resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(testGenesisL2Ts-1))
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{f.chainID}, resp.ChainIDs)
	require.Empty(t, resp.OptimisticAtTimestamp)
	require.Nil(t, resp.Data)
	require.Equal(t, f.syncStatus.CurrentL1.ID(), resp.CurrentL1)
	require.Equal(t, f.syncStatus.SafeL2.Time, resp.CurrentSafeTimestamp)
	require.Equal(t, f.syncStatus.LocalSafeL2.Time, resp.CurrentLocalSafeTimestamp)
	require.Equal(t, f.syncStatus.FinalizedL2.Time, resp.CurrentFinalizedTimestamp)
}

func TestSuperrootAPI_BeyondUnsafe(t *testing.T) {
	f := newFixture(t)
	f.expectSyncStatus()

	// Timestamp maps to block 70, beyond UnsafeL2 (60).
	tsBeyond := testGenesisL2Ts + 70*testBlockTime
	resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsBeyond))
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{f.chainID}, resp.ChainIDs)
	require.Empty(t, resp.OptimisticAtTimestamp)
	require.Nil(t, resp.Data)
}

func TestSuperrootAPI_BeyondLocalSafe_OmitsChain(t *testing.T) {
	// op-supernode parity: blocks beyond LocalSafeL2 are treated as absent for both
	// verified and optimistic, so the chain is omitted from OptimisticAtTimestamp and
	// Data is nil. (LocalSafeBlockAtTimestamp returns ethereum.NotFound there, which
	// makes both VerifiedAt and OptimisticAt fail in op-supernode.)
	f := newFixture(t)
	f.expectSyncStatus()

	// Block 55: between LocalSafeL2 (50) and UnsafeL2 (60).
	tsOpt := testGenesisL2Ts + 55*testBlockTime
	resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsOpt))
	require.NoError(t, err)
	require.Nil(t, resp.Data)
	require.Empty(t, resp.OptimisticAtTimestamp)
}

func TestSuperrootAPI_VerifiedHappyPath(t *testing.T) {
	f := newFixture(t)
	f.expectSyncStatus()

	// Block 40: at-or-before LocalSafeL2 (50).
	tsVerified := testGenesisL2Ts + 40*testBlockTime
	hash := common.Hash{0x40}
	l1Origin := eth.BlockID{Number: 170, Hash: common.Hash{0x17}}
	ref := f.expectBlockRef(40, hash, l1Origin)
	output := f.expectOutputV0(hash)

	// SafeDB.L1AtSafeHead returns the earliest L1 at which the recorded L2 safe head reached target.
	verifiedL1 := eth.BlockID{Number: 205, Hash: common.Hash{0x20}}
	f.safeDB.ExpectL1AtSafeHead(uint64(40), verifiedL1, ref.ID(), nil)

	resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsVerified))
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	require.Equal(t, verifiedL1, resp.Data.VerifiedRequiredL1)
	require.Equal(t, eth.OutputRoot(output), resp.OptimisticAtTimestamp[f.chainID].OutputRoot)
	require.Equal(t, verifiedL1, resp.OptimisticAtTimestamp[f.chainID].RequiredL1)

	expectedSuper := eth.NewSuperV1(tsVerified, eth.ChainIDAndOutput{
		ChainID: f.chainID,
		Output:  eth.OutputRoot(output),
	})
	require.Equal(t, eth.SuperRoot(expectedSuper), resp.Data.SuperRoot)
}

func TestSuperrootAPI_SafeDBNotFound_OmitsChain(t *testing.T) {
	// op-supernode parity: when safeDBAtL2 returns ethereum.NotFound (mapped from
	// ErrL1AtSafeHeadNotFound), both VerifiedAt and OptimisticAt return NotFound and the
	// chain is OMITTED from OptimisticAtTimestamp. Returning an entry with L1Origin would
	// understate the requirement and could let op-challenger include the chain at step > 0
	// where op-supernode would trigger InvalidTransition.
	f := newFixture(t)
	f.expectSyncStatus()

	tsVerified := testGenesisL2Ts + 40*testBlockTime
	hash := common.Hash{0x40}
	l1Origin := eth.BlockID{Number: 170, Hash: common.Hash{0x17}}
	f.expectBlockRef(40, hash, l1Origin)
	f.expectOutputV0(hash)

	// SafeDB returns ErrL1AtSafeHeadNotFound (transient: empty DB or target above latest);
	// op-supernode maps this to ethereum.NotFound and OMITS the chain.
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadNotFound)

	resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsVerified))
	require.NoError(t, err)
	require.Nil(t, resp.Data)
	require.Empty(t, resp.OptimisticAtTimestamp)
}

func TestSuperrootAPI_SafeDBUnavailable_ReturnsError(t *testing.T) {
	// op-supernode parity: ErrL1AtSafeHeadUnavailable (permanent history gap) bubbles up
	// as ErrHistoryUnavailable from the RPC. Operator must intervene; consumers should not
	// silently degrade.
	f := newFixture(t)
	f.expectSyncStatus()

	tsVerified := testGenesisL2Ts + 40*testBlockTime
	hash := common.Hash{0x40}
	l1Origin := eth.BlockID{Number: 170, Hash: common.Hash{0x17}}
	f.expectBlockRef(40, hash, l1Origin)
	f.expectOutputV0(hash)

	// SafeDB returns ErrL1AtSafeHeadUnavailable: target predates recorded history.
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadUnavailable)

	_, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsVerified))
	require.Error(t, err)
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadUnavailable)
}

func TestSuperrootAPI_SafeDBDisabled_Errors(t *testing.T) {
	// A node started without --safedb.path uses safedb.Disabled. Without an up-front
	// gate, requests within LocalSafeL2 would surface ErrNotEnabled from the helper
	// while requests beyond LocalSafeL2 would short-circuit successfully with Data=nil
	// — operators serving dispute infra would see inconsistent responses. Assert the
	// handler rejects loudly before doing any work (no SyncStatus / output calls).
	f := newFixture(t)
	api := NewSuperrootAPI(f.cfg, f.l2Client, f.dr, safedb.Disabled, testlog.Logger(t, log.LevelError))

	_, err := api.AtTimestamp(context.Background(), hexutil.Uint64(testGenesisL2Ts+10*testBlockTime))
	require.ErrorContains(t, err, "safedb not enabled")
}

func TestSuperrootAPI_DriverSyncStatusError(t *testing.T) {
	f := newFixture(t)
	// Replace SyncStatus expectation with a failing one.
	failing := &failingDriver{err: errors.New("drv-fail")}
	api := NewSuperrootAPI(f.cfg, f.l2Client, failing, f.safeDB, testlog.Logger(t, log.LevelError))

	_, err := api.AtTimestamp(context.Background(), hexutil.Uint64(testGenesisL2Ts+10*testBlockTime))
	require.ErrorContains(t, err, "syncStatus")
}

func TestSuperrootAPI_OutputClientError(t *testing.T) {
	f := newFixture(t)
	f.expectSyncStatus()

	tsVerified := testGenesisL2Ts + 30*testBlockTime
	hash := common.Hash{0x30}
	f.expectBlockRef(30, hash, eth.BlockID{Number: 160})
	f.l2Client.ExpectOutputV0AtBlock(hash, (*eth.OutputV0)(nil), errors.New("output-fail"))

	_, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(tsVerified))
	require.ErrorContains(t, err, "outputV0AtBlock")
}

func TestSuperrootAPI_ChainIDsAlwaysSingleEntry(t *testing.T) {
	for _, ts := range []uint64{
		testGenesisL2Ts - 1,                // pre-genesis
		testGenesisL2Ts + 70*testBlockTime, // beyond unsafe
	} {
		t.Run("", func(t *testing.T) {
			f := newFixture(t)
			f.expectSyncStatus()
			resp, err := f.api.AtTimestamp(context.Background(), hexutil.Uint64(ts))
			require.NoError(t, err)
			require.Len(t, resp.ChainIDs, 1)
		})
	}
}

func TestSuperrootAPI_SuperRootProviderCompat(t *testing.T) {
	// Compile-time check at package scope (var _ superRootProvider = ...).
	// Runtime check: invoke through the interface to confirm op-node is drop-in for
	// op-challenger's SuperNodeRootProvider and op-dispute-mon's SuperRootProvider.
	f := newFixture(t)
	f.expectSyncStatus()

	var provider superRootProvider = superrootAPIQueryAdapter{f.api}
	resp, err := provider.SuperRootAtTimestamp(context.Background(), testGenesisL2Ts-1)
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{f.chainID}, resp.ChainIDs)
}

// failingDriver is a minimal driverClient stub that returns a configured error from SyncStatus.
type failingDriver struct {
	err error
}

func (f *failingDriver) SyncStatus(_ context.Context) (*eth.SyncStatus, error) {
	return nil, f.err
}
func (f *failingDriver) BlockRefWithStatus(_ context.Context, _ uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	return eth.L2BlockRef{}, nil, f.err
}
func (f *failingDriver) ResetDerivationPipeline(_ context.Context) error       { return f.err }
func (f *failingDriver) StartSequencer(_ context.Context, _ common.Hash) error { return f.err }
func (f *failingDriver) StopSequencer(_ context.Context) (common.Hash, error) {
	return common.Hash{}, f.err
}
func (f *failingDriver) SequencerActive(_ context.Context) (bool, error)                      { return false, f.err }
func (f *failingDriver) OnUnsafeL2Payload(_ context.Context, _ *eth.ExecutionPayloadEnvelope) {}
func (f *failingDriver) OverrideLeader(_ context.Context) error                               { return f.err }
func (f *failingDriver) ConductorEnabled(_ context.Context) (bool, error)                     { return false, f.err }
func (f *failingDriver) SetRecoverMode(_ context.Context, _ bool) error                       { return f.err }
