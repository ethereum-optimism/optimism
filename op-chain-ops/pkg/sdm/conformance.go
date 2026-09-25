package sdm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// ConformanceResult contains the canonical SDM validation result and the raw
// receipts used by the additional Lagoon conformance checks.
type ConformanceResult struct {
	*ValidationResult
	Receipts map[common.Hash]*RPCReceipt `json:"receipts"`
}

// ValidatePostExecConformance validates the consensus shape, RPC encoding,
// receipt fields, DA-footprint accounting, and replay accounting of an SDM
// block. These checks are policy-independent and apply to every SDM producer.
func ValidatePostExecConformance(ctx context.Context, rpcClient Caller, blockNum uint64) (*ConformanceResult, error) {
	validation, err := ValidatePostExecBlock(ctx, rpcClient, blockNum, DefaultValidationOptions())
	if err != nil {
		return nil, err
	}
	if err := validateUniquePostExec(validation.Block, validation.PostExecIndex); err != nil {
		return nil, err
	}
	if err := validatePayloadEntries(validation); err != nil {
		return nil, err
	}
	if err := validatePostExecRPC(ctx, rpcClient, validation.PostExecTx); err != nil {
		return nil, err
	}

	receipts := make(map[common.Hash]*RPCReceipt, len(validation.Block.Transactions))
	for i, tx := range validation.Block.Transactions {
		receipt, err := GetRPCReceipt(ctx, rpcClient, tx.Hash)
		if err != nil {
			return nil, err
		}
		if receipt.TxHash != tx.Hash {
			return nil, fmt.Errorf("receipt transaction hash %s does not match block transaction %s", receipt.TxHash, tx.Hash)
		}
		if receipt.BlockHash != validation.Block.Hash {
			return nil, fmt.Errorf("receipt %s block hash %s, want %s", tx.Hash, receipt.BlockHash, validation.Block.Hash)
		}
		if uint64(receipt.TransactionIndex) != uint64(i) {
			return nil, fmt.Errorf("receipt %s transaction index %d, want %d", tx.Hash, receipt.TransactionIndex, i)
		}
		receipts[tx.Hash] = receipt
	}
	if err := validateReceiptConformance(validation, receipts); err != nil {
		return nil, err
	}
	if err := validateReplayAccounting(validation, receipts); err != nil {
		return nil, err
	}
	return &ConformanceResult{ValidationResult: validation, Receipts: receipts}, nil
}

// ValidateVerifierAgreement checks that an independent verifier serves the
// same canonical block, transaction ordering, and opGasRefund receipt fields.
func ValidateVerifierAgreement(ctx context.Context, producer *ConformanceResult, verifier Caller) error {
	blockNum := uint64(producer.Block.Number)
	block, err := GetBlockWithTxs(ctx, verifier, blockNum)
	if err != nil {
		return fmt.Errorf("fetch verifier block %d: %w", blockNum, err)
	}
	if block.Hash != producer.Block.Hash {
		return fmt.Errorf("verifier block %d hash %s, want producer hash %s", blockNum, block.Hash, producer.Block.Hash)
	}
	if block.GasUsed != producer.Block.GasUsed {
		return fmt.Errorf("verifier block %d gasUsed %d, want %d", blockNum, block.GasUsed, producer.Block.GasUsed)
	}
	if len(block.Transactions) != len(producer.Block.Transactions) {
		return fmt.Errorf("verifier block %d has %d transactions, want %d", blockNum, len(block.Transactions), len(producer.Block.Transactions))
	}
	for i, tx := range block.Transactions {
		producerTx := producer.Block.Transactions[i]
		if tx.Hash != producerTx.Hash {
			return fmt.Errorf("verifier transaction %d hash %s, want %s", i, tx.Hash, producerTx.Hash)
		}
		verifierReceipt, err := GetRPCReceipt(ctx, verifier, tx.Hash)
		if err != nil {
			return fmt.Errorf("fetch verifier receipt %s: %w", tx.Hash, err)
		}
		producerReceipt := producer.Receipts[tx.Hash]
		if !equalUint64Ptr(verifierReceipt.OPGasRefund, producerReceipt.OPGasRefund) {
			return fmt.Errorf("verifier receipt %s opGasRefund %s, want %s", tx.Hash, formatUint64Ptr(verifierReceipt.OPGasRefund), formatUint64Ptr(producerReceipt.OPGasRefund))
		}
	}
	return nil
}

// GetRPCReceipt fetches a receipt without relying on typed transaction decoding.
func GetRPCReceipt(ctx context.Context, rpcClient Caller, txHash common.Hash) (*RPCReceipt, error) {
	var raw json.RawMessage
	if err := rpcClient.CallContext(ctx, &raw, "eth_getTransactionReceipt", txHash); err != nil {
		return nil, fmt.Errorf("eth_getTransactionReceipt(%s): %w", txHash, err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("receipt %s not found", txHash)
	}
	var receipt RPCReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("unmarshal receipt %s: %w", txHash, err)
	}
	return &receipt, nil
}

func validateUniquePostExec(block *RPCBlock, expectedIndex int) error {
	count := 0
	for i, tx := range block.Transactions {
		if uint64(tx.Type) != optypes.PostExecTxType {
			continue
		}
		count++
		if i != expectedIndex {
			return fmt.Errorf("block %d contains an unexpected post-exec transaction at index %d", block.Number, i)
		}
	}
	if count != 1 {
		return fmt.Errorf("block %d contains %d post-exec transactions, want exactly one", block.Number, count)
	}
	return nil
}

func validatePayloadEntries(validation *ValidationResult) error {
	var previous uint64
	seen := make(map[uint64]struct{}, len(validation.Payload.GasRefundEntries))
	for i, entry := range validation.Payload.GasRefundEntries {
		if _, ok := seen[entry.Index]; ok {
			return fmt.Errorf("post-exec payload contains duplicate transaction index %d", entry.Index)
		}
		if i > 0 && entry.Index <= previous {
			return fmt.Errorf("post-exec payload indexes are not strictly increasing: %d follows %d", entry.Index, previous)
		}
		seen[entry.Index] = struct{}{}
		previous = entry.Index
	}
	return nil
}

type rpcTransactionDetails struct {
	RPCTransaction
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Gas      hexutil.Uint64  `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
}

func validatePostExecRPC(ctx context.Context, rpcClient Caller, postExec *RPCTransaction) error {
	wantHash := types.NewTx(&types.PostExecTx{Data: []byte(postExec.Input)}).Hash()
	if postExec.Hash != wantHash {
		return fmt.Errorf("post-exec transaction hash %s, want canonical hash %s", postExec.Hash, wantHash)
	}

	var raw json.RawMessage
	if err := rpcClient.CallContext(ctx, &raw, "eth_getTransactionByHash", wantHash); err != nil {
		return fmt.Errorf("eth_getTransactionByHash(%s): %w", wantHash, err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("post-exec transaction %s is not resolvable by hash", wantHash)
	}
	var tx rpcTransactionDetails
	if err := json.Unmarshal(raw, &tx); err != nil {
		return fmt.Errorf("unmarshal post-exec transaction %s: %w", wantHash, err)
	}
	if tx.Hash != wantHash || uint64(tx.Type) != optypes.PostExecTxType {
		return fmt.Errorf("resolved post-exec transaction identity mismatch: hash=%s type=0x%x", tx.Hash, tx.Type)
	}
	var rawTx hexutil.Bytes
	if err := rpcClient.CallContext(ctx, &rawTx, "eth_getRawTransactionByHash", wantHash); err != nil {
		return fmt.Errorf("eth_getRawTransactionByHash(%s): %w", wantHash, err)
	}
	if len(rawTx) < 2 || rawTx[0] != byte(optypes.PostExecTxType) || !bytes.Equal(rawTx[1:], postExec.Input) {
		return fmt.Errorf("post-exec transaction %s has non-canonical raw EIP-2718 bytes", wantHash)
	}
	if tx.From != (common.Address{}) {
		return fmt.Errorf("post-exec transaction from is %s, want zero address", tx.From)
	}
	if tx.To != nil && *tx.To != (common.Address{}) {
		return fmt.Errorf("post-exec transaction to is %s, want zero address", *tx.To)
	}
	if tx.Nonce != nil && uint64(*tx.Nonce) != 0 {
		return fmt.Errorf("post-exec transaction nonce is %d, want 0", *tx.Nonce)
	}
	if uint64(tx.Gas) != 0 {
		return fmt.Errorf("post-exec transaction gas is %d, want 0", tx.Gas)
	}
	if !isZeroBig(tx.Value) {
		return fmt.Errorf("post-exec transaction value is %s, want 0", formatBig(tx.Value))
	}
	// Legacy op-reth releases populate gasPrice from the containing block's base fee even though
	// Lagoon specifies that the inapplicable field is omitted. Keep the live-network checker
	// compatible with those releases; op-alloy's serialization regression test separately pins the
	// canonical omission for newly built clients.
	return nil
}

func validateReceiptConformance(validation *ValidationResult, receipts map[common.Hash]*RPCReceipt) error {
	postExecReceipt := receipts[validation.PostExecTx.Hash]
	if postExecReceipt == nil {
		return fmt.Errorf("post-exec receipt %s missing", validation.PostExecTx.Hash)
	}
	if uint64(postExecReceipt.Type) != optypes.PostExecTxType {
		return fmt.Errorf("post-exec receipt type is 0x%x, want 0x%x", postExecReceipt.Type, optypes.PostExecTxType)
	}
	if uint64(postExecReceipt.Status) != types.ReceiptStatusSuccessful {
		return fmt.Errorf("post-exec receipt status is %d, want success", postExecReceipt.Status)
	}
	if uint64(postExecReceipt.GasUsed) != 0 {
		return fmt.Errorf("post-exec receipt gasUsed is %d, want 0", postExecReceipt.GasUsed)
	}
	if !isExplicitZeroBig(postExecReceipt.EffectiveGasPrice) {
		return fmt.Errorf("post-exec receipt effectiveGasPrice is %s, want explicit zero", formatBig(postExecReceipt.EffectiveGasPrice))
	}
	if postExecReceipt.BlobGasUsed == nil || uint64(*postExecReceipt.BlobGasUsed) != 0 {
		return fmt.Errorf("post-exec receipt blobGasUsed is %s, want 0", formatUint64Ptr(postExecReceipt.BlobGasUsed))
	}
	if postExecReceipt.OPGasRefund != nil {
		return fmt.Errorf("post-exec receipt unexpectedly exposes opGasRefund=%d", *postExecReceipt.OPGasRefund)
	}
	if !isExplicitZeroBig(postExecReceipt.L1Fee) {
		return fmt.Errorf("post-exec receipt l1Fee is %s, want explicit zero", formatBig(postExecReceipt.L1Fee))
	}
	if !isExplicitZeroBig(postExecReceipt.L1GasUsed) {
		return fmt.Errorf("post-exec receipt l1GasUsed is %s, want explicit zero", formatBig(postExecReceipt.L1GasUsed))
	}
	if postExecReceipt.L1FeeScalar != nil {
		return fmt.Errorf("post-exec receipt unexpectedly exposes removed l1FeeScalar=%s", formatBig(postExecReceipt.L1FeeScalar))
	}

	var regular *RPCReceipt
	var daFootprint uint64
	for _, tx := range validation.Block.Transactions {
		receipt := receipts[tx.Hash]
		if receipt == nil {
			return fmt.Errorf("receipt %s missing", tx.Hash)
		}
		if uint64(tx.Type) == types.DepositTxType {
			if receipt.OPGasRefund != nil {
				return fmt.Errorf("deposit receipt %s unexpectedly exposes opGasRefund=%d", tx.Hash, *receipt.OPGasRefund)
			}
			continue
		}
		if receipt.BlobGasUsed == nil {
			return fmt.Errorf("receipt %s omits blobGasUsed", tx.Hash)
		}
		if math.MaxUint64-daFootprint < uint64(*receipt.BlobGasUsed) {
			return fmt.Errorf("receipt blobGasUsed total overflows uint64")
		}
		daFootprint += uint64(*receipt.BlobGasUsed)
		if uint64(tx.Type) != optypes.PostExecTxType && regular == nil {
			regular = receipt
		}
	}
	if validation.Block.BlobGasUsed == nil {
		return fmt.Errorf("block %d omits blobGasUsed", validation.Block.Number)
	}
	if daFootprint != uint64(*validation.Block.BlobGasUsed) {
		return fmt.Errorf("receipt blobGasUsed total %d, want block blobGasUsed %d", daFootprint, *validation.Block.BlobGasUsed)
	}
	if regular == nil {
		return fmt.Errorf("block %d has no regular receipt for block-scoped L1 field comparison", validation.Block.Number)
	}
	for _, field := range []struct {
		name string
		got  *hexutil.Big
		want *hexutil.Big
	}{
		{"l1GasPrice", postExecReceipt.L1GasPrice, regular.L1GasPrice},
		{"l1BaseFeeScalar", postExecReceipt.L1BaseFeeScalar, regular.L1BaseFeeScalar},
		{"l1BlobBaseFee", postExecReceipt.L1BlobBaseFee, regular.L1BlobBaseFee},
		{"l1BlobBaseFeeScalar", postExecReceipt.L1BlobBaseFeeScalar, regular.L1BlobBaseFeeScalar},
		{"operatorFeeScalar", postExecReceipt.OperatorFeeScalar, regular.OperatorFeeScalar},
		{"operatorFeeConstant", postExecReceipt.OperatorFeeConstant, regular.OperatorFeeConstant},
		{"daFootprintGasScalar", postExecReceipt.DAFootprintGasScalar, regular.DAFootprintGasScalar},
	} {
		if !equalBigPtr(field.got, field.want) {
			return fmt.Errorf("post-exec receipt %s is %s, want block-scoped value %s", field.name, formatBig(field.got), formatBig(field.want))
		}
	}
	return nil
}

func validateReplayAccounting(validation *ValidationResult, receipts map[common.Hash]*RPCReceipt) error {
	replay := validation.Replay
	if replay == nil {
		return fmt.Errorf("block %d conformance requires debug_replaySDMBlock", validation.Block.Number)
	}
	if replay.BlockNum != uint64(validation.Block.Number) || replay.BlockHash != validation.Block.Hash {
		return fmt.Errorf("replay block identity is %d/%s, want %d/%s", replay.BlockNum, replay.BlockHash, validation.Block.Number, validation.Block.Hash)
	}
	if replay.EmbeddedPayload == nil || !reflect.DeepEqual(*replay.EmbeddedPayload, *validation.Payload) {
		return fmt.Errorf("replay embedded payload does not match canonical post-exec payload")
	}
	if replay.Summary.MismatchCount != 0 {
		return fmt.Errorf("replay summary reports %d mismatches", replay.Summary.MismatchCount)
	}
	if replay.Summary.BlockGasUsed != uint64(validation.Block.GasUsed) {
		return fmt.Errorf("replay block gas used %d, want header gasUsed %d", replay.Summary.BlockGasUsed, validation.Block.GasUsed)
	}
	if math.MaxUint64-replay.Summary.BlockGasUsed < validation.TotalPayloadRefund {
		return fmt.Errorf("canonical gas plus payload refund overflows uint64")
	}
	wantRaw := replay.Summary.BlockGasUsed + validation.TotalPayloadRefund
	if replay.Summary.BlockRawGasUsed != wantRaw {
		return fmt.Errorf("replay raw block gas %d, want canonical gas plus refunds %d", replay.Summary.BlockRawGasUsed, wantRaw)
	}

	expectedRows := len(validation.Block.Transactions) - 1
	if len(replay.Txs) != expectedRows || replay.Summary.TxCountTotal != expectedRows {
		return fmt.Errorf("replay contains %d rows and reports %d total transactions, want %d", len(replay.Txs), replay.Summary.TxCountTotal, expectedRows)
	}
	rows := make(map[uint64]ReplaySDMTx, len(replay.Txs))
	for _, row := range replay.Txs {
		if _, ok := rows[row.TxIndex]; ok {
			return fmt.Errorf("replay contains duplicate transaction index %d", row.TxIndex)
		}
		rows[row.TxIndex] = row
	}
	for i, tx := range validation.Block.Transactions {
		if uint64(tx.Type) == optypes.PostExecTxType {
			continue
		}
		row, ok := rows[uint64(i)]
		if !ok {
			return fmt.Errorf("replay omits transaction index %d", i)
		}
		if row.TxHash != tx.Hash {
			return fmt.Errorf("replay transaction %d hash %s, want %s", i, row.TxHash, tx.Hash)
		}
		receipt := receipts[tx.Hash]
		if row.CanonicalGasUsed != uint64(receipt.GasUsed) {
			return fmt.Errorf("replay transaction %d canonical gas %d, want receipt gasUsed %d", i, row.CanonicalGasUsed, receipt.GasUsed)
		}
		if uint64(tx.Type) == types.DepositTxType {
			if row.OPGasRefundPayload != nil || row.RawGasUsed != row.CanonicalGasUsed {
				return fmt.Errorf("deposit replay row %d contains a refund or changes gas accounting", i)
			}
			continue
		}
		refund := uint64(0)
		if row.OPGasRefundPayload != nil {
			refund = *row.OPGasRefundPayload
		}
		if math.MaxUint64-row.CanonicalGasUsed < refund || row.RawGasUsed != row.CanonicalGasUsed+refund {
			return fmt.Errorf("replay transaction %d raw gas %d, want canonical %d plus refund %d", i, row.RawGasUsed, row.CanonicalGasUsed, refund)
		}
	}
	return nil
}

func equalUint64Ptr(a, b *hexutil.Uint64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func formatUint64Ptr(v *hexutil.Uint64) string {
	if v == nil {
		return "<absent>"
	}
	return fmt.Sprintf("%d", *v)
}

func isZeroBig(v *hexutil.Big) bool {
	return v == nil || (*big.Int)(v).Sign() == 0
}

func isExplicitZeroBig(v *hexutil.Big) bool {
	return v != nil && (*big.Int)(v).Sign() == 0
}

func equalBigPtr(a, b *hexutil.Big) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return (*big.Int)(a).Cmp((*big.Int)(b)) == 0
}

func formatBig(v *hexutil.Big) string {
	if v == nil {
		return "<absent>"
	}
	return (*big.Int)(v).String()
}
