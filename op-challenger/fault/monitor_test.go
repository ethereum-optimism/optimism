package fault

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestMonitorExitsWhenContextDone(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)
	actor := &stubActor{}
	gameInfo := &stubGameInfo{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := MonitorGame(ctx, logger, true, actor, gameInfo)
	require.ErrorIs(t, err, context.Canceled)
}

func TestProgressGameAndLogState(t *testing.T) {
	logger, _, actor, gameInfo := setupProgressGameTest(t)
	done := progressGame(context.Background(), logger, true, actor, gameInfo)
	require.False(t, done, "should not be done")
	require.Equal(t, 1, actor.callCount, "should perform next actions")
	require.Equal(t, 1, gameInfo.logCount, "should log latest game state")
}

func TestProgressGame_LogErrorFromAct(t *testing.T) {
	logger, handler, actor, gameInfo := setupProgressGameTest(t)
	actor.err = errors.New("Boom")
	done := progressGame(context.Background(), logger, true, actor, gameInfo)
	require.False(t, done, "should not be done")
	require.Equal(t, 1, actor.callCount, "should perform next actions")
	require.Equal(t, 1, gameInfo.logCount, "should log latest game state")
	errLog := handler.FindLog(log.LvlError, "Error when acting on game")
	require.NotNil(t, errLog, "should log error")
	require.Equal(t, actor.err, errLog.GetContextValue("err"))
}

func TestProgressGame_LogErrorWhenGameLost(t *testing.T) {
	tests := []struct {
		name            string
		status          types.GameStatus
		agreeWithOutput bool
		logLevel        log.Lvl
		logMsg          string
		statusText      string
	}{
		{
			name:            "GameLostAsDefender",
			status:          types.GameStatusChallengerWon,
			agreeWithOutput: false,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
			statusText:      "Challenger Won",
		},
		{
			name:            "GameLostAsChallenger",
			status:          types.GameStatusDefenderWon,
			agreeWithOutput: true,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
			statusText:      "Defender Won",
		},
		{
			name:            "GameWonAsDefender",
			status:          types.GameStatusDefenderWon,
			agreeWithOutput: false,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
			statusText:      "Defender Won",
		},
		{
			name:            "GameWonAsChallenger",
			status:          types.GameStatusChallengerWon,
			agreeWithOutput: true,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
			statusText:      "Challenger Won",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			logger, handler, actor, gameInfo := setupProgressGameTest(t)
			gameInfo.status = test.status

			done := progressGame(context.Background(), logger, test.agreeWithOutput, actor, gameInfo)
			require.True(t, done, "should be done")
			require.Equal(t, 0, gameInfo.logCount, "should not log latest game state")
			errLog := handler.FindLog(test.logLevel, test.logMsg)
			require.NotNil(t, errLog, "should log game result")
			require.Equal(t, test.statusText, errLog.GetContextValue("status"))
		})
	}
}

func setupProgressGameTest(t *testing.T) (log.Logger, *testlog.CapturingHandler, *stubActor, *stubGameInfo) {
	logger := testlog.Logger(t, log.LvlDebug)
	handler := &testlog.CapturingHandler{
		Delegate: logger.GetHandler(),
	}
	logger.SetHandler(handler)
	actor := &stubActor{}
	gameInfo := &stubGameInfo{}
	return logger, handler, actor, gameInfo
}

type stubActor struct {
	callCount int
	err       error
}

func (a *stubActor) Act(ctx context.Context) error {
	a.callCount++
	return a.err
}

type stubGameInfo struct {
	status   types.GameStatus
	err      error
	logCount int
}

func (s *stubGameInfo) GetGameStatus(ctx context.Context) (types.GameStatus, error) {
	return s.status, s.err
}

func (s *stubGameInfo) LogGameInfo(ctx context.Context) {
	s.logCount++
}
