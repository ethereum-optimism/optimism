package eth

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// EncodeReceipts encodes a list of receipts into raw receipts. Some non-consensus meta-data may be lost.
func EncodeReceipts(elems []*types.Receipt) ([]hexutil.Bytes, error) {
	out := make([]hexutil.Bytes, len(elems))
	for i, el := range elems {
		dat, err := el.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal receipt %d: %w", i, err)
		}
		out[i] = dat
	}
	return out, nil
}

// DecodeRawReceipts decodes receipts and adds additional blocks metadata.
// The contract-deployment addresses are not set however (high cost, depends on nonce values, unused by op-node).
func DecodeRawReceipts(block BlockID, rawReceipts []hexutil.Bytes, txHashes []common.Hash) ([]*types.Receipt, error) {
	result := make([]*types.Receipt, len(rawReceipts))
	totalIndex := uint(0)
	prevCumulativeGasUsed := uint64(0)
	for i, r := range rawReceipts {
		var x types.Receipt
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
	n := len(receipts)
	receiptIndex := sort.Search(len(receipts), func(i int) bool {

		receipt := receipts[i]
		if receipt != nil && len(receipt.Logs) > 0 {
			lastLogIdx := receipt.Logs[len(receipt.Logs)-1].Index
			return logIndex <= lastLogIdx
		}

		// Walk through the receipts until we find one that has logs
		for diff := 1; ; diff++ {
			foundAnyDirection := false

			// Check Left
			idxLeft := i - diff
			if idxLeft >= 0 {
				foundAnyDirection = true
				receipt = receipts[idxLeft]
				if receipt != nil && len(receipt.Logs) > 0 {
					return logIndex <= receipt.Logs[len(receipt.Logs)-1].Index
				}
			}

			// Check Right
			idxRight := i + diff
			if idxRight < n {
				foundAnyDirection = true
				receipt = receipts[idxRight]
				if receipt != nil && len(receipt.Logs) > 0 {
					return logIndex < receipt.Logs[0].Index
				}
			}

			if !foundAnyDirection {
				// Both left and right checks went out of bounds in the same iteration
				break
			}
		}

		return false
	})

	if receiptIndex >= len(receipts) {
		return nil, fmt.Errorf("Log index %d not found in block", logIndex)
	}

	// We found the correct receipt, now find the log within that receipt
	receipt := receipts[receiptIndex]
	if receipt == nil || len(receipt.Logs) == 0 {
		// This should be impossible if BinarySearchFunc found=true, due to the final check
		return nil, fmt.Errorf("internal error: found receipt index %d but receipt is empty/nil", receiptIndex)
	}

	// Calculate the index relative to the start of the logs in this receipt
	relativeIndex := logIndex - receipt.Logs[0].Index
	if relativeIndex >= uint(len(receipt.Logs)) {
		// This should also be impossible if the cmp function is correct
		return nil, fmt.Errorf("internal error: log index %d out of bounds for receipt %d (len %d, start %d)", logIndex, receiptIndex, len(receipt.Logs), receipt.Logs[0].Index)
	}
	return receipt.Logs[relativeIndex], nil
}
