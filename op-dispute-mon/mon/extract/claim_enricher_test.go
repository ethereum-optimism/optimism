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

func TestMaxValue(t *testing.T) {
	require.Equal(t, resolvedBondAmount.String(), "340282366920938463463374607431768211455")
}

func TestClaimEnricher(t *testing.T) {
	enricher := NewClaimEnricher()
	game := &types.EnrichedGameData{
		Claims: []types.EnrichedClaim{
			newClaimWithBond(resolvedBondAmount),
			newClaimWithBond(big.NewInt(0)),
			newClaimWithBond(big.NewInt(100)),
			newClaimWithBond(new(big.Int).Sub(resolvedBondAmount, big.NewInt(1))),
			newClaimWithBond(new(big.Int).Add(resolvedBondAmount, big.NewInt(1))),
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
