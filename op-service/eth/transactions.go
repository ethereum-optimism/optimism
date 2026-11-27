package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type L1Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

// checkRecentTxsAttempts bounds how often CheckRecentTxs restarts its search after observing a
// reorg.
const checkRecentTxsAttempts = 3

var errNonceReorg = errors.New("nonce inconsistent with the head nonce, likely due to a reorg")

// CheckRecentTxs checks the depth recent blocks for txs from the account with address addr
// and returns either:
//   - blockNum containing the last tx and true if any was found
//   - the oldest block checked and false if no nonce change was found
func CheckRecentTxs(
	ctx context.Context,
	l1 L1Client,
	depth int,
	addr common.Address,
) (blockNum uint64, found bool, err error) {
	for range checkRecentTxsAttempts {
		blockNum, found, err = checkRecentTxs(ctx, l1, depth, addr)
		if !errors.Is(err, errNonceReorg) {
			return blockNum, found, err
		}
	}
	return 0, false, err
}

func checkRecentTxs(
	ctx context.Context,
	l1 L1Client,
	depth int,
	addr common.Address,
) (blockNum uint64, found bool, err error) {
	blockHeader, err := l1.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve current block header: %w", err)
	}
	if err := CheckHeader(blockHeader); err != nil {
		return 0, false, fmt.Errorf("invalid current block header: %w", err)
	}

	currentBlock := bigs.Uint64Strict(blockHeader.Number)
	currentNonce, err := l1.NonceAt(ctx, addr, new(big.Int).SetUint64(currentBlock))
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve current nonce: %w", err)
	}

	var oldestBlock uint64
	if d := uint64(depth); currentBlock > d {
		oldestBlock = currentBlock - d
	}
	previousNonce, err := l1.NonceAt(ctx, addr, new(big.Int).SetUint64(oldestBlock))
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve previous nonce: %w", err)
	}

	if currentNonce == previousNonce {
		// Most recent tx is older than the given depth
		return oldestBlock, false, nil
	}

	// Use binary search to find the block where the nonce changed
	low := oldestBlock
	high := currentBlock

	for low < high {
		mid := low + (high-low)/2
		midNonce, err := l1.NonceAt(ctx, addr, new(big.Int).SetUint64(mid))
		if err != nil {
			return 0, false, fmt.Errorf("failed to retrieve nonce at block %d: %w", mid, err)
		}

		if midNonce > currentNonce {
			return 0, false, fmt.Errorf("nonce %d at block %d above head nonce %d: %w", midNonce, mid, currentNonce, errNonceReorg)
		} else if midNonce == currentNonce {
			high = mid
		} else {
			// midNonce < currentNonce: check the next block to see if we've found the
			// spot where the nonce transitions to the currentNonce
			nextBlockNum := mid + 1
			nextBlockNonce, err := l1.NonceAt(ctx, addr, new(big.Int).SetUint64(nextBlockNum))
			if err != nil {
				return 0, false, fmt.Errorf("failed to retrieve nonce at block %d: %w", mid, err)
			}

			if nextBlockNonce == currentNonce {
				return nextBlockNum, true, nil
			}
			low = mid + 1
		}
	}
	return oldestBlock, false, nil
}
