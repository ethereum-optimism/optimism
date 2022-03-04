package l1

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/scc"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l1/bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func QueryStateBatches(filterer *scc.StateCommitmentChainFilterer, startHeight, endHeight uint64, ctx context.Context) (map[common.Hash][]db.StateBatch, error) {
	batches := make(map[common.Hash][]db.StateBatch)

	iter, err := bridge.FilterStateBatchAppendedWithRetry(filterer, &bind.FilterOpts{
		Start:   startHeight,
		End:     &endHeight,
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

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
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return batches, nil
}
