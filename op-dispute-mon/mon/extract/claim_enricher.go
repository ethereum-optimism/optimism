package extract

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
)

var _ Enricher = (*ClaimEnricher)(nil)

type ClaimEnricher struct{}

func NewClaimEnricher() *ClaimEnricher {
	return &ClaimEnricher{}
}

func (e *ClaimEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *types.EnrichedGameData) error {
	for i, claim := range game.Claims {
		if claim.Bond.Cmp(types.ResolvedBondAmount) == 0 {
			game.Claims[i].Resolved = true
		}
	}
	return nil
}
