package extract

import (
	"context"
	"math/big"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/stretchr/testify/require"
)

func TestClaimEnricher(t *testing.T) {
	enricher := NewClaimEnricher()
	game := &types.EnrichedGameData{
		Claims: []types.EnrichedClaim{
			newClaimWithBond(types.ResolvedBondAmount),
			newClaimWithBond(big.NewInt(0)),
			newClaimWithBond(big.NewInt(100)),
			newClaimWithBond(new(big.Int).Sub(types.ResolvedBondAmount, big.NewInt(1))),
			newClaimWithBond(new(big.Int).Add(types.ResolvedBondAmount, big.NewInt(1))),
		},
	}
	caller := &mockGameCaller{}
	err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
	require.NoError(t, err)
	expected := []bool{true, false, false, false, false}
	for i, claim := range game.Claims {
		require.Equal(t, expected[i], claim.Resolved)
	}
}

func newClaimWithBond(bond *big.Int) types.EnrichedClaim {
	return types.EnrichedClaim{Claim: faultTypes.Claim{ClaimData: faultTypes.ClaimData{Bond: bond}}}
}
