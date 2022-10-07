package bridge

import (
	"context"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Portal struct {
	address  common.Address
	contract *bindings.OptimismPortal
}

func NewPortal(addr common.Address, portal *bindings.OptimismPortal) *Portal {
	return &Portal{
		address:  addr,
		contract: portal,
	}
}

func (p *Portal) Address() common.Address {
	return p.address
}

func (p *Portal) GetFinalizedWithdrawalsByBlockRange(ctx context.Context, start, end uint64) (FinalizedWithdrawalsMap, error) {
	wdsByBlockHash := make(FinalizedWithdrawalsMap)
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
		End:     &end,
	}

	var iter *bindings.OptimismPortalWithdrawalFinalizedIterator
	err := backoff.Do(3, backoff.Exponential(), func() error {
		var err error
		iter, err = p.contract.FilterWithdrawalFinalized(opts, nil)
		return err
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
