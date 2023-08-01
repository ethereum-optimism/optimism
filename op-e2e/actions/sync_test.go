package actions

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDerivationWithFlakyL1RPC(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError) // mute all the temporary derivation errors that we forcefully create
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	rng := rand.New(rand.NewSource(1234))
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build a L1 chain with 20 blocks and matching L2 chain and batches to test some derivation work
	miner.ActEmptyBlock(t)
	for i := 0; i < 20; i++ {
		sequencer.ActL1HeadSignal(t)
		sequencer.ActL2PipelineFull(t)
		sequencer.ActBuildToL1Head(t)
		batcher.ActSubmitAll(t)
		miner.ActL1StartBlock(12)(t)
		miner.ActL1IncludeTx(batcher.batcherAddr)(t)
		miner.ActL1EndBlock(t)
	}
	// Make verifier aware of head
	verifier.ActL1HeadSignal(t)

	// Now make the L1 RPC very flaky: requests will randomly fail with 50% chance
	miner.MockL1RPCErrors(func() error {
		if rng.Intn(2) == 0 {
			return errors.New("mock rpc error")
		}
		return nil
	})

	// And sync the verifier
	verifier.ActL2PipelineFull(t)
	// Verifier should be synced, even though it hit lots of temporary L1 RPC errors
	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Safe(), "verifier is synced")
}

func TestFinalizeWhileSyncing(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError) // mute all the temporary derivation errors that we forcefully create
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	verifierStartStatus := verifier.SyncStatus()

	// Build an L1 chain with 64 + 1 blocks, containing batches of L2 chain.
	// Enough to go past the finalityDelay of the engine queue,
	// to make the verifier finalize while it syncs.
	miner.ActEmptyBlock(t)
	for i := 0; i < 64+1; i++ {
		sequencer.ActL1HeadSignal(t)
		sequencer.ActL2PipelineFull(t)
		sequencer.ActBuildToL1Head(t)
		batcher.ActSubmitAll(t)
		miner.ActL1StartBlock(12)(t)
		miner.ActL1IncludeTx(batcher.batcherAddr)(t)
		miner.ActL1EndBlock(t)
	}
	l1Head := miner.l1Chain.CurrentHeader()
	// finalize all of L1
	miner.ActL1Safe(t, l1Head.Number.Uint64())
	miner.ActL1Finalize(t, l1Head.Number.Uint64())

	// Now signal L1 finality to the verifier, while the verifier is not synced.
	verifier.ActL1HeadSignal(t)
	verifier.ActL1SafeSignal(t)
	verifier.ActL1FinalizedSignal(t)

	// Now sync the verifier, without repeating the signal.
	// While it's syncing, it should finalize on interval now, based on the future L1 finalized block it remembered.
	verifier.ActL2PipelineFull(t)

	// Verify the verifier finalized something new
	require.Less(t, verifierStartStatus.FinalizedL2.Number, verifier.SyncStatus().FinalizedL2.Number, "verifier finalized L2 blocks during sync")
}

func TestUnsafeSync(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlInfo)

	sd, _, _, sequencer, seqEng, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)
	seqEngCl, err := sources.NewEngineClient(seqEng.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	for i := 0; i < 10; i++ {
		// Build a L2 block
		sequencer.ActL2StartBlock(t)
		sequencer.ActL2EndBlock(t)
		// Notify new L2 block to verifier by unsafe gossip
		seqHead, err := seqEngCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
		require.NoError(t, err)
		verifier.ActL2UnsafeGossipReceive(seqHead)(t)
		// Handle unsafe payload
		verifier.ActL2PipelineFull(t)
		// Verifier must advance its unsafe head and engine sync target.
		require.Equal(t, sequencer.L2Unsafe().Hash, verifier.L2Unsafe().Hash)
		// Check engine sync target updated.
		require.Equal(t, sequencer.L2Unsafe().Hash, sequencer.EngineSyncTarget().Hash)
		require.Equal(t, verifier.L2Unsafe().Hash, verifier.EngineSyncTarget().Hash)
	}
}

func TestEngineP2PSync(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlInfo)

	miner, seqEng, sequencer := setupSequencerTest(t, sd, log)
	// Enable engine P2P sync
	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), &sync.Config{EngineSync: true})

	seqEngCl, err := sources.NewEngineClient(seqEng.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	verifierUnsafeHead := verifier.L2Unsafe()

	// Build a L2 block. This block will not be gossiped to verifier, so verifier can not advance chain by itself.
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)

	for i := 0; i < 10; i++ {
		// Build a L2 block
		sequencer.ActL2StartBlock(t)
		sequencer.ActL2EndBlock(t)
		// Notify new L2 block to verifier by unsafe gossip
		seqHead, err := seqEngCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
		require.NoError(t, err)
		verifier.ActL2UnsafeGossipReceive(seqHead)(t)
		// Handle unsafe payload
		verifier.ActL2PipelineFull(t)
		// Verifier must advance only engine sync target.
		require.NotEqual(t, sequencer.L2Unsafe().Hash, verifier.L2Unsafe().Hash)
		require.NotEqual(t, verifier.L2Unsafe().Hash, verifier.EngineSyncTarget().Hash)
		require.Equal(t, verifier.L2Unsafe().Hash, verifierUnsafeHead.Hash)
		require.Equal(t, sequencer.L2Unsafe().Hash, verifier.EngineSyncTarget().Hash)
	}
}
