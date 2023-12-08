package fault

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
	mockValidatorError = fmt.Errorf("mock validator error")
)

func TestProgressGame_LogErrorFromAct(t *testing.T) {
	handler, game, actor := setupProgressGameTest(t)
	actor.actErr = errors.New("boom")
	status := game.ProgressGame(context.Background())
	require.Equal(t, types.GameStatusInProgress, status)
	require.Equal(t, 1, actor.callCount, "should perform next actions")
	errLog := handler.FindLog(log.LvlError, "Error when acting on game")
	require.NotNil(t, errLog, "should log error")
	require.Equal(t, actor.actErr, errLog.GetContextValue("err"))

	// Should still log game status
	msg := handler.FindLog(log.LvlInfo, "Game info")
	require.NotNil(t, msg)
	require.Equal(t, uint64(1), msg.GetContextValue("claims"))
}

func TestProgressGame_LogGameStatus(t *testing.T) {
	tests := []struct {
		name   string
		status types.GameStatus
		logMsg string
	}{
		{
			name:   "ChallengerWon",
			status: types.GameStatusChallengerWon,
			logMsg: "Game resolved",
		},
		{
			name:   "DefenderWon",
			status: types.GameStatusDefenderWon,
			logMsg: "Game resolved",
		},
		{
			name:   "GameInProgress",
			status: types.GameStatusInProgress,
			logMsg: "Game info",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			handler, game, gameState := setupProgressGameTest(t)
			gameState.status = test.status

			status := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "should perform next actions")
			require.Equal(t, test.status, status)
			errLog := handler.FindLog(log.LvlInfo, test.logMsg)
			require.NotNil(t, errLog, "should log game result")
			require.Equal(t, test.status, errLog.GetContextValue("status"))
		})
	}
}

func TestDoNotActOnCompleteGame(t *testing.T) {
	for _, status := range []types.GameStatus{types.GameStatusChallengerWon, types.GameStatusDefenderWon} {
		t.Run(status.String(), func(t *testing.T) {
			_, game, gameState := setupProgressGameTest(t)
			gameState.status = status

			fetched := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "acts the first time")
			require.Equal(t, status, fetched)

			// Should not act when it knows the game is already complete
			fetched = game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "does not act after game is complete")
			require.Equal(t, status, fetched)
		})
	}
}

func TestValidatePrestate(t *testing.T) {
	tests := []struct {
		name       string
		validators []Validator
		errors     bool
	}{
		{
			name:       "SingleValidator",
			validators: []Validator{&mockValidator{}},
			errors:     false,
		},
		{
			name:       "MultipleValidators",
			validators: []Validator{&mockValidator{}, &mockValidator{}},
			errors:     false,
		},
		{
			name:       "SingleValidator_Errors",
			validators: []Validator{&mockValidator{true}},
			errors:     true,
		},
		{
			name:       "MultipleValidators_Errors",
			validators: []Validator{&mockValidator{}, &mockValidator{true}},
			errors:     true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			player := &GamePlayer{
				prestateValidators: test.validators,
			}
			err := player.ValidatePrestate(context.Background())
			if test.errors {
				require.ErrorIs(t, err, mockValidatorError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

var _ Validator = (*mockValidator)(nil)

type mockValidator struct {
	err bool
}

func (m *mockValidator) Validate(ctx context.Context) error {
	if m.err {
		return mockValidatorError
	}
	return nil
}

func setupProgressGameTest(t *testing.T) (*testlog.CapturingHandler, *GamePlayer, *stubGameState) {
	logger := testlog.Logger(t, log.LvlDebug)
	handler := &testlog.CapturingHandler{
		Delegate: logger.GetHandler(),
	}
	logger.SetHandler(handler)
	gameState := &stubGameState{claimCount: 1}
	game := &GamePlayer{
		act:    gameState.Act,
		loader: gameState,
		logger: logger,
	}
	return handler, game, gameState
}

type stubGameState struct {
	status     types.GameStatus
	claimCount uint64
	callCount  int
	actErr     error
	Err        error
}

func (s *stubGameState) Act(ctx context.Context) error {
	s.callCount++
	return s.actErr
}

func (s *stubGameState) GetStatus(ctx context.Context) (types.GameStatus, error) {
	return s.status, nil
}

func (s *stubGameState) GetClaimCount(ctx context.Context) (uint64, error) {
	return s.claimCount, nil
}

func (s *stubGameState) GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	return common.Hash{}, s.Err
}
