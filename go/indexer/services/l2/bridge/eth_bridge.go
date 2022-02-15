package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
	"github.com/ethereum-optimism/optimism/l2geth/common"
)

type EthBridge struct {
	name     string
	ctx      context.Context
	address  common.Address
	client   bind.ContractFilterer
	filterer *l2bridge.L2StandardBridgeFilterer
}

func (e *EthBridge) Address() common.Address {
	return e.address
}

func (e *EthBridge) GetWithdrawalsByBlockRange(start, end uint64) (map[common.Hash][]db.Withdrawal, error) {
	withdrawalsByBlockhash := make(map[common.Hash][]db.Withdrawal)

	var iter *l2bridge.L2StandardBridgeWithdrawalInitiatedIterator
	var err error
	const NUM_RETRIES = 5
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(e.ctx, DefaultConnectionTimeout)

		iter, err = e.filterer.FilterWithdrawalInitiated(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil, nil)
		if err != nil {
			logger.Error("Unable to query withdrawal events for block range ",
				"start", start, "end", end, "error", err)
			cancel()
			continue
		}
		cancel()
	}

	for iter.Next() {
		withdrawalsByBlockhash[iter.Event.Raw.BlockHash] = append(
			withdrawalsByBlockhash[iter.Event.Raw.BlockHash], db.Withdrawal{
				TxHash:      iter.Event.Raw.TxHash,
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

func (e *EthBridge) String() string {
	return e.name
}
