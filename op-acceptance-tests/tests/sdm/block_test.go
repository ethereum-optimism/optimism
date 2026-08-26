package sdm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSDMDisabledNoRefunds(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, false)
	sdmtest.VerifyOpReth(t, sys.L2EL)

	block, included, targetBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2, "target block must contain multiple user txs")

	postExecTx, _ := sdmpkg.FindPostExecTransaction(block)
	t.Require().Nil(postExecTx, "pre-Lagoon sequencer must not include a post-exec tx")

	for _, receipt := range included {
		refund, present := getOPGasRefund(t, sys.L2EL, receipt.TxHash)
		t.Require().False(present, "legacy block %d tx %s must not expose opGasRefund",
			targetBlockNum, receipt.TxHash)
		t.Require().Zero(refund, "absent opGasRefund must decode to zero")
	}
}

func TestSDMOptInIsInertOnStockOpReth(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	sdmtest.VerifyOpReth(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	block, included, targetBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2, "stock-null target block must contain user txs")
	assertEmptyPostExecCommitment(t, block, "active stock op-reth")
	for _, receipt := range included {
		refund, present := getOPGasRefund(t, sys.L2EL, receipt.TxHash)
		t.Require().False(present, "stock-null receipt must omit opGasRefund")
		t.Require().Zero(refund, "stock-null receipt refund must decode to zero")
	}

	for _, enabled := range []bool{false, true} {
		sdmtest.SetSDMEnabled(t, sys.L2EL, enabled)
		probeBlock, probeReceipt := submitFixtureProbe(t, sys)
		assertEmptyPostExecCommitment(t, probeBlock, fmt.Sprintf("stock op-reth opt-in=%v", enabled))
		_, present := getOPGasRefund(t, sys.L2EL, probeReceipt.TxHash)
		t.Require().False(present, "stock op-reth opt-in=%v receipt must omit opGasRefund", enabled)
	}

	dsl.CheckAll(t,
		sys.L2EL.AdvancedFn(eth.Unsafe, 5),
		sys.L2ELVerifier.AdvancedFn(eth.Unsafe, 5),
	)
	t.Logger().Info("stock op-reth SDM opt-in remained inert", "target_block", targetBlockNum)
}

func TestSDMFixturePayloadReceiptAndAccounting(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newFixtureSDMRethSystem(t)
	sdmtest.VerifySDMFixture(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	block, included, targetBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2, "fixture target block must contain user txs")
	assertFixtureBlockOracle(t, sys, block, targetBlockNum)

	// The 0x7D footprint is zero; other receipts must sum to the block total.
	assertPostExecDAFootprint(t, sys.L2EL, block)

	targetRef := sys.L2EL.BlockRefByNumber(targetBlockNum)
	sys.L2ELVerifier.Reached(eth.Unsafe, targetBlockNum, 60)
	verifierRef := sys.L2ELVerifier.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(targetRef.Hash, verifierRef.Hash,
		"stock verifier must accept the fixture block with the same hash")
	assertFixtureVerifierReceipts(t, sys, block)

	dsl.CheckAll(t,
		sys.L2EL.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELVerifier.AdvancedFn(eth.Unsafe, 10),
	)
}

func TestSDMFixtureOperatorOptInControlsProduction(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newFixtureSDMRethSystem(t)
	sdmtest.VerifySDMFixture(t, sys.L2EL)

	sdmtest.SetSDMEnabled(t, sys.L2EL, false)
	offBlock, offReceipt := submitFixtureProbe(t, sys)
	assertEmptyPostExecCommitment(t, offBlock, "fixture opt-in off")
	_, present := getOPGasRefund(t, sys.L2EL, offReceipt.TxHash)
	t.Require().False(present, "opt-in off receipt must omit opGasRefund")

	sdmtest.SetSDMEnabled(t, sys.L2EL, true)
	onBlock, onReceipt := submitFixtureProbe(t, sys)
	assertFixtureBlockOracle(t, sys, onBlock, bigs.Uint64Strict(onReceipt.BlockNumber))

	sdmtest.SetSDMEnabled(t, sys.L2EL, false)
	offAgainBlock, offAgainReceipt := submitFixtureProbe(t, sys)
	assertEmptyPostExecCommitment(t, offAgainBlock, "fixture opt-in off again")
	_, present = getOPGasRefund(t, sys.L2EL, offAgainReceipt.TxHash)
	t.Require().False(present, "second opt-in off receipt must omit opGasRefund")
}

func assertEmptyPostExecCommitment(t devtest.T, block *sdmpkg.RPCBlock, producer string) {
	postExecTx, postExecPos := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "%s must include the Lagoon post-exec commitment", producer)
	t.Require().Equal(len(block.Transactions)-1, postExecPos, "%s post-exec commitment must be trailing", producer)
	payload, err := optypes.DecodePostExecPayload(postExecTx.Input)
	t.Require().NoError(err, "%s post-exec commitment must decode", producer)
	t.Require().Equal(uint64(block.Number), payload.BlockNumber, "%s commitment must match its block", producer)
	t.Require().NotNil(block.BaseFeePerGas, "%s block must expose baseFeePerGas", producer)
	t.Require().True((*big.Int)(block.BaseFeePerGas).IsUint64(), "%s baseFeePerGas must fit in uint64", producer)
	t.Require().Equal(bigs.Uint64Strict((*big.Int)(block.BaseFeePerGas)), payload.SelectedBaseFeePerGas,
		"%s commitment must match the block base fee", producer)
	t.Require().Empty(payload.GasRefundEntries, "%s must not commit SDM refunds", producer)
}

func submitFixtureProbe(t devtest.T, sys *sdmtest.RethSystem) (*sdmpkg.RPCBlock, *types.Receipt) {
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	probe := txplan.NewPlannedTx(
		alice.Plan(),
		txplan.WithTo(sdmtest.AddrPtr(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))),
		txplan.WithValue(eth.OneHundredthEther),
	)
	receipt, err := probe.Included.Eval(t.Ctx())
	t.Require().NoError(err, "fixture probe transaction must be included")
	t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "fixture probe must succeed")
	blockNum := bigs.Uint64Strict(receipt.BlockNumber)
	return sdmtest.GetBlockWithTxs(t, sys.L2EL, blockNum), receipt
}

func TestSDMPostExecBlockDerivesAndChainProgresses(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, testCase := range []struct {
		name     string
		singular bool
	}{
		{name: "span_batch"},
		{name: "singular_batch", singular: true},
	} {
		t.Run(testCase.name, func(t devtest.T) {
			// Each subtest spins up an independent devstack runtime (the batcher's
			// batch type is the only meaningful difference), so they're safe to run
			// concurrently. Cuts wall-clock from ~76s sequential to ~max(sub1, sub2).
			t.Parallel()
			testSDMPostExecBlockDerivesAndChainProgresses(t, testCase.name, testCase.singular)
		})
	}
}

func testSDMPostExecBlockDerivesAndChainProgresses(t devtest.T, batchType string, singular bool) {
	var sys *sdmtest.RethSystem
	if singular {
		sys = newFixtureSDMRethSystem(t, withSingularBatcher)
	} else {
		// Use the default SpanBatch path to verify post-exec txs derive after batching.
		sys = newFixtureSDMRethSystem(t)
	}
	sdmtest.VerifySDMFixture(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	block, included, targetBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().NotEmpty(included, "target block must include workload transactions")
	postExecTx, postExecPos := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM-enabled sequencer must include a post-exec tx before batching")
	t.Require().Greater(len(postExecTx.Input), 0, "post-exec tx input must not be empty")

	payload, err := optypes.DecodePostExecPayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode before derivation")
	t.Require().NotEmpty(payload.GasRefundEntries, "post-exec payload must be non-empty for fixture workload")
	assertFixtureBlockOracle(t, sys, block, targetBlockNum)
	targetRef := sys.L2EL.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(block.Hash, targetRef.Hash, "selected post-exec block hash must match canonical sequencer block")

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	sentinel := txplan.NewPlannedTx(
		alice.Plan(),
		txplan.WithTo(sdmtest.AddrPtr(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))),
		txplan.WithValue(eth.OneHundredthEther),
	)
	sentinelReceipt, err := sentinel.Included.Eval(t.Ctx())
	t.Require().NoError(err, "sentinel tx after the post-exec block must be included")
	t.Require().Equal(types.ReceiptStatusSuccessful, sentinelReceipt.Status, "sentinel tx must succeed")
	sentinelBlockNum := bigs.Uint64Strict(sentinelReceipt.BlockNumber)
	t.Require().Greater(sentinelBlockNum, targetBlockNum,
		"sentinel tx must land after the post-exec block so derivation proves chain progress past it")
	sentinelRef := sys.L2EL.BlockRefByNumber(sentinelBlockNum)

	l1BeforeBatching := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2CL.ReachedRefFn(safety.CrossSafe, sentinelRef.ID(), 120),
		sys.L2CLVerifier.ReachedRefFn(safety.CrossSafe, sentinelRef.ID(), 120),
		sys.L2EL.ReachedFn(eth.Safe, sentinelBlockNum, 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, sentinelBlockNum, 120),
	)
	l1AfterBatching := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Greater(l1AfterBatching.Number, l1BeforeBatching.Number,
		"L1 must advance while the batch containing the post-exec block is submitted")

	verifierPostExecRef := sys.L2ELVerifier.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(targetRef.Hash, verifierPostExecRef.Hash,
		"verifier must derive the same post-exec block as the sequencer")
	verifierSentinelRef := sys.L2ELVerifier.BlockRefByNumber(sentinelBlockNum)
	t.Require().Equal(sentinelRef.Hash, verifierSentinelRef.Hash,
		"verifier must derive blocks after the post-exec block")

	sequencerStatus := sys.L2CL.SyncStatus()
	verifierStatus := sys.L2CLVerifier.SyncStatus()
	sequencerSafeRef := sys.L2EL.BlockRefByLabel(eth.Safe)
	sequencerFinalizedRef := sys.L2EL.BlockRefByLabel(eth.Finalized)
	verifierSafeRef := sys.L2ELVerifier.BlockRefByLabel(eth.Safe)
	verifierFinalizedRef := sys.L2ELVerifier.BlockRefByLabel(eth.Finalized)
	t.Require().GreaterOrEqual(sequencerStatus.SafeL2.Number, sentinelBlockNum,
		"sequencer SyncStatus safe head must reach the sentinel block")
	t.Require().GreaterOrEqual(verifierStatus.SafeL2.Number, sentinelBlockNum,
		"verifier SyncStatus safe head must reach the sentinel block")
	t.Require().GreaterOrEqual(sequencerSafeRef.Number, sentinelBlockNum,
		"sequencer EL safe head must reach the sentinel block")
	t.Require().GreaterOrEqual(verifierSafeRef.Number, sentinelBlockNum,
		"verifier EL safe head must reach the sentinel block")
	t.Require().LessOrEqual(sequencerStatus.FinalizedL2.Number, sequencerStatus.SafeL2.Number,
		"sequencer SyncStatus finalized head must not be ahead of safe head")
	t.Require().LessOrEqual(verifierStatus.FinalizedL2.Number, verifierStatus.SafeL2.Number,
		"verifier SyncStatus finalized head must not be ahead of safe head")
	t.Require().LessOrEqual(sequencerFinalizedRef.Number, sequencerSafeRef.Number,
		"sequencer EL finalized head must not be ahead of safe head")
	t.Require().LessOrEqual(verifierFinalizedRef.Number, verifierSafeRef.Number,
		"verifier EL finalized head must not be ahead of safe head")

	var totalSDMRefund uint64
	for _, entry := range payload.GasRefundEntries {
		totalSDMRefund += entry.GasRefund
	}
	t.Logger().Info("post exec block num",
		"batch_type", batchType,
		"block_num", targetBlockNum,
		"block_hash", targetRef.Hash)
	t.Logger().Info("post exec tx",
		"batch_type", batchType,
		"tx_hash", postExecTx.Hash,
		"tx_index", postExecPos,
		"tx_type", postExecTx.Type,
		"payload_bytes", len(postExecTx.Input))
	t.Logger().Info("printed SDMGasEntries",
		"batch_type", batchType,
		"entries", payload.GasRefundEntries,
		"entry_count", len(payload.GasRefundEntries),
		"total_sdm_refund", totalSDMRefund)
	t.Logger().Info("L2SafeBlock",
		"batch_type", batchType,
		"post_exec_block_is_safe", sequencerSafeRef.Number >= targetBlockNum && verifierSafeRef.Number >= targetBlockNum,
		"sequencer_safe_block", sequencerSafeRef.ID(),
		"verifier_safe_block", verifierSafeRef.ID(),
		"post_exec_block", targetRef.ID(),
		"sentinel_block", sentinelRef.ID())

	t.Logger().Info("TestSDMPostExecBlockDerivesAndChainProgresses passed",
		"batch_type", batchType,
		"post_exec_block", targetBlockNum,
		"post_exec_block_hash", targetRef.Hash,
		"post_exec_tx_index", postExecPos,
		"payload_entries", len(payload.GasRefundEntries),
		"sentinel_block", sentinelBlockNum,
		"sentinel_block_hash", sentinelRef.Hash,
		"sequencer_sync_status", sequencerStatus,
		"verifier_sync_status", verifierStatus,
		"sequencer_safe_head", sequencerSafeRef.ID(),
		"sequencer_finalized_head", sequencerFinalizedRef.ID(),
		"verifier_safe_head", verifierSafeRef.ID(),
		"verifier_finalized_head", verifierFinalizedRef.ID(),
		"l1_before_batching", l1BeforeBatching.ID(),
		"l1_after_batching", l1AfterBatching.ID())
}

// TestSDMPostExecBlockDerivesOnIsolatedVerifier checks that a verifier with no L2 P2P connectivity
// — one that can only learn the chain by deriving it from L1 — still reproduces a PostExec block
// and keeps its safe head moving.
//
// This exercises the force-build path: with no gossiped unsafe block to consolidate against,
// op-node hands the derived attributes (which embed the block's 0x7D tx) to the EL to rebuild via
// FCU-with-attributes (`no_tx_pool = true`). The verifier never opts into SDM production, so its EL
// must still rebuild the block in Verify mode purely from the embedded payload. The
// consolidation-path test (TestSDMPostExecBlockDerivesAndChainProgresses) cannot catch a
// regression here because its verifier receives the block over P2P first and only consolidates.
func TestSDMPostExecBlockDerivesOnIsolatedVerifier(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSDMRethSystemWithIsolatedVerifier(t)
	sdmtest.VerifySDMFixture(t, sys.L2EL)
	sdmtest.VerifyOpReth(t, sys.L2ELVerifier)

	// Produce a PostExec block on the sequencer, plus a sentinel after it so derivation must carry
	// the verifier past the PostExec block to succeed.
	block, included, targetBlockNum := sdmtest.MustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().NotEmpty(included, "target block must include workload transactions")
	postExecTx, _ := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM-enabled sequencer must include a post-exec tx before batching")
	payload, err := optypes.DecodePostExecPayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().NotEmpty(payload.GasRefundEntries,
		"post-exec payload must be non-empty for fixture workload")
	assertFixtureBlockOracle(t, sys, block, targetBlockNum)
	targetRef := sys.L2EL.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(block.Hash, targetRef.Hash,
		"selected post-exec block must match canonical sequencer block")

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	sentinel := txplan.NewPlannedTx(
		alice.Plan(),
		txplan.WithTo(sdmtest.AddrPtr(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))),
		txplan.WithValue(eth.OneHundredthEther),
	)
	sentinelReceipt, err := sentinel.Included.Eval(t.Ctx())
	t.Require().NoError(err, "sentinel tx after the post-exec block must be included")
	t.Require().Equal(types.ReceiptStatusSuccessful, sentinelReceipt.Status, "sentinel tx must succeed")
	sentinelBlockNum := bigs.Uint64Strict(sentinelReceipt.BlockNumber)
	t.Require().Greater(sentinelBlockNum, targetBlockNum, "sentinel must land after the post-exec block")
	sentinelRef := sys.L2EL.BlockRefByNumber(sentinelBlockNum)

	// Precondition that makes this test meaningful: with the batcher stopped and the verifier off
	// the P2P mesh, the verifier has learned nothing — its unsafe head is still at genesis while
	// the sequencer is well ahead. So any safe progress below comes from L1 derivation + force-build,
	// not from gossip + consolidation.
	sequencerUnsafeBefore := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verifierUnsafeBefore := sys.L2ELVerifier.BlockRefByLabel(eth.Unsafe)
	t.Require().GreaterOrEqual(sequencerUnsafeBefore.Number, targetBlockNum,
		"sequencer must have built the post-exec block locally before batching")
	t.Require().Equal(uint64(0), verifierUnsafeBefore.Number,
		"isolated verifier must not receive any unsafe blocks over P2P; it should still be at genesis")

	// Start batching: the verifier can now derive from L1. With no unsafe block to consolidate
	// against, op-node force-builds each derived block — including the PostExec block — through the
	// EL payload builder.
	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2CLVerifier.ReachedRefFn(safety.CrossSafe, sentinelRef.ID(), 120),
		sys.L2ELVerifier.ReachedFn(eth.Safe, sentinelBlockNum, 120),
	)

	// The verifier rebuilt the PostExec block (and the blocks around it) byte-for-byte: a Verify-mode
	// rejection or a duplicated 0x7D would change the block hash and stall safe progress.
	verifierPostExecRef := sys.L2ELVerifier.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(targetRef.Hash, verifierPostExecRef.Hash,
		"isolated verifier must rebuild the same post-exec block as the sequencer")
	verifierSentinelRef := sys.L2ELVerifier.BlockRefByNumber(sentinelBlockNum)
	t.Require().Equal(sentinelRef.Hash, verifierSentinelRef.Hash,
		"isolated verifier must rebuild blocks after the post-exec block")

	t.Logger().Info("TestSDMPostExecBlockDerivesOnIsolatedVerifier passed",
		"post_exec_block", targetBlockNum,
		"post_exec_block_hash", targetRef.Hash,
		"sentinel_block", sentinelBlockNum,
		"payload_entries", len(payload.GasRefundEntries),
		"verifier_unsafe_before_batching", verifierUnsafeBefore.Number)
}
