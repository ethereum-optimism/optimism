package extract

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
)

var _ Enricher = (*BondEnricher)(nil)

var ErrIncorrectCreditCount = errors.New("incorrect credit count")

type BondCaller interface {
	GetCredits(context.Context, rpcblock.Block, ...common.Address) ([]*big.Int, error)
}

type BondEnricher struct{}

func NewBondEnricher() *BondEnricher {
	return &BondEnricher{}
}

func (b *BondEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	recipients := make(map[common.Address]bool)
	for _, claim := range game.Claims {
		if claim.CounteredBy != (common.Address{}) {
			recipients[claim.CounteredBy] = true
		} else {
			recipients[claim.Claimant] = true
		}
	}
	recipientAddrs := maps.Keys(recipients)
	credits, err := caller.GetCredits(ctx, block, recipientAddrs...)
	if err != nil {
		return err
	}
	if len(credits) != len(recipients) {
		return fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectCreditCount, len(recipients), len(credits))
	}
	game.Credits = make(map[common.Address]*big.Int)
	for i, credit := range credits {
		game.Credits[recipientAddrs[i]] = credit
	}
	return nil
}
