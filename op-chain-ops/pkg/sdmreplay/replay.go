package sdmreplay

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Source is the data source used by replay logic.
type Source interface {
	BlockNumberSource
	ClientVersion(ctx context.Context) (string, error)
	ChainID(ctx context.Context) (uint64, error)
	GetBlockByNumber(ctx context.Context, blockNum uint64) (*RPCBlock, error)
	GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*RPCReceipt, error)
	ReplaySdmBlock(ctx context.Context, blockNum uint64, comparePayload bool, compareReceipts bool) (*ReplaySdmBlock, error)
}

// ReplayMode controls where per-tx refund accounting comes from.
type ReplayMode string

const (
	ReplayModeCounterfactualEnabled ReplayMode = "counterfactual_enabled"
)

// ReplayRange processes blocks sequentially and aggregates the requested output.
func ReplayRange(ctx context.Context, src Source, cfg Config) (*RangeResult, error) {
	if cfg.Workers == 0 {
		cfg.Workers = 1
	}
	if cfg.Workers != 1 {
		return nil, fmt.Errorf("--workers=%d is not supported yet; use 1", cfg.Workers)
	}
	if cfg.Format == "" {
		cfg.Format = "jsonl"
	}
	if cfg.Format != "jsonl" {
		return nil, fmt.Errorf("unsupported format %q", cfg.Format)
	}
	if cfg.FromBlock > cfg.ToBlock {
		return nil, fmt.Errorf("from block %d is greater than to block %d", cfg.FromBlock, cfg.ToBlock)
	}
	if cfg.IncludeTrace {
		return nil, fmt.Errorf("--include-trace is not supported with debug_replaySDMBlock")
	}

	headBlock, err := src.GetBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("get head block: %w", err)
	}
	chainID, err := src.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}
	clientVersion, err := src.ClientVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get client version: %w", err)
	}

	result := &RangeResult{
		RunConfig: RunConfigRecord{
			Type:               "run_config",
			RPC:                cfg.RPCURL,
			FromBlock:          cfg.FromBlockSelector,
			ToBlock:            cfg.ToBlockSelector,
			ResolvedFromBlock:  cfg.FromBlock,
			ResolvedToBlock:    cfg.ToBlock,
			HeadBlock:          headBlock,
			ChainID:            chainID,
			ClientVersion:      clientVersion,
			ReplayMode:         string(ReplayModeCounterfactualEnabled),
			ComparePayload:     cfg.ComparePayload,
			CompareRPCReceipts: cfg.CompareRPCReceipts,
			SummaryOnly:        cfg.SummaryOnly,
			IncludeTrace:       false,
		},
		Summary: SummaryRecord{
			Type:       "summary",
			FromBlock:  cfg.FromBlock,
			ToBlock:    cfg.ToBlock,
			ReplayMode: string(ReplayModeCounterfactualEnabled),
		},
	}

	var totalRefundRatio float64

	for blockNum := cfg.FromBlock; blockNum <= cfg.ToBlock; blockNum++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		blockResult, err := replayBlock(ctx, src, blockNum, cfg)
		if err != nil {
			return nil, err
		}
		if cfg.SkipEmptyBlocks && blockResult.Block.TxCountUser == 0 {
			result.Summary.BlocksSkipped++
			continue
		}

		result.Blocks = append(result.Blocks, *blockResult)
		result.RunConfig.ReplayMode = blockResult.Block.ReplayMode
		result.Summary.ReplayMode = blockResult.Block.ReplayMode
		result.Summary.BlocksProcessed++
		if blockResult.Block.SDMTxPresent {
			result.Summary.BlocksWithSDMTx++
		}
		result.Summary.TxCountTotal += blockResult.Block.TxCountTotal
		result.Summary.TxCountUser += blockResult.Block.TxCountUser
		result.Summary.TotalGasUsed += blockResult.Block.BlockGasUsed
		result.Summary.ReplayRefundTotal += blockResult.Block.ReplayRefundTotal
		result.Summary.NodeReceiptRefundTotal += blockResult.Block.NodeReceiptRefundTotal
		result.Summary.PayloadRefundTotal += blockResult.Block.PayloadRefundTotal
		result.Summary.EffectiveGasTotal += blockResult.Block.BlockEffectiveGas
		result.Summary.MismatchCount += blockResult.Block.MismatchCount
		totalRefundRatio += blockResult.Block.AvgRefundRatio * float64(blockResult.Block.TxCountUser)
	}

	if result.Summary.TotalGasUsed > 0 {
		result.Summary.TotalRefundRatio = float64(result.Summary.ReplayRefundTotal) / float64(result.Summary.TotalGasUsed)
	}
	if result.Summary.TxCountUser > 0 {
		result.Summary.AvgRefundRatio = totalRefundRatio / float64(result.Summary.TxCountUser)
	}

	if cfg.FailOnMismatch && result.Summary.MismatchCount > 0 {
		return result, fmt.Errorf("found %d mismatch record(s)", result.Summary.MismatchCount)
	}
	return result, nil
}

func replayBlock(ctx context.Context, src Source, blockNum uint64, cfg Config) (*BlockResult, error) {
	replay, err := src.ReplaySdmBlock(ctx, blockNum, cfg.ComparePayload, cfg.CompareRPCReceipts)
	if err != nil {
		if strings.Contains(err.Error(), "method not found") {
			return nil, fmt.Errorf("node does not expose debug_replaySDMBlock; run against the modified op-reth node: %w", err)
		}
		return nil, err
	}

	block, err := src.GetBlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, fmt.Errorf("load block %d: %w", blockNum, err)
	}

	blockTxs := make(map[uint64]RPCTransaction, len(block.Transactions))
	for idx, tx := range block.Transactions {
		blockTxs[uint64(idx)] = tx
	}

	blockRecord := BlockRecord{
		Type:                   "block",
		BlockNum:               replay.Summary.BlockNum,
		BlockHash:              replay.Summary.BlockHash.Hex(),
		ParentHash:             replay.ParentHash.Hex(),
		TxCountTotal:           replay.Summary.TxCountTotal,
		TxCountUser:            replay.Summary.TxCountUser,
		SDMTxPresent:           replay.Summary.SDMTxPresent,
		SDMPayloadEntryCount:   replay.Summary.SDMPayloadEntryCount,
		BlockGasUsed:           replay.Summary.BlockGasUsed,
		BlockOPGasRefund:       replay.Summary.ReplayRefundTotal,
		BlockEffectiveGas:      replay.Summary.BlockEffectiveGas,
		NodeReceiptRefundTotal: replay.Summary.NodeReceiptRefundTotal,
		ReplayRefundTotal:      replay.Summary.ReplayRefundTotal,
		PayloadRefundTotal:     replay.Summary.PayloadRefundTotal,
		MismatchCount:          replay.Summary.MismatchCount,
		ReplayMode:             replay.Summary.ReplayMode,
	}
	if blockRecord.BlockGasUsed > 0 {
		blockRecord.BlockRefundRatio = float64(blockRecord.ReplayRefundTotal) / float64(blockRecord.BlockGasUsed)
	}

	mismatches := make([]MismatchRecord, 0, len(replay.Mismatches))
	for _, mismatch := range replay.Mismatches {
		record := MismatchRecord{
			Type:     "mismatch",
			BlockNum: mismatch.BlockNum,
			Category: mismatch.Category,
			Message:  mismatch.Message,
		}
		if mismatch.TxIndex != nil {
			record.TxIndex = int(*mismatch.TxIndex)
			if tx, ok := blockTxs[*mismatch.TxIndex]; ok {
				record.TxHash = tx.Hash.Hex()
			}
		}
		if mismatch.Expected != nil {
			record.Expected = *mismatch.Expected
		}
		if mismatch.Actual != nil {
			record.Actual = *mismatch.Actual
		}
		mismatches = append(mismatches, record)
	}

	txRecords := make([]TxRecord, 0, len(replay.Txs))
	var totalRatio float64

	for _, tx := range replay.Txs {
		var (
			from string
			to   string
		)
		if blockTx, ok := blockTxs[tx.TxIndex]; ok {
			from = blockTx.From.Hex()
			if blockTx.To != nil {
				to = blockTx.To.Hex()
			}
		}

		payloadRefund := uint64Value(tx.OPGasRefundPayload)
		receiptRefund := uint64Value(tx.OPGasRefundReceipt)
		refundRatio := 0.0
		if !tx.IsDepositTx && tx.GasUsed > 0 {
			refundRatio = float64(tx.OPGasRefundReplay) / float64(tx.GasUsed)
			totalRatio += refundRatio
		}

		if cfg.SummaryOnly {
			continue
		}

		receipt, err := src.GetTransactionReceipt(ctx, tx.TxHash)
		if err != nil {
			return nil, fmt.Errorf("load receipt for tx %s in block %d: %w", tx.TxHash.Hex(), blockNum, err)
		}

		txRecord := TxRecord{
			Type:               "tx",
			BlockNum:           replay.BlockNum,
			TxIndex:            int(tx.TxIndex),
			ReplayTxIndex:      int(tx.ReplayTxIndex),
			TxHash:             tx.TxHash.Hex(),
			TxType:             fmt.Sprintf("0x%x", tx.TxType),
			From:               from,
			To:                 to,
			GasUsed:            tx.GasUsed,
			OPGasRefundReplay:  tx.OPGasRefundReplay,
			OPGasRefundReceipt: receiptRefund,
			OPGasRefundPayload: payloadRefund,
			EffectiveGas:       tx.EffectiveGas,
			RefundRatio:        refundRatio,
			Status:             uint64(receipt.Status),
			IsSDMTx:            false,
			IsDepositTx:        tx.IsDepositTx,
			Mismatch:           tx.Mismatch,
			AccountingSource:   "debug_replaySDMBlock",
		}
		txRecords = append(txRecords, txRecord)
	}

	if blockRecord.TxCountUser > 0 {
		blockRecord.AvgRefundRatio = totalRatio / float64(blockRecord.TxCountUser)
	}

	return &BlockResult{
		Block:      blockRecord,
		Txs:        txRecords,
		Mismatches: mismatches,
	}, nil
}

func uint64Value(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}
