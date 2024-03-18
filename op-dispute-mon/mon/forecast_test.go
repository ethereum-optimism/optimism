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
	failedForecastLog   = "Failed to forecast game"
	lostGameLog         = "Unexpected game result"
	unexpectedResultLog = "Forecasting unexpected game result"
	expectedResultLog   = "Forecasting expected game result"
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

	t.Run("ChallengerWonGame_Agree", func(t *testing.T) {
		forecast, m, _, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusChallengerWon, RootClaim: mockRootClaim}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("expectedResult"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("actualResult"))

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.AgreeChallengerWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("ChallengerWonGame_Disagree", func(t *testing.T) {
		forecast, m, _, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusChallengerWon, RootClaim: common.Hash{0xbb}}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.Nil(t, l)

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.DisagreeChallengerWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("DefenderWonGame_Agree", func(t *testing.T) {
		forecast, m, _, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusDefenderWon, RootClaim: mockRootClaim}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.Nil(t, l)

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.AgreeDefenderWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("DefenderWonGame_Disagree", func(t *testing.T) {
		forecast, m, _, logs := setupForecastTest(t)
		expectedGame := monTypes.EnrichedGameData{Status: types.GameStatusDefenderWon, RootClaim: common.Hash{0xbb}}
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{&expectedGame})
		l := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(lostGameLog))
		require.NotNil(t, l)
		require.Equal(t, expectedGame.Proxy, l.AttrValue("game"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("expectedResult"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("actualResult"))

		expectedMetrics := zeroGameAgreement()
		expectedMetrics[metrics.DisagreeDefenderWins] = 1
		require.Equal(t, expectedMetrics, m.gameAgreement)
	})

	t.Run("SingleGame", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{}})
		require.Equal(t, 1, rollup.calls)
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(failedForecastLog)))
	})

	t.Run("MultipleGames", func(t *testing.T) {
		forecast, _, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []*monTypes.EnrichedGameData{{}, {}, {}})
		require.Equal(t, 3, rollup.calls)
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter(failedForecastLog)))
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
	require.Equal(t, len(games), rollup.calls)
	levelFilter := testlog.NewLevelFilter(log.LevelError)
	messageFilter := testlog.NewMessageFilter(failedForecastLog)
	require.Nil(t, logs.FindLog(levelFilter, messageFilter))
}

func setupForecastTest(t *testing.T) (*forecast, *mockForecastMetrics, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	validator := &stubOutputValidator{}
	metrics := &mockForecastMetrics{
		gameAgreement: zeroGameAgreement(),
	}
	return newForecast(logger, metrics, validator), metrics, validator, capturedLogs
}

func zeroGameAgreement() map[metrics.GameAgreementStatus]int {
	return map[metrics.GameAgreementStatus]int{
		metrics.AgreeChallengerAhead:    0,
		metrics.DisagreeChallengerAhead: 0,
		metrics.AgreeDefenderAhead:      0,
		metrics.DisagreeDefenderAhead:   0,
		metrics.AgreeDefenderWins:       0,
		metrics.DisagreeDefenderWins:    0,
		metrics.AgreeChallengerWins:     0,
		metrics.DisagreeChallengerWins:  0,
	}
}

type mockForecastMetrics struct {
	gameAgreement           map[metrics.GameAgreementStatus]int
	claimResolutionDelayMax float64
}

func (m *mockForecastMetrics) RecordGameAgreement(status metrics.GameAgreementStatus, count int) {
	m.gameAgreement[status] = count
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

type stubOutputValidator struct {
	calls int
	err   error
}

func (s *stubOutputValidator) CheckRootAgreement(_ context.Context, _ uint64, _ uint64, rootClaim common.Hash) (bool, common.Hash, error) {
	s.calls++
	if s.err != nil {
		return false, common.Hash{}, s.err
	}
	return rootClaim == mockRootClaim, mockRootClaim, nil
}
