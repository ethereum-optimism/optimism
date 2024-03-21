package extract

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
)

var _ Enricher = (*ClaimEnricher)(nil)

type ClaimEnricher struct{}

func NewClaimEnricher() *ClaimEnricher {
	return &ClaimEnricher{}
}

var resolvedBondAmount = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

func (e *ClaimEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *types.EnrichedGameData) error {
	for i, claim := range game.Claims {
		if claim.Bond.Cmp(resolvedBondAmount) == 0 {
			game.Claims[i].Resolved = true
		}
	}
	return nil
}
