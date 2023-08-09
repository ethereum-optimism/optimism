package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Portal struct {
	address  common.Address
	contract *bindings.OptimismPortal
}

func NewPortal(addrs services.AddressManager) *Portal {
	address, contract := addrs.OptimismPortal()

	return &Portal{
		address:  address,
		contract: contract,
	}
}

func (p *Portal) Address() common.Address {
	return p.address
}

func (p *Portal) GetProvenWithdrawalsByBlockRange(ctx context.Context, start, end uint64) (ProvenWithdrawalsMap, error) {
	wdsByBlockHash := make(ProvenWithdrawalsMap)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	iter, err := backoff.Do(ctx, 3, backoff.Exponential(), func() (*bindings.OptimismPortalWithdrawalProvenIterator, error) {
		return p.contract.FilterWithdrawalProven(opts, nil, nil, nil)
	})
	if err != nil {
		return nil, err
	}

	defer iter.Close()
	for iter.Next() {
		wdsByBlockHash[iter.Event.Raw.BlockHash] = append(
			wdsByBlockHash[iter.Event.Raw.BlockHash], db.ProvenWithdrawal{
				WithdrawalHash: iter.Event.WithdrawalHash,
				From:           iter.Event.From,
				To:             iter.Event.To,
				TxHash:         iter.Event.Raw.TxHash,
				LogIndex:       iter.Event.Raw.Index,
			},
		)
	}

	return wdsByBlockHash, iter.Error()
}

func (p *Portal) GetFinalizedWithdrawalsByBlockRange(ctx context.Context, start, end uint64) (FinalizedWithdrawalsMap, error) {
	wdsByBlockHash := make(FinalizedWithdrawalsMap)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	iter, err := backoff.Do(ctx, 3, backoff.Exponential(), func() (*bindings.OptimismPortalWithdrawalFinalizedIterator, error) {
		return p.contract.FilterWithdrawalFinalized(opts, nil)
	})
	if err != nil {
		return nil, err
	}

	defer iter.Close()
	for iter.Next() {
		wdsByBlockHash[iter.Event.Raw.BlockHash] = append(
			wdsByBlockHash[iter.Event.Raw.BlockHash], db.FinalizedWithdrawal{
				TxHash:         iter.Event.Raw.TxHash,
				WithdrawalHash: iter.Event.WithdrawalHash,
				Success:        iter.Event.Success,
				LogIndex:       iter.Event.Raw.Index,
			},
		)
	}

	return wdsByBlockHash, iter.Error()
}
