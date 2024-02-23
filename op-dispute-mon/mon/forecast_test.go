package mon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	failedForecastLog     = "Failed to forecast game"
	expectedInProgressLog = "Game is not in progress, skipping forecast"
	unexpectedResultLog   = "Forecasting unexpected game result"
	expectedResultLog     = "Forecasting expected game result"
)

func TestForecast_Forecast_BasicTests(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{})
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("RollupFetchFails", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		rollup.err = errors.New("boom")
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{}})
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrRootAgreement, rollup.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("ChallengerWonGameSkipped", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusChallengerWon}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DefenderWonGameSkipped", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusDefenderWon}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})

	t.Run("SingleGame", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{}})
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("MultipleGames", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{}, {}, {}})
		require.Equal(t, 3, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})
}

func TestForecast_Forecast_EndLogs(t *testing.T) {
	t.Parallel()

	t.Run("AgreeDefenderWins", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		games := []*monTypes.EnrichedGameData{{
			Status:    types.GameStatusInProgress,
			RootClaim: mockRootClaim,
			Claims:    createDeepClaimList()[:1],
		}}
		forecast.Forecast(context.Background(), games)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, mockRootClaim, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})

	t.Run("AgreeChallengerWins", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		games := []*monTypes.EnrichedGameData{{
			Status:    types.GameStatusInProgress,
			RootClaim: mockRootClaim,
			Claims:    createDeepClaimList()[:2],
		}}
		forecast.Forecast(context.Background(), games)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelWarn)
		messageFilter = testlog.NewMessageFilter(unexpectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, mockRootClaim, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DisagreeChallengerWins", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{
			Status: types.GameStatusInProgress,
			Claims: createDeepClaimList()[:2],
		}})
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, common.Hash{}, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DisagreeDefenderWins", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{
			Status: types.GameStatusInProgress,
			Claims: createDeepClaimList()[:1],
		}})
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelWarn)
		messageFilter = testlog.NewMessageFilter(unexpectedResultLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, common.Hash{}, l.AttrValue("rootClaim"))
		require.Equal(t, mockRootClaim, l.AttrValue("expected"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})
}

func TestForecast_Forecast_MultipleGames(t *testing.T) {
	forecast, _, rollup, logs := setupForecastTest(t)
	gameStatus := []types.GameStatus{
		types.GameStatusChallengerWon,
		types.GameStatusInProgress,
		types.GameStatusInProgress,
		types.GameStatusDefenderWon,
		types.GameStatusInProgress,
		types.GameStatusInProgress,
		types.GameStatusDefenderWon,
		types.GameStatusChallengerWon,
		types.GameStatusChallengerWon,
	}
	claims := [][]faultTypes.Claim{
		createDeepClaimList()[:1],
		createDeepClaimList()[:2],
		createDeepClaimList()[:2],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
		createDeepClaimList()[:1],
	}
	rootClaims := []common.Hash{
		{},
		{},
		mockRootClaim,
		{},
		{},
		mockRootClaim,
		{},
		{},
		{},
	}
	games := make([]*monTypes.EnrichedGameData, 9)
	for i := range games {
		games[i] = &monTypes.EnrichedGameData{
			Status:    gameStatus[i],
			Claims:    claims[i],
			RootClaim: rootClaims[i],
		}
	}
	forecast.Forecast(context.Background(), games)
	require.Equal(t, 4, rollup.calls)
	levelFilter := testlog.NewLevelFilter(log.LevelError)
	messageFilter := testlog.NewMessageFilter(failedForecastLog)
	require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	levelFilter = testlog.NewLevelFilter(log.LevelDebug)
	messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
	require.Len(t, logs.FindLogs(levelFilter, messageFilter), 5)
}

func setupForecastTest(t *testing.T) (*forecast, *mockForecastMetrics, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	validator := &stubOutputValidator{}
	metrics := &mockForecastMetrics{}
	return newForecast(logger, metrics, validator), metrics, validator, capturedLogs
}

type mockForecastMetrics struct {
	agreeDefenderAhead      int
	disagreeDefenderAhead   int
	agreeChallengerAhead    int
	disagreeChallengerAhead int
	claimResolutionDelayMax float64
}

func (m *mockForecastMetrics) RecordGameAgreement(status metrics.GameAgreementStatus, count int) {
	switch status {
	case metrics.AgreeDefenderAhead:
		m.agreeDefenderAhead = count
	case metrics.DisagreeDefenderAhead:
		m.disagreeDefenderAhead = count
	case metrics.AgreeChallengerAhead:
		m.agreeChallengerAhead = count
	case metrics.DisagreeChallengerAhead:
		m.disagreeChallengerAhead = count
	}
}

func (m *mockForecastMetrics) RecordClaimResolutionDelayMax(delay float64) {
	m.claimResolutionDelayMax = delay
}

func createDeepClaimList() []faultTypes.Claim {
	return []faultTypes.Claim{
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(0, big.NewInt(0)),
			},
			ContractIndex:       0,
			ParentContractIndex: math.MaxInt64,
			Claimant:            common.HexToAddress("0x111111"),
		},
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(1, big.NewInt(0)),
			},
			ContractIndex:       1,
			ParentContractIndex: 0,
			Claimant:            common.HexToAddress("0x222222"),
		},
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(2, big.NewInt(0)),
			},
			ContractIndex:       2,
			ParentContractIndex: 1,
			Claimant:            common.HexToAddress("0x111111"),
		},
	}
}
