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
	GetRequiredBonds(context.Context, rpcblock.Block, ...*big.Int) ([]*big.Int, error)
}

type BondEnricher struct{}

func NewBondEnricher() *BondEnricher {
	return &BondEnricher{}
}

func (b *BondEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if err := b.enrichCredits(ctx, block, caller, game); err != nil {
		return err
	}
	if err := b.enrichRequiredBonds(ctx, block, caller, game); err != nil {
		return err
	}
	return nil
}

func (b *BondEnricher) enrichCredits(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
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

func (b *BondEnricher) enrichRequiredBonds(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	positions := make([]*big.Int, len(game.Claims))
	for _, claim := range game.Claims {
		// If the claim is not resolved, we don't need to get the bond
		// for it since the Bond field in the claim will be accurate.
		if !claim.Resolved {
			continue
		}
		positions = append(positions, claim.Position.ToGIndex())
	}
	bonds, err := caller.GetRequiredBonds(ctx, block, positions...)
	if err != nil {
		return err
	}
	if len(bonds) != len(positions) {
		return fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectCreditCount, len(positions), len(bonds))
	}
	game.RequiredBonds = make(map[int]*big.Int)
	bondIndex := 0
	for i, claim := range game.Claims {
		if !claim.Resolved {
			game.RequiredBonds[i] = claim.Bond
			continue
		}
		game.RequiredBonds[i] = bonds[bondIndex]
		bondIndex++
	}
	return nil
}
