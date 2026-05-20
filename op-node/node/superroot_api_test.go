package node

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

const (
	testL2ChainID    = 420
	testGenesisL1Num = uint64(100)
	testBlockTime    = uint64(2)
	testGenesisL2Ts  = uint64(1000)
)

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

// expectBlockRef mocks a successful BlockRefWithStatus, returning ref alongside the
// fixture's syncStatus (the production code uses that status for the response, so
// callers don't need a separate SyncStatus expectation).
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

// expectBlockMissing mocks BlockRefWithStatus failing for blockNum (e.g. beyond unsafe
// head) and the SyncStatus fallback that production code makes to populate the response.
func (f *fixture) expectBlockMissing(blockNum uint64) {
	f.dr.ExpectBlockRefWithStatus(blockNum, eth.L2BlockRef{}, nil, errors.New("not found"))
	f.dr.Mock.On("SyncStatus").Return(f.syncStatus)
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
	// Pre-genesis: rollup.TargetBlockNumber errors. Surface that error directly so
	// dispute infra (which never legitimately queries before genesis) gets a clear
	// signal rather than a successful empty response.
	f := newFixture(t)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts-1)
	require.Error(t, err)
	require.ErrorContains(t, err, "target block number")
}

func TestSuperrootAPI_BeyondUnsafe(t *testing.T) {
	f := newFixture(t)

	// Block 70: beyond UnsafeL2 (60). BlockRefWithStatus fails; we fall back to
	// SyncStatus and omit the chain.
	f.expectBlockMissing(70)

	resp, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+70*testBlockTime)
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{f.chainID}, resp.ChainIDs)
	require.Empty(t, resp.OptimisticAtTimestamp)
	require.Nil(t, resp.Data)
}

func TestSuperrootAPI_BeyondLocalSafe_OmitsChain(t *testing.T) {
	// op-supernode parity: blocks beyond LocalSafeL2 fail both VerifiedAt and OptimisticAt
	// (LocalSafeBlockAtTimestamp returns ethereum.NotFound), so the chain is omitted from
	// OptimisticAtTimestamp and Data is nil.
	f := newFixture(t)

	// Block 55: between LocalSafeL2 (50) and UnsafeL2 (60). BlockRefWithStatus succeeds
	// (block exists) but blockNum > LocalSafeL2 triggers the omit path.
	f.expectBlockRef(55, common.Hash{0x55}, eth.BlockID{Number: 180})

	resp, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+55*testBlockTime)
	require.NoError(t, err)
	require.Nil(t, resp.Data)
	require.Empty(t, resp.OptimisticAtTimestamp)
}

func TestSuperrootAPI_VerifiedHappyPath(t *testing.T) {
	f := newFixture(t)

	tsVerified := testGenesisL2Ts + 40*testBlockTime
	hash := common.Hash{0x40}
	ref := f.expectBlockRef(40, hash, eth.BlockID{Number: 170, Hash: common.Hash{0x17}})
	output := f.expectOutputV0(hash)

	verifiedL1 := eth.BlockID{Number: 205, Hash: common.Hash{0x20}}
	f.safeDB.ExpectL1AtSafeHead(uint64(40), verifiedL1, ref.ID(), nil)

	resp, err := f.api.atTimestamp(context.Background(), tsVerified)
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
	// SafeDB lag (target above latest, or DB empty) is transient; op-supernode maps it
	// to ethereum.NotFound and omits the chain.
	f := newFixture(t)

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadNotFound)

	resp, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.NoError(t, err)
	require.Nil(t, resp.Data)
	require.Empty(t, resp.OptimisticAtTimestamp)
}

func TestSuperrootAPI_SafeDBUnavailable_ReturnsError(t *testing.T) {
	// Permanent SafeDB gap (target predates recorded history) — op-supernode bubbles
	// ErrHistoryUnavailable to the RPC; operator must intervene.
	f := newFixture(t)

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadUnavailable)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadUnavailable)
}

func TestSuperrootAPI_SafeDBDisabled_Errors(t *testing.T) {
	// A node started without --safedb.path uses safedb.Disabled, whose L1AtSafeHead
	// returns safedb.ErrNotEnabled. That error must propagate (not degrade to Data nil).
	f := newFixture(t)
	api := NewSuperrootAPI(f.cfg, f.l2Client, f.dr, safedb.Disabled, testlog.Logger(t, log.LevelError))

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)

	_, err := api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, safedb.ErrNotEnabled)
}

func TestSuperrootAPI_DriverError(t *testing.T) {
	// Driver failures (both the primary BlockRefWithStatus call and the SyncStatus
	// fallback) surface as a single error from the RPC.
	f := newFixture(t)
	failing := &failingDriver{err: errors.New("drv-fail")}
	api := NewSuperrootAPI(f.cfg, f.l2Client, failing, f.safeDB, testlog.Logger(t, log.LevelError))

	_, err := api.atTimestamp(context.Background(), testGenesisL2Ts+10*testBlockTime)
	require.ErrorContains(t, err, "drv-fail")
}

func TestSuperrootAPI_OutputClientError(t *testing.T) {
	f := newFixture(t)

	hash := common.Hash{0x30}
	f.expectBlockRef(30, hash, eth.BlockID{Number: 160})
	f.l2Client.ExpectOutputV0AtBlock(hash, (*eth.OutputV0)(nil), errors.New("output-fail"))

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+30*testBlockTime)
	require.ErrorContains(t, err, "outputV0AtBlock")
}

// failingDriver returns a configured error from every driverClient method.
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

