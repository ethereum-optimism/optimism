package unsafe_only

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
)

func TestUnsafeOnly_ReorgRecovery(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// L2CLB is the verifier without follow source, derivation enabled

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)

	// Pass the L1 genesis
	sys.L1Network.WaitForBlock()

	// Stop auto advancing L1
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	require.Eventually(func() bool {
		// Advance single L1 block
		require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: common.Hash{}}))
		require.NoError(ts.Next(ctx))
		l1head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		l2Unsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)

		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Safe", l2Unsafe)
		// Wait until unsafe L2 block has L1 origin point after the startL1Block
		return l2Unsafe.Number > 0 && l2Unsafe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier unsafe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Unsafe, l2BlockBeforeReorg.Number, 3)

	// Reorg L1 block which safe block L1 Origin points to
	l1BlockBeforeReorg := sys.L1EL.BlockRefByNumber(l2BlockBeforeReorg.L1Origin.Number)
	logger.Info("Triggering L1 reorg", "l1", l1BlockBeforeReorg)
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1BlockBeforeReorg.ParentHash}))
	require.NoError(ts.Next(ctx))

	// Start advancing L1
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	// Make sure L1 reorged
	sys.L1EL.WaitForBlockNumber(l1BlockBeforeReorg.Number)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)
	logger.Info("Triggered L1 reorg", "l1", l1BlockAfterReorg)
	require.NotEqual(l1BlockAfterReorg.Hash, l1BlockBeforeReorg.Hash)

	// Need to poll until the L2CL detects L1 Reorg and trigger L2 Reorg
	// What happens:
	//  L2CL detects L1 Reorg and reset the pipeline. op-node example logs: "reset: detected L1 reorg"
	//  L2EL detects L2 Reorg and reorgs. op-geth example logs: "Chain reorg detected"
	sys.L2EL.ReorgTriggered(l2BlockBeforeReorg, 30)
	l2BlockAfterReorg := sys.L2EL.BlockRefByNumber(l2BlockBeforeReorg.Number)
	require.NotEqual(l2BlockAfterReorg.Hash, l2BlockBeforeReorg.Hash)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	attempts := 30
	dsl.CheckAll(t,
		sys.L2CL.MatchedFn(sys.L2CLC, types.LocalUnsafe, attempts),
		sys.L2CLB.MatchedFn(sys.L2CLC, types.LocalUnsafe, attempts),
		sys.L2EL.MatchedFn(sys.L2ELC, types.LocalUnsafe, attempts),
		sys.L2ELB.MatchedFn(sys.L2ELC, types.LocalUnsafe, attempts),
	)
}

func TestUnsafeOnly_VerifierUnsafeGapClosed(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()
	attempts := 10

	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2EL.MatchedUnsafe(sys.L2ELB, attempts)
	sys.L2CL.MatchedUnsafe(sys.L2CLB, attempts)

	// Case 1: Closing the gap starting from genesis
	sys.L2CLB.Stop()
	sys.L2ELB.DisconnectPeerWith(sys.L2EL)
	// Wipe EL to genesis
	sys.L2ELB.Stop()
	sys.L2ELB.Start()
	// Check EL rewinded to genesis. Unsafe gap introduced
	sys.L2ELB.UnsafeHead().IsGenesis()
	// Verifier CL triggers EL Sync to close the gap including genesis
	sys.L2CLB.Start()
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2ELB.PeerWith(sys.L2EL)
	// Gap is closed
	sys.L2CLB.MatchedUnsafe(sys.L2CL, attempts)
	sys.L2ELB.MatchedUnsafe(sys.L2EL, attempts)

	// Case 2: Closing the gap not starting from genesis
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, 3)
	// Turn back the CLP2P
	sys.L2CLB.ConnectPeer(sys.L2CL)
	// gap is closed again
	sys.L2CLB.MatchedUnsafe(sys.L2CL, attempts)
	sys.L2ELB.MatchedUnsafe(sys.L2EL, attempts)

	// Derivation did not happen
	sys.L2CL.SafeHead().IsGenesis()

	// Derivation happened at the second verifier
	require.Greater(sys.L2CLC.SafeHead().BlockRef.Number, uint64(0))

	t.Cleanup(func() {
		sys.L2ELB.Start()
		sys.L2ELB.PeerWith(sys.L2EL)
		sys.L2CLB.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
	})
}

func TestUnsafeOnly_SequencerRestart(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()

	attempts := 10

	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2EL.MatchedUnsafe(sys.L2ELB, attempts)
	sys.L2CL.MatchedUnsafe(sys.L2CLB, attempts)

	// Stop the sequencer
	sys.L2CL.Stop()
	sys.L2ELB.NotAdvancedUnsafe(3)

	// Restart the sequencer
	sys.L2CL.Start()
	// Sequencer produces blocks again
	sys.L2CL.AdvancedUnsafe(3, attempts)

	// Derivation did not happen at sequencer
	sys.L2CL.SafeHead().IsGenesis()

	// Stop the sequencer with API
	sys.L2CL.StopSequencer()
	sys.L2ELB.NotAdvancedUnsafe(3)

	// Restart the sequencer with API
	sys.L2CL.StartSequencer()
	// Sequencer produces blocks again
	sys.L2CL.AdvancedUnsafe(3, attempts)

	// Derivation did not happen at sequencer
	sys.L2CL.SafeHead().IsGenesis()

	// Derivation happened at the second verifier
	safeHeadNum := sys.L2CLC.SafeHead().BlockRef.Number
	require.Greater(safeHeadNum, uint64(0))

	t.Cleanup(func() {
		sys.L2CL.Start()
	})
}
