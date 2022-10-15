package l1

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/bindings/legacy/scc"
	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services/l1/bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func QueryStateBatches(filterer *scc.StateCommitmentChainFilterer, startHeight, endHeight uint64, ctx context.Context) (map[common.Hash][]db.StateBatch, error) {
	batches := make(map[common.Hash][]db.StateBatch)

	iter, err := bridge.FilterStateBatchAppendedWithRetry(ctx, filterer, &bind.FilterOpts{
		Start: startHeight,
		End:   &endHeight,
	})
	if err != nil {
		return nil, err
	}

	defer iter.Close()
	for iter.Next() {
		batches[iter.Event.Raw.BlockHash] = append(
			batches[iter.Event.Raw.BlockHash], db.StateBatch{
				Index:     iter.Event.BatchIndex,
				Root:      iter.Event.BatchRoot,
				Size:      iter.Event.BatchSize,
				PrevTotal: iter.Event.PrevTotalElements,
				ExtraData: iter.Event.ExtraData,
				BlockHash: iter.Event.Raw.BlockHash,
			})
	}
	return batches, iter.Error()
}
