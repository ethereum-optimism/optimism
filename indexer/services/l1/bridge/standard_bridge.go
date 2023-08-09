package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type StandardBridge struct {
	name     string
	address  common.Address
	contract *bindings.L1StandardBridge
}

func (s *StandardBridge) Address() common.Address {
	return s.address
}

func (s *StandardBridge) GetDepositsByBlockRange(ctx context.Context, start, end uint64) (DepositsMap, error) {
	depositsByBlockhash := make(DepositsMap)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	iter, err := backoff.Do(ctx, 3, backoff.Exponential(), func() (*bindings.L1StandardBridgeERC20DepositInitiatedIterator, error) {
		return s.contract.FilterERC20DepositInitiated(opts, nil, nil, nil)
	})
	if err != nil {
		return nil, err
	}

	defer iter.Close()
	for iter.Next() {
		depositsByBlockhash[iter.Event.Raw.BlockHash] = append(
			depositsByBlockhash[iter.Event.Raw.BlockHash], db.Deposit{
				TxHash:      iter.Event.Raw.TxHash,
				L1Token:     iter.Event.L1Token,
				L2Token:     iter.Event.L2Token,
				FromAddress: iter.Event.From,
				ToAddress:   iter.Event.To,
				Amount:      iter.Event.Amount,
				Data:        iter.Event.ExtraData,
				LogIndex:    iter.Event.Raw.Index,
			})
	}

	return depositsByBlockhash, iter.Error()
}

func (s *StandardBridge) String() string {
	return s.name
}
