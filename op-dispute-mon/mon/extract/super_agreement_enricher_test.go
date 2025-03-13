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
		validator.client = nil
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
		require.ErrorIs(t, err, rollup.outputErr)
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
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
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
}

func setupSuperValidatorTest(t *testing.T) (*SuperAgreementEnricher, *stubSupervisorClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubSupervisorClient{derivedFromL1BlockNum: 0}
	metrics := &stubOutputMetrics{}
	validator := NewSuperAgreementEnricher(logger, metrics, client, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
}

type stubSupervisorClient struct {
	requestedTimestamp    uint64
	outputErr             error
	derivedFromL1BlockNum uint64
}

func (s *stubSupervisorClient) SuperRootAtTimestamp(_ context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	s.requestedTimestamp = uint64(timestamp)
	if s.outputErr != nil {
		return eth.SuperRootResponse{}, s.outputErr
	}
	return eth.SuperRootResponse{
		CrossSafeDerivedFrom: eth.BlockID{Number: s.derivedFromL1BlockNum},
		Timestamp:            uint64(timestamp),
		SuperRoot:            eth.Bytes32(mockRootClaim),
		Version:              eth.SuperRootVersionV1,
	}, nil
}
