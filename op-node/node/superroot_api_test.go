package node

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	api := NewSuperrootAPI(cfg, l2Client, dr, safeDB)

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

// expectBlockRef mocks a successful BlockRefWithStatus, returning ref alongside
// the fixture's syncStatus (no separate SyncStatus expectation needed).
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

// expectBlockMissing mocks the beyond-unsafe path: NotFound paired with a non-nil
// status, per the driver contract.
func (f *fixture) expectBlockMissing(blockNum uint64) {
	f.dr.ExpectBlockRefWithStatus(blockNum, eth.L2BlockRef{}, f.syncStatus, ethereum.NotFound)
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
	// Pre-genesis: surface TargetBlockNumber's error rather than a silent empty response.
	f := newFixture(t)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts-1)
	require.ErrorContains(t, err, "target block number")
}

func TestSuperrootAPI_BeyondUnsafe(t *testing.T) {
	// Block 70 > UnsafeL2 (60): omit chain, populate sync fields via the SyncStatus fallback.
	f := newFixture(t)
	f.expectBlockMissing(70)

	resp, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+70*testBlockTime)
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{f.chainID}, resp.ChainIDs)
	require.Empty(t, resp.OptimisticAtTimestamp)
	require.Nil(t, resp.Data)
}

func TestSuperrootAPI_BeyondLocalSafe_OmitsChain(t *testing.T) {
	// Block 55: between LocalSafeL2 (50) and UnsafeL2 (60). op-supernode omits the
	// chain here; we do too.
	f := newFixture(t)
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

func TestSuperrootAPI_VerifiedAtGenesisL2(t *testing.T) {
	// L2 genesis is trivially safe at L1 block 0; SafeDB is not consulted.
	f := newFixture(t)

	genesisHash := f.cfg.Genesis.L2.Hash
	f.expectBlockRef(0, genesisHash, eth.BlockID{Number: 0})
	output := f.expectOutputV0(genesisHash)

	resp, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	require.Equal(t, eth.BlockID{Number: 0}, resp.Data.VerifiedRequiredL1)
	require.Equal(t, eth.OutputRoot(output), resp.OptimisticAtTimestamp[f.chainID].OutputRoot)
	require.Equal(t, eth.BlockID{Number: 0}, resp.OptimisticAtTimestamp[f.chainID].RequiredL1)
	f.safeDB.Mock.AssertNotCalled(t, "L1AtSafeHead")
}

func TestSuperrootAPI_SafeDBNotFound_Errors(t *testing.T) {
	// Block is at-or-below LocalSafeL2; missing SafeDB record is inconsistency, propagate.
	f := newFixture(t)

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadNotFound)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadNotFound)
}

func TestSuperrootAPI_SafeDBUnavailable_ReturnsError(t *testing.T) {
	// Permanent SafeDB gap: propagate. Operator must intervene.
	f := newFixture(t)

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)
	f.safeDB.ExpectL1AtSafeHead(uint64(40), eth.BlockID{}, eth.BlockID{}, safedb.ErrL1AtSafeHeadUnavailable)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadUnavailable)
}

func TestSuperrootAPI_SafeDBDisabled_Errors(t *testing.T) {
	// Without --safedb.path, safedb.Disabled returns ErrNotEnabled; propagate it.
	f := newFixture(t)
	api := NewSuperrootAPI(f.cfg, f.l2Client, f.dr, safedb.Disabled)

	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.expectOutputV0(hash)

	_, err := api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, safedb.ErrNotEnabled)
}

func TestSuperrootAPI_BlockRefError_Propagates(t *testing.T) {
	// Non-NotFound BlockRefWithStatus errors fail the RPC (no silent degradation).
	f := newFixture(t)
	driverErr := errors.New("driver-loop fail")
	f.dr.ExpectBlockRefWithStatus(40, eth.L2BlockRef{}, nil, driverErr)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, driverErr)
}

func TestSuperrootAPI_OutputNotFound_Errors(t *testing.T) {
	// Ref already resolved; later NotFound on its hash means state shifted, propagate.
	f := newFixture(t)
	hash := common.Hash{0x40}
	f.expectBlockRef(40, hash, eth.BlockID{Number: 170})
	f.l2Client.ExpectOutputV0AtBlock(hash, (*eth.OutputV0)(nil), ethereum.NotFound)

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+40*testBlockTime)
	require.ErrorIs(t, err, ethereum.NotFound)
}

func TestSuperrootAPI_OutputClientError(t *testing.T) {
	// Non-NotFound OutputV0AtBlock errors propagate.
	f := newFixture(t)
	hash := common.Hash{0x30}
	f.expectBlockRef(30, hash, eth.BlockID{Number: 160})
	f.l2Client.ExpectOutputV0AtBlock(hash, (*eth.OutputV0)(nil), errors.New("output-fail"))

	_, err := f.api.atTimestamp(context.Background(), testGenesisL2Ts+30*testBlockTime)
	require.ErrorContains(t, err, "outputV0AtBlock")
}
