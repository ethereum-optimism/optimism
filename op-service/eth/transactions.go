package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type L1Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

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

// CheckRecentTxs checks the depth recent blocks for transactions from the account with address addr
// and returns the most recent block and true, if any was found, or the oldest block checked and false, if not.
func CheckRecentTxs(
	ctx context.Context,
	l1 L1Client,
	depth int,
	addr common.Address,
) (recentBlock uint64, found bool, err error) {
	blockHeader, err := l1.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve current block header: %w", err)
	}

	currentBlock := blockHeader.Number
	currentNonce, err := l1.NonceAt(ctx, addr, currentBlock)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve current nonce: %w", err)
	}

	oldestBlock := new(big.Int)
	oldestBlock.Sub(currentBlock, big.NewInt(int64(depth)))
	previousNonce, err := l1.NonceAt(ctx, addr, oldestBlock)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve previous nonce: %w", err)
	}

	if currentNonce == previousNonce {
		return oldestBlock.Uint64(), false, nil
	}

	// Decrease block num until we find the block before the most recent batcher tx was sent
	targetNonce := currentNonce - 1
	for currentNonce > targetNonce && currentBlock.Cmp(oldestBlock) != -1 {
		currentBlock.Sub(currentBlock, big.NewInt(1))
		currentNonce, err = l1.NonceAt(ctx, addr, currentBlock)
		if err != nil {
			return 0, false, fmt.Errorf("failed to retrieve nonce: %w", err)
		}
	}
	return currentBlock.Uint64() + 1, true, nil
}
