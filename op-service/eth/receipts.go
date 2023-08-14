package eth

import (
	"fmt"
	"math/big"

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
