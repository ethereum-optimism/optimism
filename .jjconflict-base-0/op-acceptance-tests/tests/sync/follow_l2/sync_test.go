package follow_l2

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

func TestFollowL2_Safe_Finalized_CurrentL1(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	logger := t.Logger()

	// Takes about 2 minutes for L1 finalization
	attempts := 70
	target := uint64(3)

	// L2CL is the sequencer with CL follow source, derivation disabled
	// L2CLB is the verifier without follow source, derivation enabled
	// L2CLC is the verifier with CL follow source, derivation disabled
	// All verifiers must eventually advance unsafe, safe, finalized
	checkMatchedAll := func(lvl types.SafetyLevel) {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
			sys.L2CLC.ReachedFn(lvl, target, attempts),
		)
		dsl.CheckAll(t,
			sys.L2CLB.MatchedFn(sys.L2CL, lvl, attempts),
			sys.L2CLB.MatchedFn(sys.L2CLC, lvl, attempts),
		)
	}

	checkMatchedAll(types.LocalUnsafe)
	logger.Info("Unsafe head advanced due to CLP2P", "target", target)

	checkMatchedAll(types.LocalSafe)
	logger.Info("Safe head followed source", "target", target)

	checkMatchedAll(types.Finalized)
	logger.Info("Finalized head followed source", "target", target)

	attempts = 10
	dsl.CheckAll(t,
		sys.L2CLC.CurrentL1MatchedFn(sys.L2CLB, attempts),
		sys.L2CL.CurrentL1MatchedFn(sys.L2CLB, attempts),
	)
	logger.Info("CurrentL1 followed source", "currentL1", sys.L2CL.SyncStatus().CurrentL1, "currentL1C", sys.L2CLC.SyncStatus().CurrentL1)
}

func TestFollowL2_ReorgRecovery(gt *testing.T) {
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
		l2Safe := sys.L2ELB.BlockRefByLabel(eth.Safe)

		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Safe.L1Origin, "l2Safe", l2Safe)
		// Wait until safe L2 block has L1 origin point after the startL1Block
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2ELB.BlockRefByLabel(eth.Safe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier safe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 3)

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
	//  L2ELB detects L2 Reorg and reorgs. op-geth example logs: "Chain reorg detected"
	sys.L2ELB.ReorgTriggered(l2BlockBeforeReorg, 30)
	l2BlockAfterReorg := sys.L2ELB.BlockRefByNumber(l2BlockBeforeReorg.Number)
	require.NotEqual(l2BlockAfterReorg.Hash, l2BlockBeforeReorg.Hash)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	attempts := 30
	dsl.CheckAll(t,
		sys.L2CL.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
		sys.L2CL.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
	)
}

func TestFollowL2_WithoutCLP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	attempts := 20
	target := uint64(3)

	// L2CLB is the verifier without follow source, derivation enabled
	sys.L2CLB.Advanced(types.LocalUnsafe, target, attempts)

	// The test's primary target is the L2CLC, with follow source and derivation disabled
	// Normally there should be delta between safe head between unsafe head
	status := sys.L2CLC.SyncStatus()
	require.NotEqual(status.LocalSafeL2, status.UnsafeL2)

	logger.Info("Disconnect CLP2P")
	// L2CLC is the verifier with follow source, derivation disabled
	// Disconnect CLP2P of verifier which follow source is enabled
	sys.L2CLC.DisconnectPeer(sys.L2CLB)
	sys.L2CLB.DisconnectPeer(sys.L2CLC)
	sys.L2CLC.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLC)

	// Advance few safe blocks
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)
	sys.L2CLC.Matched(sys.L2CLB, types.LocalSafe, attempts)

	// Make sure the safe head reaches non-moving unsafe head
	sys.L2CLC.Reached(types.LocalSafe, sys.L2CLC.UnsafeHead().BlockRef.Number, attempts)
	// The only data source for L2CLC is the follow source.
	// L2CLC unsafe head will only be advancing with safe head together
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Advance few safe blocks
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Check once again that the unsafe head is moving together with safe head
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Recover CLP2P
	logger.Info("Recover CLP2P")
	sys.L2CLC.ConnectPeer(sys.L2CLB)
	sys.L2CLB.ConnectPeer(sys.L2CLC)
	sys.L2CLC.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLC)

	// Sequencer unsafe payload will arrive to the verifier, triggering EL sync and filling in the unsafe gap
	dsl.CheckAll(t,
		// Match with sequencer with derivation disabled
		sys.L2CLC.MatchedFn(sys.L2CL, types.LocalSafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CL, types.LocalUnsafe, attempts),
		// Match with other verifier with derivation enabled
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
	)

	t.Cleanup(func() {
		sys.L2CLC.ConnectPeer(sys.L2CLB)
		sys.L2CLB.ConnectPeer(sys.L2CLC)
		sys.L2CLC.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLC)
	})
}
