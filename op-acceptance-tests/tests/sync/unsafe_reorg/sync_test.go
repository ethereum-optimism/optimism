package unsafereorg

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

func TestKona_ReorgRecovery(gt *testing.T) {
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

	l1Threshold := uint64(4)
	require.Eventually(func() bool {
		// Advance single L1 block
		require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: common.Hash{}}))
		require.NoError(ts.Next(ctx))
		l1head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		l2Unsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

		logger.Info("l1 info", "l1_head", l1head, "l1_origin", l2Unsafe.L1Origin, "l2Unsafe", l2Unsafe)
		return l2Unsafe.L1Origin.Number >= l1Threshold
	}, 120*time.Second, 2*time.Second)

	l2BlockBeforeReorg := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// needs a few time
	sys.L2CL.Advanced(types.LocalUnsafe, l2BlockBeforeReorg.Number+4, 20)

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
	// sys.L2ELB.ReorgTriggered(l2BlockBeforeReorg, 30)
	// l2BlockAfterReorg := sys.L2ELB.BlockRefByNumber(l2BlockBeforeReorg.Number)
	// require.NotEqual(l2BlockAfterReorg.Hash, l2BlockBeforeReorg.Hash)
	// logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	// attempts := 30
	// dsl.CheckAll(t,
	// 	sys.L2CL.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
	// 	sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
	// 	sys.L2CL.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
	// 	sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
	// )

	time.Sleep(20 * time.Second)

	t.Cleanup(func() {
		{
			head := sys.L2CL.UnsafeHead()
			for i := range head.BlockRef.Number + 1 {
				res := sys.L2EL.BlockRefByNumber(i)
				logger.Info("seq wowow", "bn", res, "l1", res.L1Origin)
			}
		}
		{
			head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
			for i := range head.Number + 1 {
				logger.Info("l1 wowow", "bn", sys.L1EL.BlockRefByNumber(i))
			}
		}
	})
}
