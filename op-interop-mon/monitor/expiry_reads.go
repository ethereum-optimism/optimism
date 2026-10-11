package monitor

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// maxLogRange bounds the blocks of one eth_getLogs query; longer ranges are split.
const maxLogRange = 10_000

// ExpiryClient reads one L2 chain for the expiry, refund and liquidity checks. Every check decides
// from logs and storage at a block, never from the success of a contract call.
type ExpiryClient interface {
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, optypes.Receipts, error)
	GetStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockTag string) (common.Hash, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

// ethExpiryClient adds eth_getLogs to an EthClient.
type ethExpiryClient struct {
	*sources.EthClient
}

var _ ExpiryClient = ethExpiryClient{}

// NewExpiryClient wraps an EthClient as an ExpiryClient.
func NewExpiryClient(c *sources.EthClient) ExpiryClient {
	return ethExpiryClient{EthClient: c}
}

func (c ethExpiryClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	if q.FromBlock == nil || q.ToBlock == nil || q.BlockHash != nil {
		return nil, errors.New("expiry log queries must name a block range")
	}
	arg := map[string]any{
		"fromBlock": hexutil.EncodeBig(q.FromBlock),
		"toBlock":   hexutil.EncodeBig(q.ToBlock),
		"topics":    q.Topics,
	}
	if len(q.Addresses) > 0 {
		arg["address"] = q.Addresses
	}
	var logs []types.Log
	if err := c.RPC().CallContext(ctx, &logs, "eth_getLogs", arg); err != nil {
		return nil, err
	}
	return logs, nil
}

// storageAt reads a storage word at a block number.
func storageAt(ctx context.Context, c ExpiryClient, addr common.Address, slot common.Hash, block uint64) (common.Hash, error) {
	return c.GetStorageAt(ctx, addr, slot, hexutil.EncodeUint64(block))
}

// filterLogsChunked runs a log query over [from, to], at most maxLogRange blocks per request.
func filterLogsChunked(ctx context.Context, c ExpiryClient, q ethereum.FilterQuery, from, to uint64) ([]types.Log, error) {
	var out []types.Log
	for start := from; start <= to; start += maxLogRange {
		end := min(start+maxLogRange-1, to)
		q.FromBlock = new(big.Int).SetUint64(start)
		q.ToBlock = new(big.Int).SetUint64(end)
		logs, err := c.FilterLogs(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("get logs in [%d, %d]: %w", start, end, err)
		}
		out = append(out, logs...)
		if end == to {
			break
		}
	}
	return out, nil
}

// blockAtOrBefore returns the last block, at or below `head`, whose timestamp is at most `ts`. OP
// chains have a fixed block time, so it first guesses the block from head's block time and checks
// the guess, and falls back to a binary search.
func blockAtOrBefore(ctx context.Context, c ExpiryClient, ts uint64, head eth.BlockInfo) (eth.BlockInfo, error) {
	if head.Time() <= ts {
		return head, nil
	}
	if head.NumberU64() == 0 {
		return nil, fmt.Errorf("no block at or before timestamp %d", ts)
	}
	parent, err := c.InfoByNumber(ctx, head.NumberU64()-1)
	if err != nil {
		return nil, err
	}
	if blockTime := head.Time() - parent.Time(); blockTime > 0 {
		back := (head.Time() - ts + blockTime - 1) / blockTime
		if back <= head.NumberU64() {
			guess, err := c.InfoByNumber(ctx, head.NumberU64()-back)
			if err != nil {
				return nil, err
			}
			if guess.Time() <= ts {
				next, err := c.InfoByNumber(ctx, guess.NumberU64()+1)
				if err != nil {
					return nil, err
				}
				if next.Time() > ts {
					return guess, nil
				}
			}
		}
	}
	// Binary search for the last block with time <= ts in [0, head).
	lo, hi := uint64(0), head.NumberU64()
	var found eth.BlockInfo
	for lo < hi {
		mid := lo + (hi-lo)/2
		info, err := c.InfoByNumber(ctx, mid)
		if err != nil {
			return nil, err
		}
		if info.Time() <= ts {
			found = info
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no block at or before timestamp %d", ts)
	}
	return found, nil
}
