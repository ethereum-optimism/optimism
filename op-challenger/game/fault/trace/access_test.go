package trace

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestAccessor_UsesSelector(t *testing.T) {
	ctx := context.Background()
	depth := types.Depth(4)
	provider1 := test.NewAlphabetWithProofProvider(t, big.NewInt(0), depth, nil)
	provider2 := alphabet.NewTraceProvider(big.NewInt(0), depth)
	claim := types.Claim{}
	game := types.NewGameState([]types.Claim{claim}, depth)
	pos1 := types.NewPositionFromGIndex(big.NewInt(4))
	pos2 := types.NewPositionFromGIndex(big.NewInt(6))

	accessor := &Accessor{
		selector: func(ctx context.Context, actualGame types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
			require.Equal(t, game, actualGame)
			require.Equal(t, claim, ref)

			if pos == pos1 {
				return provider1, nil
			} else if pos == pos2 {
				return provider2, nil
			}
			return nil, fmt.Errorf("incorrect position requested: %v", pos)
		},
	}

	t.Run("Get", func(t *testing.T) {
		actual, err := accessor.Get(ctx, game, claim, pos1)
		require.NoError(t, err)
		expected, err := provider1.Get(ctx, pos1)
		require.NoError(t, err)
		require.Equal(t, expected, actual)

		actual, err = accessor.Get(ctx, game, claim, pos2)
		require.NoError(t, err)
		expected, err = provider2.Get(ctx, pos2)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("GetStepData", func(t *testing.T) {
		actualPrestate, actualProofData, actualPreimageData, err := accessor.GetStepData(ctx, game, claim, pos1)
		require.NoError(t, err)
		expectedPrestate, expectedProofData, expectedPreimageData, err := provider1.GetStepData(ctx, pos1)
		require.NoError(t, err)
		require.Equal(t, expectedPrestate, actualPrestate)
		require.Equal(t, expectedProofData, actualProofData)
		require.Equal(t, expectedPreimageData, actualPreimageData)

		actualPrestate, actualProofData, actualPreimageData, err = accessor.GetStepData(ctx, game, claim, pos2)
		require.NoError(t, err)
		expectedPrestate, expectedProofData, expectedPreimageData, err = provider2.GetStepData(ctx, pos2)
		require.NoError(t, err)
		require.Equal(t, expectedPrestate, actualPrestate)
		require.Equal(t, expectedProofData, actualProofData)
		require.Equal(t, expectedPreimageData, actualPreimageData)
	})

	t.Run("GetL2BlockNumberChallenge", func(t *testing.T) {
		provider := &ChallengeTraceProvider{
			TraceProvider: provider1,
		}
		accessor := &Accessor{
			selector: func(ctx context.Context, actualGame types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
				require.Equal(t, game, actualGame)
				require.Equal(t, game.Claims()[0], ref)
				require.Equal(t, types.RootPosition, pos)
				return provider, nil
			},
		}
		challenge, err := accessor.GetL2BlockNumberChallenge(ctx, game)
		require.NoError(t, err)
		require.NotNil(t, challenge)
		require.Equal(t, eth.Bytes32{0xaa, 0xbb}, challenge.Output.OutputRoot)
	})
}

type ChallengeTraceProvider struct {
	types.TraceProvider
}

func (c *ChallengeTraceProvider) GetL2BlockNumberChallenge(_ context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	return &types.InvalidL2BlockNumberChallenge{
		Output: &eth.OutputResponse{OutputRoot: eth.Bytes32{0xaa, 0xbb}},
	}, nil
}
