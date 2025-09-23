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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDetector_CheckSuperRootAgreement(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWhenNoSupervisorClient", func(t *testing.T) {
		validator, _, _ := setupSuperValidatorTest(t)
		validator.clients = nil // Set to nil to test the error case
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrSupervisorRpcRequired)
	})

	t.Run("SkipOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{0, 1, 2, 3, 6, 254, 255, 1337}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupSuperValidatorTest(t)
				validator.clients = nil // Should not error even though there's no supervisor client
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:     200,
					L2BlockNumber: 0,
					RootClaim:     mockRootClaim,
				}
				err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
				require.NoError(t, err)
				require.Zero(t, metrics.fetchTime)
			})
		}
	})

	t.Run("FetchAllNonOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{4, 5, 7, 8, 10, 49812} // Treat unknown game types as using super roots
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupSuperValidatorTest(t)
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:     200,
					L2BlockNumber: 0,
					RootClaim:     mockRootClaim,
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
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrAllSupervisorNodesUnavailable)
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
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     common.Hash{},
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
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
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
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
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
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     common.Hash{},
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
			L1HeadNum:     200,
			L2BlockNumber: 100,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputNotFound", func(t *testing.T) {
		validator, client, metrics := setupSuperValidatorTest(t)
		// The supervisor client automatically translates RPC errors back to ethereum.NotFound for us
		client.outputErr = ethereum.NotFound
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     100,
			L2BlockNumber: 42984924,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllSupervisorNodesReturnError", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		for _, client := range clients {
			client.outputErr = errors.New("boom")
		}
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAllSupervisorNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllSupervisorNodesReturnNotFound", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		for _, client := range clients {
			client.outputErr = ethereum.NotFound
		}
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("SomeSupervisorNodesOutOfSync", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		clients[0].outputErr = ethereum.NotFound
		clients[1].outputErr = nil
		clients[2].outputErr = nil
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("SupervisorNodesDiverged", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		divergedRoot := common.HexToHash("0x5678")
		clients[0].superRoot = mockRootClaim
		clients[1].superRoot = divergedRoot
		clients[2].superRoot = divergedRoot
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllSupervisorNodesAgree", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		clients[0].derivedFromL1BlockNum = 200
		clients[1].derivedFromL1BlockNum = 199
		clients[2].derivedFromL1BlockNum = 201
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesMatchClaimAndSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 4)
		clients[0].outputErr = ethereum.NotFound
		clients[1].outputErr = ethereum.NotFound
		clients[2].superRoot = mockRootClaim
		clients[2].derivedFromL1BlockNum = 100 // Safe because L1HeadNum is 200
		clients[3].superRoot = mockRootClaim
		clients[3].derivedFromL1BlockNum = 150 // Safe because L1HeadNum is 200
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim) // Should disagree due to mixed responses (divergence)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesDontMatchClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)
		differentRoot := common.HexToHash("0x9999")
		clients[0].outputErr = ethereum.NotFound
		clients[1].superRoot = differentRoot
		clients[1].derivedFromL1BlockNum = 100
		clients[2].superRoot = differentRoot
		clients[2].derivedFromL1BlockNum = 150
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim) // Should disagree due to mixed responses (divergence)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_SuperRootMatchesClaim_NoneReportSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)

		for _, client := range clients {
			client.superRoot = mockRootClaim
			client.derivedFromL1BlockNum = 250 // Not safe because L1HeadNum is 200
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim, "Should set ExpectedRootClaim to empty hash when not safe")
		require.False(t, game.AgreeWithClaim, "Should disagree because none report it as safe")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_SuperRootDifferentFromClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiSupervisorTest(t, 3)

		differentRoot := common.HexToHash("0xdifferent")
		for _, client := range clients {
			client.superRoot = differentRoot
			client.derivedFromL1BlockNum = 100 // Safe because L1HeadNum is 200
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 999,
			},
			L1HeadNum:     200,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim, "Should disagree because super root differs from claim")
		require.NotZero(t, metrics.fetchTime)
	})
}

func setupSuperValidatorTest(t *testing.T) (*SuperAgreementEnricher, *stubSupervisorClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSupervisorClient{derivedFromL1BlockNum: 0, superRoot: mockRootClaim}
	metrics := &stubOutputMetrics{}
	validator := NewSuperAgreementEnricher(logger, metrics, []SuperRootProvider{client}, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
}

func setupMultiSupervisorTest(t *testing.T, numNodes int) (*SuperAgreementEnricher, []*stubSupervisorClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	clients := make([]*stubSupervisorClient, numNodes)
	supervisorClients := make([]SuperRootProvider, numNodes)
	for i := range clients {
		clients[i] = &stubSupervisorClient{
			derivedFromL1BlockNum: 0,
			superRoot:             mockRootClaim,
		}
		supervisorClients[i] = clients[i]
	}
	metrics := &stubOutputMetrics{}
	validator := NewSuperAgreementEnricher(logger, metrics, supervisorClients, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, clients, metrics
}

type stubSupervisorClient struct {
	requestedTimestamp    uint64
	outputErr             error
	derivedFromL1BlockNum uint64
	superRoot             common.Hash
}

func (s *stubSupervisorClient) SuperRootAtTimestamp(_ context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	s.requestedTimestamp = uint64(timestamp)
	if s.outputErr != nil {
		return eth.SuperRootResponse{}, s.outputErr
	}
	return eth.SuperRootResponse{
		CrossSafeDerivedFrom: eth.BlockID{Number: s.derivedFromL1BlockNum},
		Timestamp:            uint64(timestamp),
		SuperRoot:            eth.Bytes32(s.superRoot),
		Version:              eth.SuperRootVersionV1,
	}, nil
}
