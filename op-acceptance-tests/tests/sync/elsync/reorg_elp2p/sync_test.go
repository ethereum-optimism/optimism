package reorg_elp2p

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

func TestUnsafeGapFillAfterSafeReorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	logger = logger.With("###", "###")

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
	// L2CL will receive unsafe payloads from sequencer
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

func TestUnsafeGapFillAfterUnsafeReorg_RestartOpNode(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	logger = logger.With("###", "###")

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

func TestUnsafeGapFillAfterUnsafeReorg_RestartCLP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	logger = logger.With("###", "###")

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

	// TODO: will verifier automatically detect L1 reorg and drop blocks?
	//	 It reorgs
	// t=2025-10-16T17:32:29.039+0900 lvl=info msg="Chain reorg detected" number=7 hash=0x6708f85e055b90d77e23344bfb226d5229b0d536558286388fbda24d25c3fa08 drop=14 dropfrom=0x5f3bbd8450231da1e87bb6e32c418be72655185167500c909cd4b6325c23748a add=1 addfrom=0x11f6667c2d60dbc970bb511bbb6c44cfdb960b851b006b92088ae24f446cd14c global=true
	// but why does block ref by label still returns 12? Find out

	// Wait until verifier reset and dropped all reorg blocks
	// require.NoError(retry.Do0(ctx, 30, &retry.FixedStrategy{Dur: 2 * time.Second},
	// 	func() error {
	// 		unsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	// 		logger.Info("Node Status", "unsafe", unsafe)
	// 		if unsafe.Hash == rewindTo.Hash {
	// 			return nil
	// 		}
	// 		return errors.New("still resetting")
	// 	}))

	// "reset: detected L1 reorg"
	//  "walking sync start"

	// lets try to enable p2p. does this trigger rewind?
	// logger.Info("Before ConnectPeer")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)
	// logger.Info("After ConnectPeer")

	// Unsafe gap is closed
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 50)

	gt.Cleanup(func() {
		sys.L2Batcher.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLB)
	})
}

func TestUnsafeReorgToSafe(gt *testing.T) {
	// We may do not need this test
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	delta := uint64(10)
	dsl.CheckAll(t,
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30),
	)

	sys.L2CLB.Stop()
	sys.L2ELB.DisconnectPeerWith(sys.L2EL)

	genesis := sys.L2ELB.BlockRefByNumber(0)
	unsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	safe := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.GreaterOrEqual(unsafe.Number, safe.Number)

	// Make unsafe head diff between sequencer EL and verifier EL
	sys.L2CL.Advanced(types.LocalUnsafe, 3, 30)

	logger.Info("Unsafe blocks exists which not yet promoted to safe", "unsafe", unsafe.Number, "safe", safe.Number)

	// Trigger Reorg to block diverged from sequencer
	res := sys.L2ELB.NewPayloadWithFault(sys.L2EL, safe.Number).IsValid()
	sys.L2ELB.ForkchoiceUpdateRaw(res.BlockHash, res.BlockHash, genesis.Hash, nil).IsValid()

	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Safe).Number)
	require.NotEqual(safe.Hash, sys.L2ELB.BlockRefByLabel(eth.Safe).Hash)

	// Trigger Reorg to sequencer produced block
	sys.L2ELB.NewPayload(sys.L2EL, safe.Number).IsValid()
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, safe.Number, safe.Number, 0, nil).IsValid()
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)
	require.Equal(safe.Number, sys.L2ELB.BlockRefByLabel(eth.Safe).Number)
	require.Equal(safe.Hash, sys.L2ELB.BlockRefByLabel(eth.Safe).Hash)

	// Peer again for EL Sync preparation
	sys.L2ELB.PeerWith(sys.L2EL)

	require.GreaterOrEqual(sys.L2EL.BlockRefByLabel(eth.Unsafe).Number, unsafe.Number+1)
	// Trigger EL Sync to fill in the gap
	target := sys.L2EL.BlockRefByNumber(unsafe.Number + 1)
	targetHash := target.Hash
	targetHash[0] = targetHash[0] + 1 // inject fault
	// op-geth logs
	// 	t=2025-10-15T00:40:26.803+0900 lvl=warn msg="Fetching the unknown forkchoice head from network" hash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=debug msg="Attempting to retrieve sync target" peer=dabe984226b4e6ab1e1debfd4dd94d669aba83288d3e95038fef7df26e2d0f16 hash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=debug msg="Fetching batch of headers" id=dabe984226b4e6ab1e1debfd4dd94d669aba83288d3e95038fef7df26e2d0f16 conn=staticdial count=1 fromhash=0x84da371982ce4edcc1eb2fe7cbf2f886265619637645e64a25e13898275b9a30 skip=0 reverse=false global=true
	//  t=2025-10-15T00:40:26.803+0900 lvl=warn msg="Could not retrieve unknown head from peers" global=true
	sys.L2ELB.ForkchoiceUpdateRaw(targetHash, targetHash, genesis.Hash, nil).IsSyncing() // WaitUntilValid(10)

	gt.Cleanup(func() {
		sys.L2ELB.PeerWith(sys.L2EL)
		sys.L2CLB.Start()
	})
}
