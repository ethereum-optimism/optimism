package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type StandardBridge struct {
	ctx      context.Context
	address  common.Address
	client   bind.ContractFilterer
	filterer *l1bridge.L1StandardBridgeFilterer
}

func (s *StandardBridge) Address() common.Address {
	return s.address
}

func (s *StandardBridge) GetDepositsByBlockRange(start, end uint64) (map[common.Hash][]db.Deposit, error) {
	depositsByBlockhash := make(map[common.Hash][]db.Deposit)

	var iter *l1bridge.L1StandardBridgeERC20DepositInitiatedIterator
	var err error
	const NUM_RETRIES = 5
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(s.ctx, DefaultConnectionTimeout)

		iter, err = s.filterer.FilterERC20DepositInitiated(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil, nil)
		if err != nil {
			logger.Error("Unable to query deposit events for block range ",
				"start", start, "end", end, "error", err)
			cancel()
			continue
		}
		cancel()
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
