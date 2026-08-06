package eth

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// DecodeRawReceipts decodes receipts and adds additional blocks metadata.
// The contract-deployment addresses are not set however (high cost, depends on nonce values, unused by op-node).
// The OP JSON-only fee fields (L1GasPrice, OperatorFeeScalar, ...) are not part
// of the consensus encoding and remain nil; the consensus-encoded deposit
// receipt fields (DepositNonce, DepositReceiptVersion) are decoded.
func DecodeRawReceipts(block BlockID, rawReceipts []hexutil.Bytes, txHashes []common.Hash) (optypes.Receipts, error) {
	result := make(optypes.Receipts, len(rawReceipts))
	totalIndex := uint(0)
	prevCumulativeGasUsed := uint64(0)
	for i, r := range rawReceipts {
		var x optypes.Receipt
		if err := x.UnmarshalBinary(r); err != nil {
			return nil, fmt.Errorf("failed to decode receipt %d: %w", i, err)
		}
		x.TxHash = txHashes[i]
		x.BlockHash = block.Hash
		x.BlockNumber = new(big.Int).SetUint64(block.Number)
		x.TransactionIndex = uint(i)
		x.GasUsed = x.CumulativeGasUsed - prevCumulativeGasUsed
		// contract address meta-data is not computed.
		prevCumulativeGasUsed = x.CumulativeGasUsed
		for _, l := range x.Logs {
			l.BlockNumber = block.Number
			l.TxHash = x.TxHash
			l.TxIndex = uint(i)
			l.BlockHash = block.Hash
			l.Index = totalIndex
			totalIndex += 1
		}
		result[i] = &x
	}
	return result, nil
}

// Assumes receipts are sorted by transaction index.
func GetLogAtIndex(receipts []*types.Receipt, logIndex uint) (*types.Log, error) {
	// Find the receipt that might contain our log
	for _, rec := range receipts {
		if len(rec.Logs) > 0 {
			firstLogIndex := rec.Logs[0].Index
			lastLogIndex := rec.Logs[len(rec.Logs)-1].Index
			if firstLogIndex <= logIndex && logIndex <= lastLogIndex {
				return rec.Logs[logIndex-firstLogIndex], nil
			}
		}
	}
	return nil, fmt.Errorf("internal error: log index %d not found in receipts", logIndex)
}
