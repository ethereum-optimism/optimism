package derivation_parity

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	holoceneBoundaryForks = "pre-Holocene L2 block, post-Holocene L1 block"

	rejectedNearHoloceneDivergence = "A derived payload the engine rejects near Holocene activation: " +
		"kona retries the rejected payload, op-node replaces it with a deposits-only block"
)

// TestDerivationParityRejectedPayloadNearHoloceneActivation covers a derived payload that the
// engine rejects, for an L2 block from before Holocene activation whose batch is in an L1 block
// from after it. From Holocene on, a node replaces a rejected derived payload with a deposits-only
// block. op-node decides whether Holocene applies from the L1 block it derived the payload from,
// kona-node from the L2 block's timestamp.
//
// Pinned outcome:
//   - control: the batch's payload is valid. Both nodes derive it and record the same safe head.
//   - crafted: the batch carries a transfer from an account with no balance. Batch validation does
//     not check balances, so both nodes derive the block, and the engine rejects it. op-node replaces
//     it with a deposits-only block and keeps deriving. kona-node finds Holocene inactive and keeps
//     retrying the rejected payload, so its safe head stays at the block before.
//
// A fix to either node fails the crafted case: flip the kona-node assertions, and the record to
// agree.
func TestDerivationParityRejectedPayloadNearHoloceneActivation(gt *testing.T) {
	recordWhenPassed(gt, &parityreport.Record{Case: "control", Forks: holoceneBoundaryForks, Agree: true})
	recordWhenPassed(gt, &parityreport.Record{Case: "crafted", Forks: holoceneBoundaryForks, Divergence: rejectedNearHoloceneDivergence})
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t, withHoloceneAfterGenesis(holoceneOffset))
	require := t.Require()
	rollupCfg := sys.L2Network.Escape().RollupConfig()

	// Both blocks are from before Holocene, and both batches go to L1 after it activates there.
	const controlNum, craftedNum = 1, 2
	require.False(rollupCfg.IsHolocene(sys.L2Network.TimestampForBlockNum(craftedNum)), "the crafted block must be from before Holocene")
	sys.SequencerEL.Reached(eth.Unsafe, craftedNum, 30)
	control := sequencerBlock(sys, controlNum)
	crafted := sequencerBlock(sys, craftedNum)
	sys.L1EL.WaitForTime(*rollupCfg.HoloceneTime)

	// control: the sequencer's block 1 plus a transfer from the prefunded funder. It differs from
	// the block the verifiers hold as unsafe, so both build it from the derived attributes.
	funded := signedTransfer(t, sys, sys.Funder, 0, baseFee(control))
	controlL1 := sys.L2Network.PostBatches([]*eth.ExecutionPayload{withTransaction(t, control, funded)})
	require.True(rollupCfg.IsHolocene(controlL1.Time), "the control batch must be in a post-Holocene L1 block")
	dsl.CheckAll(t,
		sys.OpNode.ReachedFn(safety.LocalSafe, controlNum, 30),
		sys.Kona.ReachedFn(safety.LocalSafe, controlNum, 30))
	sys.OpNodeEL.AssertUserTxInBlock(controlNum, funded.Hash())
	sys.OpNode.AwaitMinL1Processed(controlL1.Number)
	sys.Kona.AwaitMinL1Processed(controlL1.Number)
	controlRecord := sys.OpNode.SafeHeadRecordAtL1Block(controlL1.Number)
	require.Equal(&eth.SafeHeadResponse{L1Block: controlL1.ID(), SafeHead: sys.OpNodeEL.BlockRefByNumber(controlNum).ID()}, controlRecord,
		"op-node must record the control block as safe at the control batch's L1 block")
	require.Equal(controlRecord, sys.Kona.SafeHeadRecordAtL1Block(controlL1.Number),
		"kona-node must record the same safe head as op-node at the control batch's L1 block")

	// crafted: the sequencer's block 2, on top of the control block, plus a transfer from an
	// account with no balance.
	unfunded, err := crypto.ToECDSA(crypto.Keccak256([]byte("derivation parity: unfunded account")))
	require.NoError(err)
	dsl.NewKey(t, unfunded).User(sys.SequencerEL).VerifyBalanceExact(eth.ZeroWei)
	crafted.ParentHash = controlRecord.SafeHead.Hash
	craftedL1 := sys.L2Network.PostBatches([]*eth.ExecutionPayload{
		withTransaction(t, crafted, signedTransfer(t, sys, unfunded, 0, baseFee(crafted))),
	})
	require.True(rollupCfg.IsHolocene(craftedL1.Time), "the crafted batch must be in a post-Holocene L1 block")

	// op-node finds Holocene active at the batch's L1 block and replaces the rejected block.
	sys.OpNode.Reached(safety.LocalSafe, craftedNum, 30)
	sys.OpNode.AwaitMinL1Processed(craftedL1.Number)
	verifyDepositsOnly(t, sys.OpNodeEL, craftedNum)
	require.Equal(&eth.SafeHeadResponse{L1Block: craftedL1.ID(), SafeHead: sys.OpNodeEL.BlockRefByNumber(craftedNum).ID()},
		sys.OpNode.SafeHeadRecordAtL1Block(craftedL1.Number),
		"op-node must record the deposits-only replacement as safe at the crafted batch's L1 block")

	// kona-node finds Holocene inactive at the block's timestamp and keeps retrying the rejected
	// payload. Once its derivation reaches the crafted batch's L1 block, honest batches for the
	// blocks after it move op-node on and leave kona-node behind, its derivation stuck at that block.
	sys.Kona.AwaitMinL1Processed(craftedL1.Number - 1)
	sys.Sequencer.Reached(safety.LocalSafe, craftedNum, 30)
	sys.L2Batcher.Start()
	sys.OpNode.Reached(safety.LocalSafe, craftedNum+3, 60)
	konaStatus := sys.Kona.SyncStatus()
	require.Equal(craftedL1.Number, konaStatus.CurrentL1.Number,
		"kona-node's derivation must stay at the crafted batch's L1 block")
	require.Greater(sys.OpNode.SyncStatus().CurrentL1.Number, craftedL1.Number,
		"op-node's derivation must have moved past the crafted batch's L1 block")
	require.Equal(controlRecord.SafeHead, konaStatus.LocalSafeL2.ID(),
		"kona-node must keep the control block as its safe head")
	require.Equal(controlRecord, sys.Kona.SafeHeadRecordAtL1Block(craftedL1.Number),
		"kona-node must record nothing for the crafted batch's L1 block")
}
