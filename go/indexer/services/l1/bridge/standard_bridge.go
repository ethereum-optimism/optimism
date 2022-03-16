package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type StandardBridge struct {
	name     string
	ctx      context.Context
	address  common.Address
	client   bind.ContractFilterer
	filterer *l1bridge.L1StandardBridgeFilterer
}

func (s *StandardBridge) Address() common.Address {
	return s.address
}

func (s *StandardBridge) GetDepositsByBlockRange(start, end uint64) (DepositsMap, error) {
	depositsByBlockhash := make(DepositsMap)

	iter, err := FilterERC20DepositInitiatedWithRetry(s.filterer, &bind.FilterOpts{
		Start:   start,
		End:     &end,
		Context: s.ctx,
	})
	if err != nil {
		logger.Error("Error fetching filter", "err", err)
	}

	for iter.Next() {
		depositsByBlockhash[iter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[iter.Event.Raw.BlockHash], db.Deposit{
				TxHash:      iter.Event.Raw.TxHash,
				L1Token:     iter.Event.L1Token,
				L2Token:     iter.Event.L2Token,
				FromAddress: iter.Event.From,
				ToAddress:   iter.Event.To,
				Amount:      iter.Event.Amount,
				Data:        iter.Event.Data,
				LogIndex:    iter.Event.Raw.Index,
			})
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return depositsByBlockhash, nil
}

func (s *StandardBridge) String() string {
	return s.name
}
