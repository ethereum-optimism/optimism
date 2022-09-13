package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type EthBridge struct {
	name     string
	ctx      context.Context
	address  common.Address
	client   bind.ContractFilterer
	filterer *bindings.L1StandardBridgeFilterer
}

func (e *EthBridge) Address() common.Address {
	return e.address
}

func (e *EthBridge) GetDepositsByBlockRange(start, end uint64) (DepositsMap, error) {
	depositsByBlockhash := make(DepositsMap)

	iter, err := FilterETHDepositInitiatedWithRetry(e.ctx, e.filterer, &bind.FilterOpts{
		Start: start,
		End:   &end,
	})
	if err != nil {
		logger.Error("Error fetching filter", "err", err)
	}

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

	return depositsByBlockhash, nil
}

func (s *EthBridge) GetWithdrawalsByBlockRange(start, end uint64) (WithdrawalsMap, error) {
	withdrawalsByBlockHash := make(WithdrawalsMap)

	iter, err := FilterETHWithdrawalFinalizedWithRetry(s.ctx, s.filterer, &bind.FilterOpts{
		Start: start,
		End:   &end,
	})

	if err != nil {
		logger.Error("Error fetching filter", "err", err)
	}

	for iter.Next() {
		withdrawalsByBlockHash[iter.Event.Raw.BlockHash] = append(
			withdrawalsByBlockHash[iter.Event.Raw.BlockHash], db.Withdrawal{
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

	return withdrawalsByBlockHash, nil
}

func (e *EthBridge) String() string {
	return e.name
}
