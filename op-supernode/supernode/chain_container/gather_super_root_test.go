package chain_container

import (
	"context"
	"math/big"
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// gatherFixture wires a chain container with the existing in-package mocks
// (mockVirtualNode, mockEngineController) and a rollup config that maps
// timestamp == block number. Tests configure the mocks then call
// GatherSuperRootData; the consistency tests additionally bump c.gen
// mid-gather to verify the start/end check fires.
type gatherFixture struct {
	c    *simpleChainContainer
	vn   *mockVirtualNode
	eng  *mockEngineController
	hook func() // invoked once on the first SyncStatus call after the start
}

func newGatherFixture(t *testing.T) *gatherFixture {
	cfg := &opnodecfg.Config{
		Rollup: rollup.Config{
			L2ChainID: big.NewInt(420),
			BlockTime: 1,
		},
	}
	vn := newMockVirtualNode()
	eng := newMockEngineController()
	c := &simpleChainContainer{
		chainID: eth.ChainIDFromUInt64(420),
		log:     createTestLogger(t),
		engine:  eng,
		vn:      vn,
		vncfg:   cfg,
	}
	return &gatherFixture{c: c, vn: vn, eng: eng}
}

// configureCanonical sets up the mocks so a gather call returns a populated
// ChainSuperRootData with the given safe head and block ref. The test can
// then verify the happy path or perturb counters mid-gather.
func (f *gatherFixture) configureCanonical(t *testing.T, num uint64) {
	t.Helper()
	hash := common.Hash{0xaa, byte(num)}
	f.vn.safeHeadL1 = eth.BlockID{Number: 50, Hash: common.Hash{0x50}}
	f.vn.safeHeadL2 = eth.BlockID{Number: num, Hash: hash}
	f.eng.l2BlockRefByNumberResult = eth.L2BlockRef{Number: num, Hash: hash, Time: num}
	f.eng.outputV0Result = &eth.OutputV0{StateRoot: eth.Bytes32{0xee}, BlockHash: hash}
}

func TestGatherSuperRootData_HappyPath(t *testing.T) {
	t.Parallel()
	f := newGatherFixture(t)
	f.configureCanonical(t, 100)

	data, err := f.c.GatherSuperRootData(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, data.SyncStatus)
	require.NotNil(t, data.Verified, "verified data should be populated when c.gen is stable")
	require.NotNil(t, data.Optimistic, "optimistic data should be populated when c.gen is stable")
}

func TestGatherSuperRootData_GenBumpedDuringGather_ReturnsInconsistent(t *testing.T) {
	t.Parallel()
	f := newGatherFixture(t)
	f.configureCanonical(t, 100)

	// Inject a gen bump inside one of the in-flight reads. mockVirtualNode's
	// SyncStatus is invoked multiple times during gather (by SyncStatus,
	// VerifiedAt's LocalSafeBlockAtTimestamp, OptimisticAt's
	// LocalSafeBlockAtTimestamp). Bumping on first call simulates a VN
	// restart or RewindEngine that fires after gather has already captured
	// startGen.
	bumped := false
	f.vn.syncStatusOverride = func() (*eth.SyncStatus, error) {
		if !bumped {
			bumped = true
			f.c.gen.Add(1)
		}
		return &eth.SyncStatus{
			CurrentL1:   eth.L1BlockRef{Hash: f.vn.safeHeadL1.Hash, Number: f.vn.safeHeadL1.Number},
			LocalSafeL2: eth.L2BlockRef{Hash: f.vn.safeHeadL2.Hash, Number: f.vn.safeHeadL2.Number},
		}, nil
	}

	_, err := f.c.GatherSuperRootData(context.Background(), 100)
	require.ErrorIs(t, err, ErrInconsistentSnapshot)
}

// TestGatherSuperRootData_SetVNBumpsGen confirms the chain container's
// generation counter increments on setVN, which is the signal we rely on for
// "VN was recreated mid-gather".
func TestGatherSuperRootData_SetVNBumpsGen(t *testing.T) {
	t.Parallel()
	f := newGatherFixture(t)
	before := f.c.gen.Load()
	f.c.setVN(newMockVirtualNode())
	require.Equal(t, before+1, f.c.gen.Load(), "setVN must bump the generation counter")
}

// TestGatherSuperRootData_RewindEngineBumpsGen confirms RewindEngine bumps
// the generation counter as soon as it claims c.resetting, before any of the
// engine-mutating work runs.
func TestGatherSuperRootData_RewindEngineBumpsGen(t *testing.T) {
	t.Parallel()
	f := newGatherFixture(t)
	f.eng.rewindFunc = func(ctx context.Context, ts uint64) error {
		// At this point RewindEngine has already bumped c.gen.
		return context.Canceled
	}
	before := f.c.gen.Load()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel up-front so the retry loop exits quickly
	_ = f.c.RewindEngine(ctx, 1234, eth.BlockRef{Number: 100})

	require.GreaterOrEqual(t, f.c.gen.Load(), before+1, "RewindEngine must bump the generation counter")
}
