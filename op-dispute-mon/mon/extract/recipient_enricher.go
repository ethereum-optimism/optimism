package extract

import (
	"context"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

var _ Enricher = (*RecipientEnricher)(nil)

type RecipientEnricher struct{}

func NewRecipientEnricher() *RecipientEnricher {
	return &RecipientEnricher{}
}

func (w *RecipientEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.EnrichedGameData) error {
	recipients := make(map[common.Address]bool)
	for _, claim := range game.Claims {
		if claim.CounteredBy != (common.Address{}) {
			recipients[claim.CounteredBy] = true
		} else {
			recipients[claim.Claimant] = true
		}
	}
	if game.BlockNumberChallenger != (common.Address{}) {
		recipients[game.BlockNumberChallenger] = true
	}
	game.Recipients = recipients
	return nil
}
