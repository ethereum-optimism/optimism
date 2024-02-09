package mon

import (
	"context"
	"errors"
	"fmt"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []types.GameMetadata{})
		require.Equal(t, 0, creator.calls)
		require.Equal(t, 0, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("ContractCreationFails", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.err = errors.New("boom")
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 0, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrContractCreation, creator.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("MetadataFetchFails", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.err = errors.New("boom")
		creator.caller.rootClaim = []common.Hash{mockRootClaim}
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrMetadataFetch, creator.caller.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("ClaimsFetchFails", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.claimsErr = errors.New("boom")
		creator.caller.rootClaim = []common.Hash{mockRootClaim}
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrClaimFetch, creator.caller.claimsErr)
		require.Equal(t, expectedErr, err)
	})

	t.Run("RollupFetchFails", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		rollup.err = errors.New("boom")
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		creator.caller.rootClaim = []common.Hash{mockRootClaim}
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusChallengerWon}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		expectedGame := types.GameMetadata{}
		forecast.Forecast(context.Background(), []types.GameMetadata{expectedGame})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame, l.AttrValue("game"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DefenderWonGameSkipped", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusDefenderWon}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		expectedGame := types.GameMetadata{}
		forecast.Forecast(context.Background(), []types.GameMetadata{expectedGame})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame, l.AttrValue("game"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})

	t.Run("SingleGame", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecastLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("MultipleGames", func(t *testing.T) {
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress, types.GameStatusInProgress, types.GameStatusInProgress}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1], createDeepClaimList()[:1], createDeepClaimList()[:1]}
		creator.caller.rootClaim = []common.Hash{{}, {}, {}}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}, {}, {}})
		require.Equal(t, 3, creator.calls)
		require.Equal(t, 3, creator.caller.calls)
		require.Equal(t, 3, creator.caller.claimsCalls)
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller.status = []types.GameStatus{types.GameStatusInProgress}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:2]}
		creator.caller.rootClaim = []common.Hash{mockRootClaim}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: []types.GameStatus{types.GameStatusInProgress}}
		creator.caller.rootClaim = []common.Hash{{}}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:2]}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
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
		forecast, _, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: []types.GameStatus{types.GameStatusInProgress}}
		creator.caller.rootClaim = []common.Hash{{}}
		creator.caller.claims = [][]faultTypes.Claim{createDeepClaimList()[:1]}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
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
	forecast, _, creator, rollup, logs := setupForecastTest(t)
	creator.caller.status = []types.GameStatus{
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
	creator.caller.claims = [][]faultTypes.Claim{
		createDeepClaimList()[:1],
		createDeepClaimList()[:2],
		createDeepClaimList()[:2],
		createDeepClaimList()[:1],
	}
	creator.caller.rootClaim = []common.Hash{
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
	games := make([]types.GameMetadata, 9)
	forecast.Forecast(context.Background(), games)
	require.Equal(t, 9, creator.calls)
	require.Equal(t, 9, creator.caller.calls)
	require.Equal(t, 4, creator.caller.claimsCalls)
	require.Equal(t, 4, rollup.calls)
	levelFilter := testlog.NewLevelFilter(log.LevelError)
	messageFilter := testlog.NewMessageFilter(failedForecastLog)
	require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	levelFilter = testlog.NewLevelFilter(log.LevelDebug)
	messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
	require.Len(t, logs.FindLogs(levelFilter, messageFilter), 5)
}

func setupForecastTest(t *testing.T) (*forecast, *mockForecastMetrics, *mockGameCallerCreator, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	validator := &stubOutputValidator{}
	caller := &mockGameCaller{rootClaim: []common.Hash{mockRootClaim}}
	creator := &mockGameCallerCreator{caller: caller}
	metrics := &mockForecastMetrics{}
	return newForecast(logger, metrics, creator, validator), metrics, creator, validator, capturedLogs
}

type mockForecastMetrics struct {
	agreeDefenderAhead      int
	disagreeDefenderAhead   int
	agreeChallengerAhead    int
	disagreeChallengerAhead int
}

func (m *mockForecastMetrics) RecordGameAgreement(status string, count int) {
	switch status {
	case "agree_defender_ahead":
		m.agreeDefenderAhead = count
	case "disagree_defender_ahead":
		m.disagreeDefenderAhead = count
	case "agree_challenger_ahead":
		m.agreeChallengerAhead = count
	case "disagree_challenger_ahead":
		m.disagreeChallengerAhead = count
	}
}
