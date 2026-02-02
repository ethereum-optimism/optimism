package extract

import (
	"context"
	"errors"
	"testing"

	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestSuperNodeEndpointTracking verifies that all endpoint tracking fields are properly populated
func TestSuperNodeEndpointTracking(t *testing.T) {
	t.Run("TrackErrorsCorrectly", func(t *testing.T) {
		validator, clients, _ := setupMultiSuperNodeTest(t, 3)
		clients[0].outputErr = errors.New("error1")
		clients[1].outputErr = errors.New("error2")
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 100

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999, // Super root game type
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		// Verify error tracking
		require.Equal(t, 3, game.SuperNodeEndpointTotalCount, "Should track total endpoints")
		require.Equal(t, 2, game.SuperNodeEndpointErrorCount, "Should track 2 errors")
		require.Equal(t, 2, len(game.SuperNodeEndpointErrors), "Should track 2 unique endpoint errors")
		require.True(t, game.SuperNodeEndpointErrors["client-0"], "Should track client-0 error")
		require.True(t, game.SuperNodeEndpointErrors["client-1"], "Should track client-1 error")
	})

	t.Run("TrackNotFoundCount", func(t *testing.T) {
		validator, clients, _ := setupMultiSuperNodeTest(t, 3)
		clients[0].notFound = true
		clients[1].notFound = true
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 100

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 3, game.SuperNodeEndpointTotalCount)
		require.Equal(t, 2, game.SuperNodeEndpointNotFoundCount, "Should track 2 not found responses")
		require.Equal(t, 0, game.SuperNodeEndpointErrorCount, "Should have no errors")
	})

	t.Run("TrackSafeUnsafeCounts", func(t *testing.T) {
		validator, clients, _ := setupMultiSuperNodeTest(t, 4)
		// Two clients report safe (derivedFromL1BlockNum <= game.L1HeadNum)
		clients[0].superRoot = mockRootClaim
		clients[0].derivedFromL1BlockNum = 100 // Safe
		clients[1].superRoot = mockRootClaim
		clients[1].derivedFromL1BlockNum = 200 // Safe
		// Two clients report unsafe (derivedFromL1BlockNum > game.L1HeadNum)
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 201 // Unsafe
		clients[3].superRoot = mockRootClaim
		clients[3].derivedFromL1BlockNum = 300 // Unsafe

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		// This should result in disagreement due to mixed safety
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 4, game.SuperNodeEndpointTotalCount)
		require.Equal(t, 2, game.SuperNodeEndpointSafeCount, "Should track 2 safe assessments")
		require.Equal(t, 2, game.SuperNodeEndpointUnsafeCount, "Should track 2 unsafe assessments")
		require.True(t, game.HasMixedSuperSafety(), "Should detect mixed safety")
	})

	t.Run("TrackDivergentSuperRoots", func(t *testing.T) {
		validator, clients, _ := setupMultiSuperNodeTest(t, 3)
		divergedRoot := common.HexToHash("0xdivergent")
		clients[0].superRoot = mockRootClaim
		clients[0].derivedFromL1BlockNum = 100
		clients[1].superRoot = divergedRoot
		clients[1].derivedFromL1BlockNum = 100
		clients[2].superRoot = divergedRoot
		clients[2].derivedFromL1BlockNum = 100

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.True(t, game.SuperNodeEndpointDifferentSuperRoots, "Should flag divergent super roots")
		require.False(t, game.AgreeWithClaim, "Should disagree when super roots diverge")
	})

	t.Run("TrackMixedAvailability", func(t *testing.T) {
		validator, clients, _ := setupMultiSuperNodeTest(t, 3)
		clients[0].notFound = true
		clients[1].superRoot = mockRootClaim
		clients[1].derivedFromL1BlockNum = 100
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 100

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 3, game.SuperNodeEndpointTotalCount)
		require.Equal(t, 1, game.SuperNodeEndpointNotFoundCount)
		require.True(t, game.HasMixedSuperAvailability(), "Should detect mixed availability")
	})

	t.Run("AllFieldsZeroWhenNoEndpoints", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		validator := NewSuperAgreementEnricher(logger, &stubOutputMetrics{}, []SuperRootProvider{}, clock.NewDeterministicClock(time.Unix(9824924, 499)))

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:               200,
			L2SequenceNumber:        0,
			RootClaim:               mockRootClaim,
			SuperNodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrSuperNodeRpcRequired)

		// Verify all counts remain zero when no endpoints
		require.Equal(t, 0, game.SuperNodeEndpointTotalCount)
		require.Equal(t, 0, game.SuperNodeEndpointErrorCount)
		require.Equal(t, 0, game.SuperNodeEndpointNotFoundCount)
		require.Equal(t, 0, game.SuperNodeEndpointSafeCount)
		require.Equal(t, 0, game.SuperNodeEndpointUnsafeCount)
		require.False(t, game.SuperNodeEndpointDifferentSuperRoots)
	})
}
