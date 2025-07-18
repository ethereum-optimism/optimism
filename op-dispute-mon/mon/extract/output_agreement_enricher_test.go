package extract

import (
	"context"
	"errors"
	"fmt"
	"math"
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

func TestOutputAgreementEnricher(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWhenNoRollupClient", func(t *testing.T) {
		validator, _, _ := setupOutputValidatorTest(t)
		validator.clients = nil
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrRollupRpcRequired)
	})

	t.Run("SkipNonOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{4, 5, 7, 8, 10, 49812}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupOutputValidatorTest(t)
				validator.clients = nil // Should not error even though there's no rollup client
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

	t.Run("FetchAllOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{0, 1, 2, 3, 6, 254, 255, 1337}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupOutputValidatorTest(t)
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

	t.Run("AllNodesReturnError", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		for _, client := range clients {
			client.outputErr = errors.New("boom")
		}
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAllNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllNodesReturnNotFound", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		for _, client := range clients {
			client.outputErr = errors.New("not found")
		}
		game := &types.EnrichedGameData{
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

	t.Run("SomeNodesOutOfSync", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].outputErr = errors.New("not found")
		clients[1].outputErr = nil
		clients[2].outputErr = nil
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesMatchClaimAndSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 4)
		clients[0].outputErr = errors.New("not found")
		clients[1].outputErr = errors.New("not found")
		clients[2].outputRoot = mockRootClaim
		clients[2].safeHeadNum = 100
		clients[3].outputRoot = mockRootClaim
		clients[3].safeHeadNum = 100
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesDontMatchClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		differentRoot := common.HexToHash("0x9999")
		clients[0].outputErr = errors.New("not found")
		clients[1].outputRoot = differentRoot
		clients[2].outputRoot = differentRoot
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("NodesDiverged", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		divergedRoot := common.HexToHash("0x5678")
		clients[0].outputRoot = mockRootClaim
		clients[1].outputRoot = divergedRoot
		clients[2].outputRoot = divergedRoot
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadNum = 100
		clients[1].safeHeadNum = 99
		clients[2].safeHeadNum = 101
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("SafeHeadError", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadErr = errors.New("boom")
		clients[1].safeHeadErr = nil
		clients[2].safeHeadErr = nil
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_NotSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadNum = 50
		clients[1].safeHeadNum = 60
		clients[2].safeHeadNum = 70
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 80,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_OutputMatchesClaim_NoneReportSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)

		for _, client := range clients {
			client.outputRoot = mockRootClaim
			client.safeHeadNum = 40
		}

		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 50, // Higher than all safe heads
			RootClaim:     mockRootClaim,
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim, "Should set ExpectedRootClaim to empty hash when not safe")
		require.False(t, game.AgreeWithClaim, "Should disagree because none report it as safe")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_OutputDifferentFromClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)

		differentRoot := common.HexToHash("0xdifferent")
		for _, client := range clients {
			client.outputRoot = differentRoot
			// Safe head numbers don't matter here since the output doesn't match the claim
		}

		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 50,
			RootClaim:     mockRootClaim,
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim, "Should set ExpectedRootClaim to the agreed output")
		require.False(t, game.AgreeWithClaim, "Should disagree because output doesn't match claim")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("BlockNumberLargerThanInt64", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		// RPC block numbers must be a int64 to be valid. Anything bigger than that should be treated as invalid
		// without even making a request to the node.
		rollup.outputErr = errors.New("should not have even requested the output root")
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: uint64(math.MaxInt64) + 1,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})
}

func setupOutputValidatorTest(t *testing.T) (*OutputAgreementEnricher, *stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubRollupClient{safeHeadNum: 99999999999}
	metrics := &stubOutputMetrics{}
	validator := NewOutputAgreementEnricher(logger, metrics, []OutputRollupClient{client}, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
}

func setupMultiNodeTest(t *testing.T, numNodes int) (*OutputAgreementEnricher, []*stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	clients := make([]*stubRollupClient, numNodes)
	rollupClients := make([]OutputRollupClient, numNodes)
	for i := range clients {
		clients[i] = &stubRollupClient{
			safeHeadNum: 99999999999,
			outputRoot:  mockRootClaim,
		}
		rollupClients[i] = clients[i]
	}
	metrics := &stubOutputMetrics{}
	validator := NewOutputAgreementEnricher(logger, metrics, rollupClients, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, clients, metrics
}

type stubOutputMetrics struct {
	fetchTime float64
}

func (s *stubOutputMetrics) RecordOutputFetchTime(fetchTime float64) {
	s.fetchTime = fetchTime
}

type stubRollupClient struct {
	blockNum    uint64
	outputErr   error
	safeHeadErr error
	safeHeadNum uint64
	outputRoot  common.Hash
}

func (s *stubRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	if s.outputErr != nil {
		return nil, s.outputErr
	}
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(s.outputRoot)}, nil
}

func (s *stubRollupClient) SafeHeadAtL1Block(_ context.Context, _ uint64) (*eth.SafeHeadResponse, error) {
	if s.safeHeadErr != nil {
		return nil, s.safeHeadErr
	}
	return &eth.SafeHeadResponse{
		SafeHead: eth.BlockID{
			Number: s.safeHeadNum,
		},
	}, nil
}
