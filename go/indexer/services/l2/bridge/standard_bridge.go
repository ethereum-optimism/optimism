package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
	"github.com/ethereum-optimism/optimism/l2geth/common"
)

type StandardBridge struct {
	name     string
	ctx      context.Context
	address  common.Address
	client   bind.ContractFilterer
	filterer *l2bridge.L2StandardBridgeFilterer
}

func (s *StandardBridge) Address() common.Address {
	return s.address
}

func (s *StandardBridge) GetWithdrawalsByBlockRange(start, end uint64) (WithdrawalsMap, error) {
	withdrawalsByBlockhash := make(map[common.Hash][]db.Withdrawal)

	iter, err := FilterWithdrawalInitiatedWithRetry(s.filterer, &bind.FilterOpts{
		Start:   start,
		End:     &end,
		Context: s.ctx,
	})
	if err != nil {
		logger.Error("Error fetching filter", "err", err)
	}

	for iter.Next() {
		withdrawalsByBlockhash[iter.Event.Raw.BlockHash] = append(
			withdrawalsByBlockhash[iter.Event.Raw.BlockHash], db.Withdrawal{
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

	return withdrawalsByBlockhash, nil
}

func (s *StandardBridge) String() string {
	return s.name
}
