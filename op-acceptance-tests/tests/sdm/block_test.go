package sdm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdmreplay"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// rpcTransaction is a minimal representation of a transaction from eth_getBlockByNumber
// with full transactions. We use raw JSON to avoid depending on op-geth types for deposit
// fields that may differ between op-geth and op-reth RPC responses.
type rpcTransaction struct {
	Hash                common.Hash     `json:"hash"`
	Type                hexutil.Uint64  `json:"type"`
	From                common.Address  `json:"from"`
	To                  *common.Address `json:"to"`
	Input               hexutil.Bytes   `json:"input"`
	Gas                 hexutil.Uint64  `json:"gas"`
	IsSystemTx          *bool           `json:"isSystemTx,omitempty"`          // op-geth style
	IsSystemTransaction *bool           `json:"isSystemTransaction,omitempty"` // op-reth style
	SourceHash          *common.Hash    `json:"sourceHash,omitempty"`
}

// rpcBlock is a minimal representation of a block from eth_getBlockByNumber(n, true).
type rpcBlock struct {
	Number       hexutil.Uint64   `json:"number"`
	Hash         common.Hash      `json:"hash"`
	GasUsed      hexutil.Uint64   `json:"gasUsed"`
	Transactions []rpcTransaction `json:"transactions"`
}

type replaySdmPayloadEntry struct {
	Index     uint64 `json:"index"`
	GasRefund uint64 `json:"gas_refund"`
}

type replaySdmPayload struct {
	Version uint64                  `json:"version"`
	Entries []replaySdmPayloadEntry `json:"gas_refund_entries"`
}

type replaySdmRefundEvent struct {
	ClaimingReplayTxIndex      uint64         `json:"claiming_replay_tx_index"`
	ClaimingTxIndex            uint64         `json:"claiming_tx_index"`
	Kind                       string         `json:"kind"`
	Amount                     uint64         `json:"amount"`
	Address                    common.Address `json:"address"`
	Slot                       *common.Hash   `json:"slot"`
	FirstWarmedByReplayTxIndex uint64         `json:"first_warmed_by_replay_tx_index"`
	FirstWarmedByTxIndex       uint64         `json:"first_warmed_by_tx_index"`
}

type replaySdmTx struct {
	TxIndex            uint64                 `json:"tx_index"`
	ReplayTxIndex      uint64                 `json:"replay_tx_index"`
	TxHash             common.Hash            `json:"tx_hash"`
	TxType             uint64                 `json:"tx_type"`
	IsDepositTx        bool                   `json:"is_deposit_tx"`
	GasUsed            uint64                 `json:"gas_used"`
	RawGasUsed         uint64                 `json:"raw_gas_used"`
	CanonicalGasUsed   uint64                 `json:"canonical_gas_used"`
	OPGasRefundReplay  uint64                 `json:"op_gas_refund_replay"`
	OPGasRefundPayload *uint64                `json:"op_gas_refund_payload"`
	OPGasRefundReceipt *uint64                `json:"op_gas_refund_receipt"`
	EffectiveGas       uint64                 `json:"effective_gas"`
	RefundBreakdown    []replaySdmRefundEvent `json:"refund_breakdown"`
	Mismatch           bool                   `json:"mismatch"`
}

type replaySdmMismatch struct {
	Category string  `json:"category"`
	BlockNum uint64  `json:"block_num"`
	TxIndex  *uint64 `json:"tx_index"`
	Expected *uint64 `json:"expected"`
	Actual   *uint64 `json:"actual"`
	Message  string  `json:"message"`
}

type replaySdmSummary struct {
	BlockNum                  uint64      `json:"block_num"`
	BlockHash                 common.Hash `json:"block_hash"`
	TxCountTotal              int         `json:"tx_count_total"`
	TxCountUser               int         `json:"tx_count_user"`
	PostExecTxPresent         bool        `json:"post_exec_tx_present"`
	PostExecPayloadEntryCount int         `json:"post_exec_payload_entry_count"`
	BlockGasUsed              uint64      `json:"block_gas_used"`
	BlockRawGasUsed           uint64      `json:"block_raw_gas_used"`
	ReplayRefundTotal         uint64      `json:"replay_refund_total"`
	PayloadRefundTotal        uint64      `json:"payload_refund_total"`
	NodeReceiptRefundTotal    uint64      `json:"node_receipt_refund_total"`
	BlockEffectiveGas         uint64      `json:"block_effective_gas"`
	MismatchCount             int         `json:"mismatch_count"`
	ReplayMode                string      `json:"replay_mode"`
}

type replaySdmBlock struct {
	BlockNum                uint64              `json:"block_num"`
	BlockHash               common.Hash         `json:"block_hash"`
	ParentHash              common.Hash         `json:"parent_hash"`
	PostExecTxPresent       bool                `json:"post_exec_tx_present"`
	PostExecTxIndex         *uint64             `json:"post_exec_tx_index"`
	EmbeddedPayload         *replaySdmPayload   `json:"embedded_payload"`
	SynthesizedPayload      replaySdmPayload    `json:"synthesized_payload"`
	SynthesizedPayloadBytes hexutil.Bytes       `json:"synthesized_payload_bytes"`
	Txs                     []replaySdmTx       `json:"txs"`
	Mismatches              []replaySdmMismatch `json:"mismatches"`
	Summary                 replaySdmSummary    `json:"summary"`
}

// getBlockWithTxs fetches a block by number with full transaction objects via raw JSON RPC.
func getBlockWithTxs(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *rpcBlock {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var raw json.RawMessage
	err := rpcClient.CallContext(context.Background(), &raw, "eth_getBlockByNumber",
		fmt.Sprintf("0x%x", blockNum), true)
	t.Require().NoError(err, "eth_getBlockByNumber RPC failed for block %d", blockNum)
	t.Require().NotNil(raw, "block %d not found", blockNum)

	var block rpcBlock
	err = json.Unmarshal(raw, &block)
	t.Require().NoError(err, "failed to unmarshal block %d", blockNum)
	return &block
}

func replayBlockWithSDM(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) *replaySdmBlock {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var raw json.RawMessage
	err := rpcClient.CallContext(context.Background(), &raw, "debug_replaySDMBlock",
		fmt.Sprintf("0x%x", blockNum),
		map[string]bool{
			"compare_payload":  true,
			"compare_receipts": true,
		},
	)
	t.Require().NoError(err, "debug_replaySDMBlock RPC failed for block %d", blockNum)
	t.Require().NotNil(raw, "replay result for block %d must not be nil", blockNum)

	var replay replaySdmBlock
	err = json.Unmarshal(raw, &replay)
	t.Require().NoError(err, "failed to unmarshal replay result for block %d", blockNum)
	return &replay
}

// findPostExecTransaction searches for the post-exec tx anywhere in the block.
// Returns the transaction and its position if found, nil/-1 otherwise.
// The post-exec tx is identified purely by type 0x7D.
func findPostExecTransaction(block *rpcBlock) (*rpcTransaction, int) {
	for i := range block.Transactions {
		tx := &block.Transactions[i]
		if uint64(tx.Type) != 0x7d {
			continue
		}
		return tx, i
	}
	return nil, -1
}

func mustFindReplayTxByHash(t devtest.T, replay *replaySdmBlock, txHash common.Hash) *replaySdmTx {
	for i := range replay.Txs {
		if replay.Txs[i].TxHash == txHash {
			return &replay.Txs[i]
		}
	}

	t.Require().FailNowf("replay tx missing", "tx %s not found in replay for block %d", txHash, replay.BlockNum)
	return nil
}

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
	txIndex  int
	blockNum uint64
}

func mustFindRepeatedSlotBlock(
	t devtest.T,
	sys *sdmRethSystem,
	minUserTxs int,
	maxAttempts int,
) (*rpcBlock, []includedTx, uint64) {
	l := t.Logger()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
		stateBloatAddr := deployContract(t, alice, stateBloatBin)

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
				txplan.WithData(encodeRun(slotCount)),
				txplan.WithGasLimit(1_000_000),
			))
		}

		blockTxs := make(map[uint64][]includedTx)
		for i, ptx := range plannedTxs {
			receipt, err := ptx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "attempt %d tx %d: failed to get receipt", attempt, i)
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
				"attempt %d tx %d: must succeed", attempt, i)

			itx := includedTx{receipt: receipt, txIndex: i, blockNum: bigs.Uint64Strict(receipt.BlockNumber)}
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

func TestSDMDisabledLegacyAccounting(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, false)
	verifyOpReth(t, sys.L2EL)

	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	t.Require().GreaterOrEqual(len(included), 2, "target block must contain multiple user txs")

	postExecTx, _ := findPostExecTransaction(block)
	t.Require().Nil(postExecTx, "SDM-disabled sequencer must not include a post-exec tx")

	for _, itx := range included {
		refund := getOPGasRefund(t, sys.L2EL, itx.receipt.TxHash)
		t.Require().Zero(refund, "legacy block %d tx %s must not expose opGasRefund",
			targetBlockNum, itx.receipt.TxHash)
	}
}

func TestSDMEnabledCanonicalGasAccounting(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	verifyOpReth(t, sys.L2EL)

	block, included, targetBlockNum := mustFindRepeatedSlotBlock(t, sys, 2, 3)
	postExecTx, postExecPos := findPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "SDM-enabled sequencer must include a post-exec tx")
	t.Require().Greater(len(postExecTx.Input), 0, "post-exec tx input must not be empty")
	t.Require().Equal(uint64(0x7d), uint64(postExecTx.Type), "post-exec tx type must be 0x7D")

	payload, err := sdmreplay.DecodePayload(postExecTx.Input)
	t.Require().NoError(err, "post-exec payload must decode")
	t.Require().Equal(uint64(1), payload.Version, "post-exec payload version must be 1")
	t.Require().NotEmpty(payload.GasRefundEntries, "post-exec payload must be non-empty for repeated-slot workload")

	receiptByHash := make(map[common.Hash]*types.Receipt, len(included))
	hasNonZeroReceiptRefund := false
	for _, itx := range included {
		receiptByHash[itx.receipt.TxHash] = itx.receipt
		refund := getOPGasRefund(t, sys.L2EL, itx.receipt.TxHash)
		if refund > 0 {
			hasNonZeroReceiptRefund = true
		}
	}
	t.Require().True(hasNonZeroReceiptRefund, "at least one repeated-slot tx must have non-zero opGasRefund")

	for _, entry := range payload.GasRefundEntries {
		t.Require().Less(int(entry.Index), len(block.Transactions), "payload index must be in block range")
		targetTx := block.Transactions[entry.Index]
		t.Require().NotEqual(uint64(types.DepositTxType), uint64(targetTx.Type), "payload must not target deposits")
		t.Require().NotEqual(uint64(0x7d), uint64(targetTx.Type), "payload must not target the SDM tx itself")

		refund := getOPGasRefund(t, sys.L2EL, targetTx.Hash)
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
	t.Require().Equal(len(replay.SynthesizedPayload.Entries), replay.Summary.PostExecPayloadEntryCount,
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
		t.Require().Equal(tx.GasUsed, tx.CanonicalGasUsed,
			"replay gas_used must already be canonical at tx index %d", tx.TxIndex)
		t.Require().Equal(tx.EffectiveGas, tx.CanonicalGasUsed,
			"replay effective_gas must alias canonical gas at tx index %d", tx.TxIndex)
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
		if tx.OPGasRefundReceipt != nil {
			t.Require().Equal(*tx.OPGasRefundReceipt, tx.OPGasRefundReplay,
				"receipt refund must match replay refund at tx index %d", tx.TxIndex)
		}

		if receipt, ok := receiptByHash[tx.TxHash]; ok {
			t.Require().Equal(receipt.GasUsed, tx.CanonicalGasUsed,
				"receipt gasUsed must already be canonical for tx %s", tx.TxHash)
			if refund := getOPGasRefund(t, sys.L2EL, tx.TxHash); refund > 0 {
				t.Require().Greater(tx.RawGasUsed, receipt.GasUsed,
					"raw gas must exceed receipt gas when refund is non-zero for tx %s", tx.TxHash)
			}
		}
	}
	t.Require().True(hasReplayRefund, "replay must produce non-zero refunds for repeated-slot block")

	var totalReplayRefund uint64
	for _, entry := range replay.SynthesizedPayload.Entries {
		sourceTx := block.Transactions[entry.Index]
		refund := getOPGasRefund(t, sys.L2EL, sourceTx.Hash)
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
	t.Require().Equal(totalReplayRefund, replay.Summary.NodeReceiptRefundTotal,
		"summary receipt refund total must match synthesized payload")
	t.Require().Equal(uint64(block.GasUsed), replay.Summary.BlockGasUsed,
		"block gasUsed must already be canonical")
	t.Require().Equal(replay.Summary.BlockRawGasUsed,
		replay.Summary.BlockGasUsed+replay.Summary.ReplayRefundTotal,
		"raw block gas must equal canonical block gas plus total refund")

	t.Logger().Info("TestSDMEnabledCanonicalGasAccounting passed",
		"block_num", targetBlockNum,
		"block_hash", block.Hash,
		"user_txs", len(included),
		"post_exec_tx_index", postExecPos,
		"payload_entries", len(payload.GasRefundEntries),
		"replay_refund_total", replay.Summary.ReplayRefundTotal,
		"block_gas_used", replay.Summary.BlockGasUsed,
		"block_raw_gas_used", replay.Summary.BlockRawGasUsed)
}

func TestSDMRepeatedSlotDedupAndRefundCapBehavior(gt *testing.T) {
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

		sameRefund := getOPGasRefund(t, sys.L2EL, receipts[1].TxHash)
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

		manyRefund := getOPGasRefund(t, sys.L2EL, receipts[3].TxHash)
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

// TestSDMMultiCategoryBatch submits transactions from multiple categories in a single burst,
// without calling .Eval() between submissions. This tests that different tx types
// (transfer, compute, events, state writes) can be batched into the same block.
func TestSDMMultiCategoryBatchSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSDMRethSystem(t, true)
	l := t.Logger()

	clientVersion := verifyOpReth(t, sys.L2EL)
	l.Info("Verified op-reth", "version", clientVersion)

	// Fund alice
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.ZeroWei)

	// Deploy contracts
	computeHeavyAddr := deployContract(t, alice, computeHeavyBin)
	stateBloatAddr := deployContract(t, alice, stateBloatBin)
	eventLoggerAddr := alice.DeployEventLogger()
	l.Info("Deployed contracts",
		"computeHeavy", computeHeavyAddr,
		"stateBloat", stateBloatAddr,
		"eventLogger", eventLoggerAddr)

	// Submit a diverse batch of transactions without waiting between them
	startNonce := alice.PendingNonce()
	type batchEntry struct {
		category string
		ptx      *txplan.PlannedTx
	}
	var batch []batchEntry

	categories := []struct {
		name string
		opts func(nonce uint64) []txplan.Option
	}{
		{
			name: "eoa_transfer",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(bob.Address())),
					txplan.WithValue(eth.OneHundredthEther),
				}
			},
		},
		{
			name: "compute_heavy",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(computeHeavyAddr)),
					txplan.WithData(encodeRun(200)),
					txplan.WithGasLimit(200_000),
				}
			},
		},
		{
			name: "event_emitter",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(eventLoggerAddr)),
					txplan.WithData(encodeEmitLog(3, 64)),
					txplan.WithGasLimit(200_000),
				}
			},
		},
		{
			name: "state_bloat",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(stateBloatAddr)),
					txplan.WithData(encodeRun(20)),
					txplan.WithGasLimit(500_000),
				}
			},
		},
		// Second round of same categories to trigger cross-tx warming
		{
			name: "compute_heavy_2",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(computeHeavyAddr)),
					txplan.WithData(encodeRun(200)),
					txplan.WithGasLimit(200_000),
				}
			},
		},
		{
			name: "event_emitter_2",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(eventLoggerAddr)),
					txplan.WithData(encodeEmitLog(3, 64)),
					txplan.WithGasLimit(200_000),
				}
			},
		},
		{
			name: "state_bloat_2",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(stateBloatAddr)),
					txplan.WithData(encodeRun(20)),
					txplan.WithGasLimit(500_000),
				}
			},
		},
		{
			name: "eoa_transfer_2",
			opts: func(nonce uint64) []txplan.Option {
				return []txplan.Option{
					txplan.WithTo(addrPtr(bob.Address())),
					txplan.WithValue(eth.OneHundredthEther),
				}
			},
		},
	}

	l.Info("Submitting batch", "txCount", len(categories), "startNonce", startNonce)

	for i, cat := range categories {
		nonce := startNonce + uint64(i)
		ptx := submitTxWithoutWait(t, alice, nonce, cat.opts(nonce)...)
		batch = append(batch, batchEntry{category: cat.name, ptx: ptx})
		l.Info("Submitted", "category", cat.name, "nonce", nonce)
	}

	// Wait for all to be included
	blockCounts := make(map[uint64]int)
	for i, entry := range batch {
		receipt, err := entry.ptx.Included.Eval(t.Ctx())
		t.Require().NoError(err, "tx %d (%s): failed to get receipt", i, entry.category)
		t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status,
			"tx %d (%s): must succeed", i, entry.category)

		blockNum := bigs.Uint64Strict(receipt.BlockNumber)
		blockCounts[blockNum]++

		refund := getOPGasRefund(t, sys.L2EL, receipt.TxHash)
		l.Info("Included",
			"category", entry.category,
			"block", blockNum,
			"txIdx", receipt.TransactionIndex,
			"gasUsed", receipt.GasUsed,
			"opGasRefund", refund)
	}

	// Report distribution
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

// addrPtr returns a pointer to the given address (helper for txplan.WithTo).
func addrPtr(addr common.Address) *common.Address {
	return &addr
}
