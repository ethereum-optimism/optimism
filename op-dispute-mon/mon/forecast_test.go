package mon

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	failedForecaseLog     = "Failed to forecast game"
	expectedInProgressLog = "Game is not in progress, skipping forecast"
	unexpectedResultLog   = "Forecasting unexpected game result"
	expectedResultLog     = "Forecasting expected game result"
)

func TestForecast_Forecast_BasicTests(t *testing.T) {
	t.Parallel()

	t.Run("NoGames", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		forecast.Forecast(context.Background(), []types.GameMetadata{})
		require.Equal(t, 0, creator.calls)
		require.Equal(t, 0, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("ContractCreationFails", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.err = errors.New("boom")
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 0, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrContractCreation, creator.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("MetadataFetchFails", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.err = errors.New("boom")
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrMetadataFetch, creator.caller.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("ClaimsFetchFails", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.claimsErr = errors.New("boom")
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrClaimFetch, creator.caller.claimsErr)
		require.Equal(t, expectedErr, err)
	})

	t.Run("RollupFetchFails", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		rollup.err = errors.New("boom")
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		err := l.AttrValue("err")
		expectedErr := fmt.Errorf("%w: %w", ErrRootAgreement, rollup.err)
		require.Equal(t, expectedErr, err)
	})

	t.Run("ChallengerWonGameSkipped", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusChallengerWon}
		creator.caller.claims = createDeepClaimList()[:1]
		expectedGame := types.GameMetadata{}
		forecast.Forecast(context.Background(), []types.GameMetadata{expectedGame})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame, l.AttrValue("game"))
		require.Equal(t, types.GameStatusChallengerWon, l.AttrValue("status"))
	})

	t.Run("DefenderWonGameSkipped", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusDefenderWon}
		creator.caller.claims = createDeepClaimList()[:1]
		expectedGame := types.GameMetadata{}
		forecast.Forecast(context.Background(), []types.GameMetadata{expectedGame})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 0, creator.caller.claimsCalls)
		require.Equal(t, 0, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		l := logs.FindLog(levelFilter, messageFilter)
		require.NotNil(t, l)
		require.Equal(t, expectedGame, l.AttrValue("game"))
		require.Equal(t, types.GameStatusDefenderWon, l.AttrValue("status"))
	})

	t.Run("SingleGame", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.claims = createDeepClaimList()[:1]
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})

	t.Run("MultipleGames", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller.claims = createDeepClaimList()[:1]
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		forecast.Forecast(context.Background(), []types.GameMetadata{{}, {}, {}})
		require.Equal(t, 3, creator.calls)
		require.Equal(t, 3, creator.caller.calls)
		require.Equal(t, 3, creator.caller.claimsCalls)
		require.Equal(t, 3, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
		levelFilter = testlog.NewLevelFilter(log.LevelDebug)
		messageFilter = testlog.NewMessageFilter(expectedInProgressLog)
		require.Nil(t, logs.FindLog(levelFilter, messageFilter))
	})
}

func TestForecast_Forecast_EndLogs(t *testing.T) {
	t.Parallel()

	t.Run("AgreeDefenderWins", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.claims = createDeepClaimList()[:1]
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
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

	t.Run("AgreeChallengerWins", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.claims = createDeepClaimList()[:2]
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
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

	t.Run("DisagreeChallengerWins", func(t *testing.T) {
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.rootClaim = common.Hash{}
		creator.caller.claims = createDeepClaimList()[:2]
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
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
		forecast, creator, rollup, logs := setupForecastTest(t)
		creator.caller = &mockGameCaller{status: types.GameStatusInProgress}
		creator.caller.rootClaim = common.Hash{}
		creator.caller.claims = createDeepClaimList()[:1]
		forecast.Forecast(context.Background(), []types.GameMetadata{{}})
		require.Equal(t, 1, creator.calls)
		require.Equal(t, 1, creator.caller.calls)
		require.Equal(t, 1, creator.caller.claimsCalls)
		require.Equal(t, 1, rollup.calls)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		messageFilter := testlog.NewMessageFilter(failedForecaseLog)
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

func setupForecastTest(t *testing.T) (*forecast, *mockGameCallerCreator, *stubOutputValidator, *testlog.CapturingHandler) {
	logger, capturedLogs := testlog.CaptureLogger(t, log.LvlDebug)
	validator := &stubOutputValidator{}
	caller := &mockGameCaller{rootClaim: mockRootClaim}
	creator := &mockGameCallerCreator{caller: caller}
	return newForecast(logger, creator, validator), creator, validator, capturedLogs
}
