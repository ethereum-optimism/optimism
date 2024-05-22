package gasprice

import (
	"context"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// SuggestOptimismPriorityFee returns a max priority fee value that can be used such that newly
// created transactions have a very high chance to be included in the following blocks, using a
// simplified and more predictable algorithm appropriate for chains like Optimism with a single
// known block builder.
//
// In the typical case, which results whenever the last block had room for more transactions, this
// function returns a minimum suggested priority fee value. Otherwise it returns the higher of this
// minimum suggestion or 10% over the median effective priority fee from the last block.
//
// Rationale: For a chain such as Optimism where there is a single block builder whose behavior is
// known, we know priority fee (as long as it is non-zero) has no impact on the probability for tx
// inclusion as long as there is capacity for it in the block. In this case then, there's no reason
// to return any value higher than some fixed minimum. Blocks typically reach capacity only under
// extreme events such as airdrops, meaning predicting whether the next block is going to be at
// capacity is difficult *except* in the case where we're already experiencing the increased demand
// from such an event. We therefore expect whether the last known block is at capacity to be one of
// the best predictors of whether the next block is likely to be at capacity. (An even better
// predictor is to look at the state of the transaction pool, but we want an algorithm that works
// even if the txpool is private or unavailable.)
//
// In the event the next block may be at capacity, the algorithm should allow for average fees to
// rise in order to reach a market price that appropriately reflects demand. We accomplish this by
// returning a suggestion that is a significant amount (10%) higher than the median effective
// priority fee from the previous block.
func (oracle *Oracle) SuggestOptimismPriorityFee(ctx context.Context, h *types.Header, headHash common.Hash) *big.Int {
	suggestion := new(big.Int).Set(oracle.minSuggestedPriorityFee)

	// find the maximum gas used by any of the transactions in the block to use as the capacity
	// margin
	receipts, err := oracle.backend.GetReceipts(ctx, headHash)
	if receipts == nil || err != nil {
		log.Error("failed to get block receipts", "err", err)
		return suggestion
	}
	var maxTxGasUsed uint64
	for i := range receipts {
		gu := receipts[i].GasUsed
		if gu > maxTxGasUsed {
			maxTxGasUsed = gu
		}
	}

	// sanity check the max gas used value
	if maxTxGasUsed > h.GasLimit {
		log.Error("found tx consuming more gas than the block limit", "gas", maxTxGasUsed)
		return suggestion
	}

	if h.GasUsed+maxTxGasUsed > h.GasLimit {
		// A block is "at capacity" if, when it is built, there is a pending tx in the txpool that
		// could not be included because the block's gas limit would be exceeded. Since we don't
		// have access to the txpool, we instead adopt the following heuristic: consider a block as
		// at capacity if the total gas consumed by its transactions is within max-tx-gas-used of
		// the block limit, where max-tx-gas-used is the most gas used by any one transaction
		// within the block. This heuristic is almost perfectly accurate when transactions always
		// consume the same amount of gas, but becomes less accurate as tx gas consumption begins
		// to vary. The typical error is we assume a block is at capacity when it was not because
		// max-tx-gas-used will in most cases over-estimate the "capacity margin". But it's better
		// to err on the side of returning a higher-than-needed suggestion than a lower-than-needed
		// one in order to satisfy our desire for high chance of inclusion and rising fees under
		// high demand.
		block, err := oracle.backend.BlockByNumber(ctx, rpc.BlockNumber(h.Number.Int64()))
		if block == nil || err != nil {
			log.Error("failed to get last block", "err", err)
			return suggestion
		}
		baseFee := block.BaseFee()
		txs := block.Transactions()
		if len(txs) == 0 {
			log.Error("block was at capacity but doesn't have transactions")
			return suggestion
		}
		tips := bigIntArray(make([]*big.Int, len(txs)))
		for i := range txs {
			tips[i] = txs[i].EffectiveGasTipValue(baseFee)
		}
		sort.Sort(tips)
		median := tips[len(tips)/2]
		newSuggestion := new(big.Int).Add(median, new(big.Int).Div(median, big.NewInt(10)))
		// use the new suggestion only if it's bigger than the minimum
		if newSuggestion.Cmp(suggestion) > 0 {
			suggestion = newSuggestion
		}
	}

	// the suggestion should be capped by oracle.maxPrice
	if suggestion.Cmp(oracle.maxPrice) > 0 {
		suggestion.Set(oracle.maxPrice)
	}

	oracle.cacheLock.Lock()
	oracle.lastHead = headHash
	oracle.lastPrice = suggestion
	oracle.cacheLock.Unlock()

	return new(big.Int).Set(suggestion)
}
