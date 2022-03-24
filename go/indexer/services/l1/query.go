package l1

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1erc20"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/scc"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l1/bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func QueryERC20(address common.Address, client *ethclient.Client) (*db.Token, error) {
	contract, err := l1erc20.NewL1ERC20(address, client)
	if err != nil {
		return nil, err
	}

	name, err := contract.Name(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	symbol, err := contract.Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	decimals, err := contract.Decimals(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	return &db.Token{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

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
