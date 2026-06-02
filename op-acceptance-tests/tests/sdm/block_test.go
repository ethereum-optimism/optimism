package sdm

import (
	"testing"

	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// submitTxWithoutWait sends a transaction to the mempool without waiting for inclusion.
// Returns the PlannedTx whose Included field can be evaluated later.
// The caller must provide a nonce to avoid the default PendingNonce lookup racing between txs.
func submitTxWithoutWait(
	t devtest.T,
	alice *dsl.EOA,
	nonce uint64,
	opts ...txplan.Option,
) *txplan.PlannedTx {
	combined := append([]txplan.Option{
		alice.Plan(),
		txplan.WithNonce(nonce),
	}, opts...)
	ptx := txplan.NewPlannedTx(combined...)
	_, err := ptx.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "failed to submit tx with nonce %d", nonce)
	return ptx
}

type includedTx struct {
	receipt  *types.Receipt
	blockNum uint64
}

func mustFindRepeatedSlotBlock(
	t devtest.T,
	sys *sdmRethSystem,
	minUserTxs int,
	maxAttempts int,
) (*sdmpkg.RPCBlock, []includedTx, uint64) {
	l := t.Logger()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
		stateBloatAddr := deployContract(t, alice, sdmpkg.StateBloatBin)

		const batchSize = 50
		const slotCount = 20
		startNonce := alice.PendingNonce()
		plannedTxs := make([]*txplan.PlannedTx, 0, batchSize)

		l.Info("Submitting repeated-slot workload",
			"attempt", attempt,
			"alice", alice.Address(),
			"contract", stateBloatAddr,
			"startNonce", startNonce,
			"batchSize", batchSize,
			"slotCount", slotCount)

		for i := 0; i < batchSize; i++ {
			nonce := startNonce + uint64(i)
			plannedTxs = append(plannedTxs, submitTxWithoutWait(
				t,
				alice,
				nonce,
				txplan.WithTo(addrPtr(stateBloatAddr)),
				txplan.WithData(sdmpkg.EncodeRun(slotCount)),
				txplan.WithGasLimit(1_000_000),
			))
		}

		blockTxs := make(map[uint64][]includedTx)
		for i, ptx := range plannedTxs {
			receipt, err := ptx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "attempt %d tx %d: failed to get receipt", attempt, i)
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
				"attempt %d tx %d: must succeed", attempt, i)

			itx := includedTx{receipt: receipt, blockNum: bigs.Uint64Strict(receipt.BlockNumber)}
			blockTxs[itx.blockNum] = append(blockTxs[itx.blockNum], itx)
		}

		var targetBlockNum uint64
		var targetIncluded []includedTx
		for blockNum, txs := range blockTxs {
			if len(txs) > len(targetIncluded) {
				targetBlockNum = blockNum
				targetIncluded = txs
			}
		}
		if len(targetIncluded) < minUserTxs {
			l.Warn("Repeated-slot workload did not produce a dense-enough block",
				"attempt", attempt,
				"requiredUserTxs", minUserTxs,
				"bestUserTxs", len(targetIncluded),
				"bestBlock", targetBlockNum)
			continue
		}

		block := getBlockWithTxs(t, sys.L2EL, targetBlockNum)
		t.Require().Greater(len(block.Transactions), 0, "block must have at least one transaction")
		t.Require().Equal(uint64(types.DepositTxType), uint64(block.Transactions[0].Type),
			"position 0 must be a deposit tx (L1 info)")
		return block, targetIncluded, targetBlockNum
	}

	t.Require().FailNowf("repeated-slot workload failed",
		"no block with at least %d user txs found after %d attempts", minUserTxs, maxAttempts)
	return nil, nil, 0
}

func TestSDMDisabledNoRefunds(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, false)
	verifyOpReth(t, sys.L2EL)

	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2, "target block must contain multiple user txs")

	postExecTx, _ := findPostExecTransaction(block)
	t.Require().Nil(postExecTx, "SDM-disabled sequencer must not include a post-exec tx")

	for _, itx := range included {
		refund, present := getOPGasRefund(t, sys.L2EL, itx.receipt.TxHash)
		t.Require().False(present, "legacy block %d tx %s must not expose opGasRefund",
			targetBlockNum, itx.receipt.TxHash)
		t.Require().Zero(refund, "absent opGasRefund must decode to zero")
	}
}

func TestSDMEnabledPayloadAndReplayMatch(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	verifyOpReth(t, sys.L2EL)

	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	postExecTx, postExecPos := findPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM-enabled sequencer must include a post-exec tx")
	t.Require().Greater(len(postExecTx.Input), 0, "post-exec tx input must not be empty")
	t.Require().Equal(uint64(sdmpkg.SDMTxType), uint64(postExecTx.Type), "post-exec tx type must be 0x7D")

	payload, err := sdmpkg.DecodePayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(sdmpkg.PostExecPayloadVersion, payload.Version, "post-exec payload version must be 1")
	t.Require().NotEmpty(payload.GasRefundEntries, "post-exec payload must be non-empty for repeated-slot workload")

	receiptByHash := make(map[common.Hash]*types.Receipt, len(included))
	hasNonZeroReceiptRefund := false
	for _, itx := range included {
		receiptByHash[itx.receipt.TxHash] = itx.receipt
		refund, _ := getOPGasRefund(t, sys.L2EL, itx.receipt.TxHash)
		if refund > 0 {
			hasNonZeroReceiptRefund = true
		}
	}
	t.Require().True(hasNonZeroReceiptRefund, "at least one repeated-slot tx must have non-zero opGasRefund")

	for _, entry := range payload.GasRefundEntries {
		t.Require().Less(int(entry.Index), len(block.Transactions), "payload index must be in block range")
		targetTx := block.Transactions[entry.Index]
		t.Require().NotEqual(uint64(types.DepositTxType), uint64(targetTx.Type), "payload must not target deposits")
		t.Require().NotEqual(uint64(sdmpkg.SDMTxType), uint64(targetTx.Type), "payload must not target the SDM tx itself")

		refund, present := getOPGasRefund(t, sys.L2EL, targetTx.Hash)
		t.Require().True(present, "SDM receipt must expose opGasRefund for tx index %d", entry.Index)
		t.Require().Equal(entry.GasRefund, refund,
			"payload refund must match receipt opGasRefund for tx index %d", entry.Index)
	}

	replay := replayBlockWithSDM(t, sys.L2EL, targetBlockNum)
	t.Require().Equal(targetBlockNum, replay.BlockNum, "replay must target the selected block")
	t.Require().Equal(block.Hash, replay.BlockHash, "replay block hash must match source block")
	t.Require().True(replay.PostExecTxPresent, "replay must report the post-exec tx in the source block")
	t.Require().NotNil(replay.PostExecTxIndex, "replay must report the post-exec tx index")
	t.Require().Equal(uint64(postExecPos), *replay.PostExecTxIndex, "replay post-exec tx index must match source block")
	t.Require().Equal(len(block.Transactions)-1, len(replay.Txs),
		"replay must strip the post-exec tx and preserve the remaining tx ordering")
	t.Require().Empty(replay.Mismatches, "canonical post-exec block should replay without mismatches")
	t.Require().Equal(len(replay.SynthesizedPayload.GasRefundEntries), replay.Summary.PostExecPayloadEntryCount,
		"summary payload entry count must match synthesized payload")

	expectedOriginalIndexes := make([]uint64, 0, len(block.Transactions)-1)
	for i := range block.Transactions {
		if i == postExecPos {
			continue
		}
		expectedOriginalIndexes = append(expectedOriginalIndexes, uint64(i))
	}

	replayRefundByIndex := make(map[uint64]uint64, len(replay.Txs))
	hasReplayRefund := false
	for i, tx := range replay.Txs {
		t.Require().Equal(uint64(i), tx.ReplayTxIndex, "replay tx indexes must be sequential")
		t.Require().Equal(expectedOriginalIndexes[i], tx.TxIndex,
			"replay tx %d must preserve original block index", i)

		sourceTx := block.Transactions[tx.TxIndex]
		t.Require().Equal(sourceTx.Hash, tx.TxHash, "replay tx hash must match source tx at index %d", tx.TxIndex)
		t.Require().Equal(uint64(sourceTx.Type), tx.TxType, "replay tx type must match source tx at index %d", tx.TxIndex)
		t.Require().Equal(uint64(types.DepositTxType) == uint64(sourceTx.Type), tx.IsDepositTx,
			"deposit classification must match source tx at index %d", tx.TxIndex)
		t.Require().Equal(tx.RawGasUsed, tx.CanonicalGasUsed+tx.OPGasRefundReplay,
			"raw gas must equal canonical gas plus refund at tx index %d", tx.TxIndex)

		if tx.OPGasRefundReplay > 0 {
			hasReplayRefund = true
		}
		replayRefundByIndex[tx.TxIndex] = tx.OPGasRefundReplay

		if tx.OPGasRefundPayload != nil {
			t.Require().Equal(*tx.OPGasRefundPayload, tx.OPGasRefundReplay,
				"payload refund must match replay refund at tx index %d", tx.TxIndex)
		}
		if receipt, ok := receiptByHash[tx.TxHash]; ok {
			t.Require().Equal(receipt.GasUsed, tx.CanonicalGasUsed,
				"receipt gasUsed must already be canonical for tx %s", tx.TxHash)
			if refund, _ := getOPGasRefund(t, sys.L2EL, tx.TxHash); refund > 0 {
				t.Require().Greater(tx.RawGasUsed, receipt.GasUsed,
					"raw gas must exceed receipt gas when refund is non-zero for tx %s", tx.TxHash)
			}
		}
	}
	t.Require().True(hasReplayRefund, "replay must produce non-zero refunds for repeated-slot block")

	var totalReplayRefund uint64
	for _, entry := range replay.SynthesizedPayload.GasRefundEntries {
		sourceTx := block.Transactions[entry.Index]
		refund, present := getOPGasRefund(t, sys.L2EL, sourceTx.Hash)
		t.Require().True(present, "SDM receipt must expose opGasRefund for tx index %d", entry.Index)
		t.Require().Equal(refund, entry.GasRefund,
			"synthesized payload refund must match receipt opGasRefund for tx index %d", entry.Index)
		t.Require().Equal(entry.GasRefund, replayRefundByIndex[entry.Index],
			"synthesized payload refund must match replay tx refund for tx index %d", entry.Index)
		totalReplayRefund += entry.GasRefund
	}
	t.Require().Equal(totalReplayRefund, replay.Summary.ReplayRefundTotal,
		"summary replay refund total must match synthesized payload")
	t.Require().Equal(totalReplayRefund, replay.Summary.PayloadRefundTotal,
		"summary payload refund total must match synthesized payload")
	t.Require().Equal(uint64(block.GasUsed), replay.Summary.BlockGasUsed,
		"block gasUsed must already be canonical")
	t.Require().Equal(replay.Summary.BlockRawGasUsed,
		replay.Summary.BlockGasUsed+replay.Summary.ReplayRefundTotal,
		"raw block gas must equal canonical block gas plus total refund")

	dsl.CheckAll(t,
		sys.L2EL.AdvancedFn(eth.Unsafe, 10),
		sys.L2ELVerifier.AdvancedFn(eth.Unsafe, 10),
	)

	t.Logger().Info("TestSDMEnabledPayloadAndReplayMatch passed",
		"block_num", targetBlockNum,
		"block_hash", block.Hash,
		"user_txs", len(included),
		"post_exec_tx_index", postExecPos,
		"payload_entries", len(payload.GasRefundEntries),
		"replay_refund_total", replay.Summary.ReplayRefundTotal,
		"block_gas_used", replay.Summary.BlockGasUsed,
		"block_raw_gas_used", replay.Summary.BlockRawGasUsed)
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
	var sys *sdmRethSystem
	if singular {
		sys = newSDMRethSystemWithBatcherOptions(t, true, withSingularBatcher)
	} else {
		// Use the default SpanBatch path to verify post-exec txs derive after batching.
		sys = newSDMRethSystem(t, true)
	}
	verifyOpReth(t, sys.L2EL)
	verifyOpReth(t, sys.L2ELVerifier)

	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().NotEmpty(included, "target block must include workload transactions")
	postExecTx, postExecPos := findPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM-enabled sequencer must include a post-exec tx before batching")
	t.Require().Greater(len(postExecTx.Input), 0, "post-exec tx input must not be empty")

	payload, err := sdmpkg.DecodePayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode before derivation")
	t.Require().NotEmpty(payload.GasRefundEntries, "post-exec payload must be non-empty for repeated-slot workload")
	targetRef := sys.L2EL.BlockRefByNumber(targetBlockNum)
	t.Require().Equal(block.Hash, targetRef.Hash, "selected post-exec block hash must match canonical sequencer block")

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	sentinel := txplan.NewPlannedTx(
		alice.Plan(),
		txplan.WithTo(addrPtr(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))),
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

func TestSDMStorageRefundBreakdown(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	verifyOpReth(t, sys.L2EL)

	const (
		sameSlotTouches = 100
		manySlotTouches = 100
		maxAttempts     = 3
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
		contract := deployContract(t, alice, slotTouchBin)
		startNonce := alice.PendingNonce()

		planned := []*txplan.PlannedTx{
			submitTxWithoutWait(
				t,
				alice,
				startNonce,
				txplan.WithTo(addrPtr(contract)),
				txplan.WithData(encodeHitSameSlot(1)),
				txplan.WithGasLimit(300_000),
			),
			submitTxWithoutWait(
				t,
				alice,
				startNonce+1,
				txplan.WithTo(addrPtr(contract)),
				txplan.WithData(encodeHitSameSlot(sameSlotTouches)),
				txplan.WithGasLimit(1_500_000),
			),
			submitTxWithoutWait(
				t,
				alice,
				startNonce+2,
				txplan.WithTo(addrPtr(contract)),
				txplan.WithData(encodeHitManySlots(manySlotTouches)),
				txplan.WithGasLimit(3_000_000),
			),
			submitTxWithoutWait(
				t,
				alice,
				startNonce+3,
				txplan.WithTo(addrPtr(contract)),
				txplan.WithData(encodeHitManySlots(manySlotTouches)),
				txplan.WithGasLimit(3_000_000),
			),
		}

		receipts := make([]*types.Receipt, len(planned))
		for i, ptx := range planned {
			receipt, err := ptx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "attempt %d tx %d: failed to get receipt", attempt, i)
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
				"attempt %d tx %d: must succeed", attempt, i)
			receipts[i] = receipt
		}

		sameWarmBlock := bigs.Uint64Strict(receipts[0].BlockNumber)
		sameClaimBlock := bigs.Uint64Strict(receipts[1].BlockNumber)
		manyWarmBlock := bigs.Uint64Strict(receipts[2].BlockNumber)
		manyClaimBlock := bigs.Uint64Strict(receipts[3].BlockNumber)
		if sameWarmBlock != sameClaimBlock || manyWarmBlock != manyClaimBlock {
			t.Logger().Warn("slot-touch workload pairs split across blocks; retrying",
				"attempt", attempt,
				"sameWarmBlock", sameWarmBlock,
				"sameClaimBlock", sameClaimBlock,
				"manyWarmBlock", manyWarmBlock,
				"manyClaimBlock", manyClaimBlock)
			continue
		}

		sameRefund, sameRefundPresent := getOPGasRefund(t, sys.L2EL, receipts[1].TxHash)
		t.Require().True(sameRefundPresent, "SDM receipt must expose opGasRefund for repeated same-slot tx")
		sameReplay := replayBlockWithSDM(t, sys.L2EL, sameClaimBlock)
		sameTx := mustFindReplayTxByHash(t, sameReplay, receipts[1].TxHash)
		t.Require().Equal(sameRefund, sameTx.OPGasRefundReplay,
			"replay refund must match receipt refund for repeated same-slot tx")

		var sameSlotSstoreEvents int
		var sameSlotSstoreRefund uint64
		for i, event := range sameTx.RefundBreakdown {
			if event.Kind != "warm_sstore" {
				continue
			}
			t.Require().Equal(uint64(2100), event.Amount, "same-slot warm SSTORE event %d must be 2100 gas", i)
			t.Require().NotNil(event.Slot, "same-slot warm SSTORE event %d must identify the touched slot", i)
			sameSlotSstoreEvents++
			sameSlotSstoreRefund += event.Amount
		}
		t.Require().Equal(1, sameSlotSstoreEvents,
			"repeating the same warmed storage slot %d times should only produce one warm SSTORE refund event", sameSlotTouches)
		t.Require().Equal(uint64(2100), sameSlotSstoreRefund,
			"repeating the same warmed storage slot should only rebate one warm SSTORE access")

		manyRefund, manyRefundPresent := getOPGasRefund(t, sys.L2EL, receipts[3].TxHash)
		t.Require().True(manyRefundPresent, "SDM receipt must expose opGasRefund for many-slot tx")
		manyReplay := replayBlockWithSDM(t, sys.L2EL, manyClaimBlock)
		manyTx := mustFindReplayTxByHash(t, manyReplay, receipts[3].TxHash)
		t.Require().Equal(manyRefund, manyTx.OPGasRefundReplay,
			"replay refund must match receipt refund for many-slot tx")

		var totalBreakdown uint64
		var manySlotSstoreEvents int
		var manySlotSstoreRefund uint64
		for i, event := range manyTx.RefundBreakdown {
			totalBreakdown += event.Amount
			if event.Kind != "warm_sstore" {
				continue
			}
			t.Require().Equal(uint64(2100), event.Amount, "warm SSTORE refund event %d must be 2100 gas", i)
			t.Require().NotNil(event.Slot, "warm SSTORE refund event %d must identify the warmed slot", i)
			manySlotSstoreEvents++
			manySlotSstoreRefund += event.Amount
		}
		t.Require().Equal(manySlotTouches, manySlotSstoreEvents,
			"touching %d distinct warmed slots should produce %d warm SSTORE refund events", manySlotTouches, manySlotTouches)
		t.Require().Equal(uint64(2100*manySlotTouches), manySlotSstoreRefund,
			"distinct warmed slots should rebate 2100 gas each")
		t.Require().Equal(manyRefund, totalBreakdown,
			"sum of many-slot refund events must equal the receipt-level refund")
		t.Require().Greater(manyRefund, manyTx.RawGasUsed/5,
			"SDM refunds are not capped at the EIP-3529 20%% rule once applied canonically")
		t.Require().Equal(receipts[3].GasUsed, manyTx.CanonicalGasUsed,
			"receipt gasUsed must already be canonical for many-slot tx")
		t.Require().Equal(manyTx.RawGasUsed, manyTx.CanonicalGasUsed+manyRefund,
			"raw gas must equal canonical gas plus SDM refund for many-slot tx")

		return
	}

	t.Require().FailNowf("slot-touch workload failed",
		"no attempt produced both same-slot and many-slot warm/claim pairs in the same block after %d attempts", maxAttempts)
}

// TestSDMMixedWorkloadSmoke submits transactions from multiple categories in a single burst,
// without calling .Eval() between submissions. This tests that different tx types
// (transfer, compute, events, state writes) can be batched into the same block.
func TestSDMMixedWorkloadSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	l := t.Logger()

	clientVersion := verifyOpReth(t, sys.L2EL)
	l.Info("Verified op-reth", "version", clientVersion)

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.ZeroWei)

	computeHeavyAddr := deployContract(t, alice, computeHeavyBin)
	stateBloatAddr := deployContract(t, alice, sdmpkg.StateBloatBin)
	eventLoggerAddr := alice.DeployEventLogger()
	l.Info("Deployed contracts",
		"computeHeavy", computeHeavyAddr,
		"stateBloat", stateBloatAddr,
		"eventLogger", eventLoggerAddr)

	startNonce := alice.PendingNonce()
	type batchEntry struct {
		category string
		ptx      *txplan.PlannedTx
	}

	categories := []struct {
		name string
		opts []txplan.Option
	}{
		{
			name: "eoa_transfer",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(bob.Address())),
				txplan.WithValue(eth.OneHundredthEther),
			},
		},
		{
			name: "compute_heavy",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(computeHeavyAddr)),
				txplan.WithData(sdmpkg.EncodeRun(200)),
				txplan.WithGasLimit(200_000),
			},
		},
		{
			name: "event_emitter",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(eventLoggerAddr)),
				txplan.WithData(encodeEmitLog(3, 64)),
				txplan.WithGasLimit(200_000),
			},
		},
		{
			name: "state_bloat",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(stateBloatAddr)),
				txplan.WithData(sdmpkg.EncodeRun(20)),
				txplan.WithGasLimit(500_000),
			},
		},
		{
			name: "compute_heavy_2",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(computeHeavyAddr)),
				txplan.WithData(sdmpkg.EncodeRun(200)),
				txplan.WithGasLimit(200_000),
			},
		},
		{
			name: "event_emitter_2",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(eventLoggerAddr)),
				txplan.WithData(encodeEmitLog(3, 64)),
				txplan.WithGasLimit(200_000),
			},
		},
		{
			name: "state_bloat_2",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(stateBloatAddr)),
				txplan.WithData(sdmpkg.EncodeRun(20)),
				txplan.WithGasLimit(500_000),
			},
		},
		{
			name: "eoa_transfer_2",
			opts: []txplan.Option{
				txplan.WithTo(addrPtr(bob.Address())),
				txplan.WithValue(eth.OneHundredthEther),
			},
		},
	}
	batch := make([]batchEntry, 0, len(categories))

	l.Info("Submitting batch", "txCount", len(categories), "startNonce", startNonce)

	for i, cat := range categories {
		nonce := startNonce + uint64(i)
		ptx := submitTxWithoutWait(t, alice, nonce, cat.opts...)
		batch = append(batch, batchEntry{category: cat.name, ptx: ptx})
		l.Info("Submitted", "category", cat.name, "nonce", nonce)
	}

	blockCounts := make(map[uint64]int)
	for i, entry := range batch {
		receipt, err := entry.ptx.Included.Eval(t.Ctx())
		t.Require().NoError(err, "tx %d (%s): failed to get receipt", i, entry.category)
		t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
			"tx %d (%s): must succeed", i, entry.category)

		blockNum := bigs.Uint64Strict(receipt.BlockNumber)
		blockCounts[blockNum]++

		refund, _ := getOPGasRefund(t, sys.L2EL, receipt.TxHash)
		l.Info("Included",
			"category", entry.category,
			"block", blockNum,
			"txIdx", receipt.TransactionIndex,
			"gasUsed", receipt.GasUsed,
			"opGasRefund", refund)
	}

	l.Info("Batch distribution", "numBlocks", len(blockCounts))
	maxInBlock := 0
	var maxBlockNum uint64
	for blockNum, count := range blockCounts {
		l.Info("Block", "number", blockNum, "txCount", count)
		if count > maxInBlock {
			maxInBlock = count
			maxBlockNum = blockNum
		}
	}

	if maxInBlock >= 2 {
		l.Info("Multi-tx block found — inspecting for SDM tx",
			"block", maxBlockNum, "txCount", maxInBlock)

		block := getBlockWithTxs(t, sys.L2EL, maxBlockNum)
		postExecTx, postExecPos := findPostExecTransaction(block)
		if postExecTx != nil {
			l.Info("Post-exec transaction present in multi-category block!",
				"block", maxBlockNum,
				"position", postExecPos,
				"inputLen", len(postExecTx.Input))
		} else {
			l.Info("No post-exec tx in block (fork not active yet)",
				"block", maxBlockNum)
		}
	} else {
		l.Warn("All txs landed in separate blocks — no cross-tx warming possible")
	}
}

func addrPtr(addr common.Address) *common.Address {
	return &addr
}
