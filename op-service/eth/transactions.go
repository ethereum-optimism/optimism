package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// EncodeTransactions encodes a list of transactions into opaque transactions.
func EncodeTransactions(elems []*types.Transaction) ([]hexutil.Bytes, error) {
	out := make([]hexutil.Bytes, len(elems))
	for i, el := range elems {
		dat, err := el.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tx %d: %w", i, err)
		}
		out[i] = dat
	}
	return out, nil
}

// DecodeTransactions decodes a list of opaque transactions into transactions.
func DecodeTransactions(data []hexutil.Bytes) ([]*types.Transaction, error) {
	dest := make([]*types.Transaction, len(data))
	for i := range dest {
		var x types.Transaction
		if err := x.UnmarshalBinary(data[i]); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tx %d: %w", i, err)
		}
		dest[i] = &x
	}
	return dest, nil
}

// TransactionsToHashes computes the transaction-hash for every transaction in the input.
func TransactionsToHashes(elems []*types.Transaction) []common.Hash {
	out := make([]common.Hash, len(elems))
	for i, el := range elems {
		out[i] = el.Hash()
	}
	return out
}
