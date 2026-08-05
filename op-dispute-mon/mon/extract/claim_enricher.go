package extract

import (
	"context"
	"fmt"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
)

var _ FaultEnricher = (*ClaimEnricher)(nil)

type ClaimCaller interface {
	IsResolved(ctx context.Context, block rpcblock.Block, claim ...faultTypes.Claim) ([]bool, error)
}

type ClaimEnricher struct{}

func NewClaimEnricher() *ClaimEnricher {
	return &ClaimEnricher{}
}

func (e *ClaimEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller FaultGameCaller, game *types.FaultGameData) error {
	claims := make([]faultTypes.Claim, 0, len(game.Claims))
	for _, claim := range game.Claims {
		claims = append(claims, claim.Claim)
	}
	resolved, err := caller.IsResolved(ctx, block, claims...)
	if err != nil {
		return fmt.Errorf("failed to retrieve resolved status: %w", err)
	}
	for i := range game.Claims {
		game.Claims[i].Resolved = resolved[i]
	}
	return nil
}
