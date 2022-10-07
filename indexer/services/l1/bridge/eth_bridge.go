package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type EthBridge struct {
	name     string
	address  common.Address
	contract *bindings.L1StandardBridge
}

func (e *EthBridge) Address() common.Address {
	return e.address
}

func (e *EthBridge) GetDepositsByBlockRange(ctx context.Context, start, end uint64) (DepositsMap, error) {
	depositsByBlockhash := make(DepositsMap)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	var iter *bindings.L1StandardBridgeETHDepositInitiatedIterator
	err := backoff.Do(3, backoff.Exponential(), func() error {
		var err error
		iter, err = e.contract.FilterETHDepositInitiated(opts, nil, nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	defer iter.Close()
	for iter.Next() {
		depositsByBlockhash[iter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[iter.Event.Raw.BlockHash], db.Deposit{
				TxHash:      iter.Event.Raw.TxHash,
				FromAddress: iter.Event.From,
				ToAddress:   iter.Event.To,
				Amount:      iter.Event.Amount,
				Data:        iter.Event.ExtraData,
				LogIndex:    iter.Event.Raw.Index,
			})
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return depositsByBlockhash, iter.Error()
}

func (e *EthBridge) String() string {
	return e.name
}
