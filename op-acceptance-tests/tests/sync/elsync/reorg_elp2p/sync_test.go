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

func TestSafeReorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// sys.L2CL.DisconnectPeer(sys.L2CLB)

	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)
	sys.L1Network.WaitForBlock()
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	for i := range 12 {
		parent := common.Hash{}
		// parent[0] = byte(i)
		_ = i

		// must send empty hash to seq. why?
		require.NoError(ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
		require.NoError(ts.Next(t.Ctx()))
		l1Unsafe := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		logger.Info("### L1", "unsafe", l1Unsafe, "time", l1Unsafe.Time)

		sys.L2Chain.WaitForBlock()
		l2Unsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
		logger.Info("### L2", "unsafe", l2Unsafe, "time", l2Unsafe.Time, "l1Origin", l2Unsafe.L1Origin)
	}

	n := 10
	require.GreaterOrEqual(12, n)
	// select a divergence block to reorg from
	var divergence eth.L1BlockRef
	{
		tip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		require.Greater(tip.Number, uint64(n), "n is larger than L1 tip, cannot reorg out block number `tip-n`")

		divergence = sys.L1EL.BlockRefByNumber(tip.Number - uint64(n))
		logger.Info("### L1 Divergence", "div", divergence)
	}

	tipL2_preReorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)

	logger.Info("### tipL2_preReorg", "number", tipL2_preReorg.Number)

	logger.Info("### L1 Div", "unsafe", sys.L1EL.BlockRefByLabel(eth.Unsafe))
	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	require.NoError(ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: divergence.ParentHash}))
	require.NoError(ts.Next(t.Ctx()))
	logger.Info("### L1 Div", "unsafe", sys.L1EL.BlockRefByLabel(eth.Unsafe))

	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	sys.L1EL.ReorgTriggered(divergence, 5)

	sys.L2CL.Reached(types.CrossSafe, tipL2_preReorg.Number, 50)

	reorg := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	logger.Info("### Reorg", "unsafe", reorg)
	logger.Info("### Reorg", "unsafe", sys.L2ELB.BlockRefByNumber(reorg.Number))

	require.Eventually(func() bool {
		unsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)

		block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
		if err != nil {
			sys.Log.Warn("failed to get L1 block info by number", "number", unsafe.L1Origin.Number, "err", err)
			return false
		}

		sys.Log.Info("current unsafe ref", "tip", unsafe, "tip_origin", unsafe.L1Origin, "l1blk", eth.InfoToL1BlockRef(block))

		// print the chains so we have information to debug if the test fails
		sys.L2Chain.PrintChain()
		sys.L1Network.PrintChain()

		return block.Hash() == unsafe.L1Origin.Hash
	}, 120*time.Second, 7*time.Second, "L1 block origin hash should match hash of block on L1 at that number. If not, it means there was a reorg, and L2 blocks L1Origin field is referencing a reorged block.")

	// confirm all L1Origin fields point to canonical blocks
	require.Eventually(func() bool {
		ref := sys.L2EL.BlockRefByLabel(eth.Unsafe)
		var err error

		// wait until L2 chains' L1Origin points to a L1 block after the one that was reorged
		if ref.L1Origin.Number < divergence.Number {
			return false
		}

		sys.Log.Info("L2 chain progressed, pointing to newer L1 block", "ref", ref, "ref_origin", ref.L1Origin, "divergence", divergence)

		for i := ref.Number; i > 0 && ref.L1Origin.Number >= divergence.Number; i-- {
			ref, err = sys.L2EL.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
			if err != nil {
				return false
			}

			if !sys.L1EL.IsCanonical(ref.L1Origin) {
				return false
			}
		}

		return true
	}, 120*time.Second, 5*time.Second, "all L1Origin fields should point to canonical L1 blocks")

}

func TestUnsafeReorgToSafe(gt *testing.T) {
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
}
