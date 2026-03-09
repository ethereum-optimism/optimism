package extract

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_CheckSuperRootAgreement(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWhenNoSuperNodeClient", func(t *testing.T) {
		validator, _, _ := setupSuperValidatorTest(t)
		validator.clients = nil // Set to nil to test the error case
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrSuperNodeRpcRequired)
	})

	t.Run("SkipOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{0, 1, 2, 3, 6, 254, 255, 1337}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupSuperValidatorTest(t)
				validator.clients = nil // Should not error even though there's no super node client
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:          200,
					L2SequenceNumber:   0,
					RootClaim:          mockRootClaim,
					NodeEndpointErrors: make(map[string]bool),
				}
				err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
				require.NoError(t, err)
				require.Zero(t, metrics.fetchTime)
			})
		}
	})

	t.Run("FetchAllNonOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{4, 5, 7, 9, 11, 49812} // Treat unknown game types as using super roots
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupSuperValidatorTest(t)
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:          200,
					L2SequenceNumber:   0,
					RootClaim:          mockRootClaim,
					NodeEndpointErrors: make(map[string]bool),
				}
				err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
				require.NoError(t, err)
				require.NotZero(t, metrics.fetchTime, "should have fetched output root")
			})
		}
	})

	t.Run("OutputFetchFails", func(t *testing.T) {
		validator, rollup, metrics := setupSuperValidatorTest(t)
		rollup.outputErr = errors.New("boom")
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrAllSuperNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_Safe", func(t *testing.T) {
		validator, _, metrics := setupSuperValidatorTest(t)
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   0,
			RootClaim:          common.Hash{},
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_Safe_DerivedFromGameHead", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		client.derivedFromL1BlockNum = 200
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_Safe_DerivedFromBeforeGameHead", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		client.derivedFromL1BlockNum = 199
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_NotSafe", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		client.derivedFromL1BlockNum = 101
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   0,
			RootClaim:          common.Hash{},
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_NotSafe", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		client.derivedFromL1BlockNum = 201
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   100,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputNotFound", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		client.notFound = true
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   42984924,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllSuperNodesReturnError", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		for _, client := range clients {
			client.outputErr = errors.New("boom")
		}
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAllSuperNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllSuperNodesReturnNotFound", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		for _, client := range clients {
			client.notFound = true
		}
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          100,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("SomeSuperNodesOutOfSync", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		clients[0].notFound = true
		clients[1].outputErr = nil
		clients[2].outputErr = nil
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("SuperNodesDiverged", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		divergedRoot := common.HexToHash("0x5678")
		clients[0].superRoot = mockRootClaim
		clients[1].superRoot = divergedRoot
		clients[2].superRoot = divergedRoot
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllSuperNodesAgree", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		clients[0].derivedFromL1BlockNum = 200
		clients[1].derivedFromL1BlockNum = 199
		clients[2].derivedFromL1BlockNum = 201
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesMatchClaimAndSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 4)
		clients[0].notFound = true
		clients[1].notFound = true
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 100 // Safe because L1HeadNum is 200
		clients[3].superRoot = mockRootClaim
		clients[3].derivedFromL1BlockNum = 150 // Safe because L1HeadNum is 200
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   50,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim) // Should disagree due to mixed responses (divergence)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesDontMatchClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)
		differentRoot := common.HexToHash("0x9999")
		clients[0].notFound = true
		clients[1].superRoot = differentRoot
		clients[1].derivedFromL1BlockNum = 100
		clients[2].superRoot = differentRoot
		clients[2].derivedFromL1BlockNum = 150
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   50,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim) // Should disagree due to mixed responses (divergence)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_SuperRootMatchesClaim_NoneReportSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)

		for _, client := range clients {
			client.superRoot = mockRootClaim
			client.derivedFromL1BlockNum = 250 // Not safe because L1HeadNum is 200
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   50,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim, "Should set ExpectedRootClaim to empty hash when not safe")
		require.False(t, game.AgreeWithClaim, "Should disagree because none report it as safe")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_SuperRootDifferentFromClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiSuperNodeTest(t, 3)

		differentRoot := common.HexToHash("0xdifferent")
		for _, client := range clients {
			client.superRoot = differentRoot
			client.derivedFromL1BlockNum = 100 // Safe because L1HeadNum is 200
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   50,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim, "Should disagree because super root differs from claim")
		require.NotZero(t, metrics.fetchTime)
	})
}

func setupSuperValidatorTest(t *testing.T) (*SuperAgreementEnricher, *stubSuperNodeClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSuperNodeClient{derivedFromL1BlockNum: 0, superRoot: mockRootClaim}
	metrics := &stubOutputMetrics{}
	validator := NewSuperAgreementEnricher(logger, metrics, []SuperRootProvider{client}, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
}

func setupMultiSuperNodeTest(t *testing.T, numNodes int) (*SuperAgreementEnricher, []*stubSuperNodeClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	clients := make([]*stubSuperNodeClient, numNodes)
	superNodeClients := make([]SuperRootProvider, numNodes)
	for i := range clients {
		clients[i] = &stubSuperNodeClient{
			derivedFromL1BlockNum: 0,
			superRoot:             mockRootClaim,
		}
		superNodeClients[i] = clients[i]
	}
	metrics := &stubOutputMetrics{}
	validator := NewSuperAgreementEnricher(logger, metrics, superNodeClients, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, clients, metrics
}

type stubSuperNodeClient struct {
	requestedTimestamp    uint64
	outputErr             error
	notFound              bool
	derivedFromL1BlockNum uint64
	superRoot             common.Hash
}

func (s *stubSuperNodeClient) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	s.requestedTimestamp = uint64(timestamp)
	if s.outputErr != nil {
		return eth.SuperRootAtTimestampResponse{}, s.outputErr
	}
	if s.notFound {
		return eth.SuperRootAtTimestampResponse{}, nil
	}
	return eth.SuperRootAtTimestampResponse{
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: eth.BlockID{Number: s.derivedFromL1BlockNum},
			Super:              eth.NewSuperV1(timestamp),
			SuperRoot:          eth.Bytes32(s.superRoot),
		},
	}, nil
}

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
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		// Verify error tracking
		require.Equal(t, 3, game.NodeEndpointTotalCount, "Should track total endpoints")
		require.Equal(t, 2, game.NodeEndpointErrorCount, "Should track 2 errors")
		require.Equal(t, 2, len(game.NodeEndpointErrors), "Should track 2 unique endpoint errors")
		require.True(t, game.NodeEndpointErrors["client-0"], "Should track client-0 error")
		require.True(t, game.NodeEndpointErrors["client-1"], "Should track client-1 error")
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
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 3, game.NodeEndpointTotalCount)
		require.Equal(t, 2, game.NodeEndpointNotFoundCount, "Should track 2 not found responses")
		require.Equal(t, 0, game.NodeEndpointErrorCount, "Should have no errors")
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
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		// This should result in disagreement due to mixed safety
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 4, game.NodeEndpointTotalCount)
		require.Equal(t, 2, game.NodeEndpointSafeCount, "Should track 2 safe assessments")
		require.Equal(t, 2, game.NodeEndpointUnsafeCount, "Should track 2 unsafe assessments")
		require.True(t, game.HasMixedSafety(), "Should detect mixed safety")
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
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.True(t, game.NodeEndpointDifferentRoots, "Should flag divergent super roots")
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
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 3, game.NodeEndpointTotalCount)
		require.Equal(t, 1, game.NodeEndpointNotFoundCount)
		require.True(t, game.HasMixedAvailability(), "Should detect mixed availability")
	})

	t.Run("AllFieldsZeroWhenNoEndpoints", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		validator := NewSuperAgreementEnricher(logger, &stubOutputMetrics{}, []SuperRootProvider{}, clock.NewDeterministicClock(time.Unix(9824924, 499)))

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:          200,
			L2SequenceNumber:   0,
			RootClaim:          mockRootClaim,
			NodeEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrSuperNodeRpcRequired)

		// Verify all counts remain zero when no endpoints
		require.Equal(t, 0, game.NodeEndpointTotalCount)
		require.Equal(t, 0, game.NodeEndpointErrorCount)
		require.Equal(t, 0, game.NodeEndpointNotFoundCount)
		require.Equal(t, 0, game.NodeEndpointSafeCount)
		require.Equal(t, 0, game.NodeEndpointUnsafeCount)
		require.False(t, game.NodeEndpointDifferentRoots)
	})
}
