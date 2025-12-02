package reorg

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
)

// TestUnsafeGapFillAfterSafeReorg demonstrates the sequence:
//  1. Verifier CLP2P is disconnected and Verifier CL is stopped.
//  2. Safe reorg occurs because L1 reorged.
//  3. Verifier restarts, and consolidation drops the verifier previously-unsafe blocks.
//  4. CLP2P is restored, the verifier backfills and the unsafe gap is closed.
func TestUnsafeGapFillAfterSafeReorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

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
		l2Safe := sys.L2EL.BlockRefByLabel(eth.Safe)
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Safe.L1Origin, "l2Safe", l2Safe)
		// Wait until safe L2 block has L1 origin point after the startL1Block
		return l2Safe.Number > 0 && l2Safe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Safe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier safe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 3)

	// Disconnect CLP2P
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	// Stop verifier CL
	sys.L2CLB.Stop()

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
	//  Batcher re-submits batch using updated L1 view
	sys.L2EL.Reached(eth.Safe, l2BlockAfterReorg.Number, 30)
	require.GreaterOrEqual(sys.L1EL.BlockRefByNumber(l2BlockAfterReorg.L1Origin.Number).Number, l1BlockAfterReorg.Number)

	// Check the divergence before restarting verifier L2CLB
	verUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	seqUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Unsafe heads", "seq", seqUnsafe, "ver", verUnsafe)
	// Verifier unsafe head cannot advance yet because L2CLB is down
	require.Greater(seqUnsafe.Number, verUnsafe.Number)
	// Verifier unsafe head diverged
	canonicalFromSeq := sys.L2EL.BlockRefByNumber(verUnsafe.Number)
	require.NotEqual(canonicalFromSeq.Hash, verUnsafe.Hash)
	logger.Info("Verifer unsafe head diverged", "verUnsafe", verUnsafe, "canonical", canonicalFromSeq)
	var rewindTo eth.L2BlockRef
	for i := verUnsafe.Number; i > 0; i-- {
		ver := sys.L2ELB.BlockRefByNumber(i)
		seq := sys.L2EL.BlockRefByNumber(i)
		if ver.Hash == seq.Hash {
			rewindTo = ver
			break
		}
	}
	logger.Info("Verifier diverged", "rewindTo", rewindTo)
	require.Greater(l1BlockAfterReorg.Number, rewindTo.L1Origin.Number)

	// Restart verifier L2CL. CLP2P disabled
	sys.L2CLB.Start()

	// Safe block reorged. Verifier L2CL will read the new L1 and reorg the safe chain
	// Unsafe head will also be updated because safe chain reorged
	sys.L2ELB.ReorgTriggered(l2BlockBeforeReorg, 10)
	logger.Info("Triggered L2 safe reorg at verifier", "l2", l2BlockAfterReorg)

	sys.L2ELB.Matched(sys.L2EL, eth.Safe, 5)

	// L2CLB has no P2P connection, so unsafe gap always exists
	seqUnsafe = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	// Reenable CLP2P
	// L2CLB will receive unsafe payloads from sequencer
	// Unsafe gap will be observed by the L2CLB, and it will be smart enough to close the gap,
	// using RR Sync(soon be deprecated), or rely on EL Sync(desired)
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	// Unsafe gap is closed
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 50)

	seqUnsafe = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap closed", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	gt.Cleanup(func() {
		sys.L2CLB.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}

// TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL demonstrates the flow where:
//  1. Verifier L2CL is stopped.
//  2. Unsafe reorg occurs because L1 reorged,
//  3. Verifier restarts and detects the L1 reorg, triggering its own unsafe reorg,
//  4. Verifier then backfills and closes the unsafe gap once reconnected via CLP2P.
func TestUnsafeGapFillAfterUnsafeReorg_RestartL2CL(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Stop the batcher not to advance safe head
	sys.L2Batcher.Stop()

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
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Unsafe", l2Unsafe)
		// Wait until unsafe L2 block has L1 origin point after the startL1Block
		return l2Unsafe.Number > 0 && l2Unsafe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	// Pick reorg block
	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make few more unsafe blocks which will be reorged out
	sys.L2EL.Advanced(eth.Unsafe, 4)
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	// Stop Verifier CL
	sys.L2CLB.Stop()

	// Reorg L1 block which unsafe block L1 Origin points to
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

	// Check the divergence before restarting verifier L2CLB
	verUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	seqUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Unsafe heads", "seq", seqUnsafe, "ver", verUnsafe)
	// Verifier unsafe head cannot advance yet because L2CLB is down
	require.Greater(seqUnsafe.Number, verUnsafe.Number)
	// Verifier unsafe head diverged
	canonicalFromSeq := sys.L2EL.BlockRefByNumber(verUnsafe.Number)
	require.NotEqual(canonicalFromSeq.Hash, verUnsafe.Hash)
	logger.Info("Verifer unsafe head diverged", "verUnsafe", verUnsafe, "canonical", canonicalFromSeq)
	var rewindTo eth.L2BlockRef
	for i := verUnsafe.Number; i > 0; i-- {
		ver := sys.L2ELB.BlockRefByNumber(i)
		seq := sys.L2EL.BlockRefByNumber(i)
		if ver.Hash == seq.Hash {
			rewindTo = ver
			break
		}
	}
	logger.Info("Verifier diverged", "rewindTo", rewindTo)
	require.Greater(l1BlockAfterReorg.Number, rewindTo.L1Origin.Number)

	// Restart verifier L2CL
	// L2CL walks back. op-node example logs "walking sync start"
	// Dropping L2 blocks which has invalid L1 origin, until we reach rewindTo
	sys.L2CLB.Start()

	// Make sure CLP2P is connected
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	// L2CLB will receive unsafe payloads from sequencer
	// Unsafe gap will be observed by the L2CLB, and it will be smart enough to close the gap,
	// using RR Sync(soon be deprecated), or rely on EL Sync(desired)

	// Unsafe gap is closed
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 50)

	seqUnsafe = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap closed", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	gt.Cleanup(func() {
		sys.L2Batcher.Start()
		sys.L2CLB.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}

// TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P demonstrates the flow where:
//  1. Verifier CLP2P is disconnected.
//  2. Unsafe reorg occurs because L1 reorged.
//  3. Verifier detects the L1 reorg, triggering its own unsafe reorg.
//  4. CLP2P is restored Verifier, the verifier backfills and the unsafe gap is closed.
func TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// Stop the batcher not to advance safe head
	sys.L2Batcher.Stop()

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
		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Unsafe", l2Unsafe)
		// Wait until unsafe L2 block has L1 origin point after the startL1Block
		return l2Unsafe.Number > 0 && l2Unsafe.L1Origin.Number > startL1Block.Number
	}, 120*time.Second, 2*time.Second)

	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	// Pick reorg block
	l2BlockBeforeReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make few more unsafe blocks which will be reorged out
	sys.L2EL.Advanced(eth.Unsafe, 4)
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	// Disconnect CLP2P
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	// verUnsafe will eventually reorged out
	verUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// Reorg L1 block which unsafe block L1 Origin points to
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

	// L2CLB is still up but only have access to L1 to update canonical view
	// verifier cannot advance unsafe head, but only reorging out blocks
	// Test can still independently find rewindTo
	rewindTo := sys.L2ELB.BlockRefByNumber(0)
	for i := verUnsafe.Number; i > 0; i-- {
		ref, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
		if err != nil {
			// May be not found since verifier EL reorging
			continue
		}
		if ref.L1Origin.Number < l1BlockAfterReorg.Number {
			rewindTo = ref
			break
		}
	}
	logger.Info("Verifier diverged", "rewindTo", rewindTo)

	// Wait until verifier reset and dropped all reorg blocks
	sys.L2CLB.Reset(types.LocalUnsafe, rewindTo)
	logger.Info("Verifier rewind done", "rewindTo", rewindTo)

	// Make sure CLP2P is connected
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	// L2CLB will receive unsafe payloads from sequencer
	// Unsafe gap will be observed by the L2CLB, and it will be smart enough to close the gap,
	// using RR Sync(soon be deprecated), or rely on EL Sync(desired)

	// Unsafe gap is closed
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 50)

	seqUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verUnsafe = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Verifier unsafe gap closed", "gap", seqUnsafe.Number-verUnsafe.Number, "seqUnsafe", seqUnsafe.Number, "verUnsafe", verUnsafe.Number)

	gt.Cleanup(func() {
		sys.L2Batcher.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}
