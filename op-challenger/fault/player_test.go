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

func TestProgressGame_LogErrorFromAct(t *testing.T) {
	handler, game, actor := setupProgressGameTest(t, true)
	actor.actErr = errors.New("boom")
	done := game.ProgressGame(context.Background())
	require.False(t, done, "should not be done")
	require.Equal(t, 1, actor.callCount, "should perform next actions")
	errLog := handler.FindLog(log.LvlError, "Error when acting on game")
	require.NotNil(t, errLog, "should log error")
	require.Equal(t, actor.actErr, errLog.GetContextValue("err"))

	// Should still log game status
	msg := handler.FindLog(log.LvlInfo, "Game info")
	require.NotNil(t, msg)
	require.Equal(t, uint64(1), msg.GetContextValue("claims"))
}

func TestCleanup(t *testing.T) {
	t.Run("default cleanup", func(t *testing.T) {
		game := &GamePlayer{
			cleanup: func() error { return nil },
		}
		require.NoError(t, game.Cleanup())
	})

	t.Run("cleanup bubbles up error", func(t *testing.T) {
		err := errors.New("wassie")
		game := &GamePlayer{
			cleanup: func() error { return err },
		}
		require.Error(t, err, game.Cleanup())
	})

}

func TestProgressGame_LogGameStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          types.GameStatus
		agreeWithOutput bool
		logLevel        log.Lvl
		logMsg          string
	}{
		{
			name:            "GameLostAsDefender",
			status:          types.GameStatusChallengerWon,
			agreeWithOutput: false,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
		},
		{
			name:            "GameLostAsChallenger",
			status:          types.GameStatusDefenderWon,
			agreeWithOutput: true,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
		},
		{
			name:            "GameWonAsDefender",
			status:          types.GameStatusDefenderWon,
			agreeWithOutput: false,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
		},
		{
			name:            "GameWonAsChallenger",
			status:          types.GameStatusChallengerWon,
			agreeWithOutput: true,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
		},
		{
			name:            "GameInProgress",
			status:          types.GameStatusInProgress,
			agreeWithOutput: true,
			logLevel:        log.LvlInfo,
			logMsg:          "Game info",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			handler, game, gameState := setupProgressGameTest(t, test.agreeWithOutput)
			gameState.preStatus = types.GameStatusInProgress
			gameState.postStatus = test.status

			done := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "should perform next actions")
			require.Equal(t, test.status != types.GameStatusInProgress, done, "should be done when not in progress")
			errLog := handler.FindLog(test.logLevel, test.logMsg)
			require.NotNil(t, errLog, "should log game result")
			require.Equal(t, test.status, errLog.GetContextValue("status"))
		})
	}
}

func TestDoNotActOnCompleteGame(t *testing.T) {
	for _, status := range []types.GameStatus{types.GameStatusChallengerWon, types.GameStatusDefenderWon} {
		t.Run(status.String(), func(t *testing.T) {
			handler, game, gameState := setupProgressGameTest(t, true)
			gameState.preStatus = status

			done := game.ProgressGame(context.Background())
			require.Equal(t, 0, gameState.callCount, "should not perform actions")
			require.True(t, done, "should be done")
			require.Empty(t, handler.Logs, "should not log game status")
		})
	}
}

func setupProgressGameTest(t *testing.T, agreeWithProposedRoot bool) (*testlog.CapturingHandler, *GamePlayer, *stubGameState) {
	logger := testlog.Logger(t, log.LvlDebug)
	handler := &testlog.CapturingHandler{
		Delegate: logger.GetHandler(),
	}
	logger.SetHandler(handler)
	gameState := &stubGameState{claimCount: 1}
	game := &GamePlayer{
		agent:                   gameState,
		agreeWithProposedOutput: agreeWithProposedRoot,
		caller:                  gameState,
		logger:                  logger,
	}
	return handler, game, gameState
}

type stubGameState struct {
	preStatus  types.GameStatus
	postStatus types.GameStatus
	claimCount uint64
	callCount  int
	actErr     error
	Err        error
}

func (s *stubGameState) Act(ctx context.Context) error {
	s.callCount++
	return s.actErr
}

func (s *stubGameState) GetGameStatus(ctx context.Context) (types.GameStatus, error) {
	if s.callCount == 0 {
		return s.preStatus, nil
	}
	return s.postStatus, nil
}

func (s *stubGameState) GetClaimCount(ctx context.Context) (uint64, error) {
	return s.claimCount, nil
}
