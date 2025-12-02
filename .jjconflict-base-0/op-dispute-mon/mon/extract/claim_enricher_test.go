package extract

import (
	"context"
	"errors"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/stretchr/testify/require"
)

func TestClaimEnricher(t *testing.T) {
	caller := &mockGameCaller{resolved: make(map[int]bool)}
	enricher := NewClaimEnricher()
	expected := []bool{true, false, false, false, false}
	game := &types.EnrichedGameData{
		Claims: claimsWithResolvedSubgames(caller, expected...),
	}
	err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
	require.NoError(t, err)
	for i, claim := range game.Claims {
		require.Equal(t, expected[i], claim.Resolved)
	}
}

func TestClaimEnricherError(t *testing.T) {
	expectedErr := errors.New("boom")
	caller := &mockGameCaller{resolved: make(map[int]bool), resolvedErr: expectedErr}
	enricher := NewClaimEnricher()
	game := &types.EnrichedGameData{
		Claims: claimsWithResolvedSubgames(caller, true, false),
	}
	err := enricher.Enrich(context.Background(), rpcblock.Latest, caller, game)
	require.ErrorIs(t, err, expectedErr)
}

func claimsWithResolvedSubgames(caller *mockGameCaller, resolved ...bool) []types.EnrichedClaim {
	claims := make([]types.EnrichedClaim, len(resolved))
	for i, r := range resolved {
		claims[i] = types.EnrichedClaim{Claim: faultTypes.Claim{ContractIndex: i}}
		caller.resolved[i] = r
	}
	return claims
}
