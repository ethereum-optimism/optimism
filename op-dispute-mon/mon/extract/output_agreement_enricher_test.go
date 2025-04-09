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

func TestDetector_CheckOutputRootAgreement(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWhenNoRollupClient", func(t *testing.T) {
		validator, _, _ := setupOutputValidatorTest(t)
		validator.client = nil
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
				validator.client = nil // Should not error even though there's no rollup client
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

	t.Run("OutputFetchFails", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		rollup.outputErr = errors.New("boom")
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, rollup.outputErr)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_Safe", func(t *testing.T) {
		validator, _, metrics := setupOutputValidatorTest(t)
		game := &types.EnrichedGameData{
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

	t.Run("OutputMatches_Safe", func(t *testing.T) {
		validator, _, metrics := setupOutputValidatorTest(t)
		game := &types.EnrichedGameData{
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
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadNum = 99
		game := &types.EnrichedGameData{
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

	t.Run("OutputMatches_SafeHeadError", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadErr = errors.New("boom")
		game := &types.EnrichedGameData{
			L1HeadNum:     200,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim) // Assume safe if we can't retrieve the safe head so monitoring isn't dependent on safe head db
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMismatch_SafeHeadError", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadErr = errors.New("boom")
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim) // Not agreed because the root doesn't match
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_NotSafe", func(t *testing.T) {
		validator, client, metrics := setupOutputValidatorTest(t)
		client.safeHeadNum = 99
		game := &types.EnrichedGameData{
			L1HeadNum:     200,
			L2BlockNumber: 100,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputNotFound", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		// This crazy error is what we actually get back from the API
		rollup.outputErr = errors.New("failed to get L2 block ref with sync status: failed to determine L2BlockRef of height 42984924, could not get payload: not found")
		game := &types.EnrichedGameData{
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
}

func setupOutputValidatorTest(t *testing.T) (*OutputAgreementEnricher, *stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubRollupClient{safeHeadNum: 99999999999}
	metrics := &stubOutputMetrics{}
	validator := NewOutputAgreementEnricher(logger, metrics, client, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
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
}

func (s *stubRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(mockRootClaim)}, s.outputErr
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
