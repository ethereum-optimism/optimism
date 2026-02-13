package generic

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockValidatorError = fmt.Errorf("mock validator error")

func TestProgressGame_LogErrorFromAct(t *testing.T) {
	handler, game, actor, _ := setupProgressGameTest(t)
	actor.actErr = errors.New("boom")
	status := game.ProgressGame(context.Background())
	require.Equal(t, types.GameStatusInProgress, status)
	require.Equal(t, 1, actor.callCount, "should perform next actions")
	levelFilter := testlog.NewLevelFilter(log.LevelError)
	msgFilter := testlog.NewMessageFilter("Error when acting on game")
	errLog := handler.FindLog(levelFilter, msgFilter)
	require.NotNil(t, errLog, "should log error")
	require.Equal(t, actor.actErr, errLog.AttrValue("err"))

	// Should still log game status
	levelFilter = testlog.NewLevelFilter(log.LevelInfo)
	msgFilter = testlog.NewMessageFilter("Game info")
	msg := handler.FindLog(levelFilter, msgFilter)
	require.NotNil(t, msg)
	require.Equal(t, "statusValue", msg.AttrValue("extra"))
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
			handler, game, gameState, _ := setupProgressGameTest(t)
			gameState.status = test.status

			status := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "should perform next actions")
			require.Equal(t, test.status, status)
			levelFilter := testlog.NewLevelFilter(log.LevelInfo)
			msgFilter := testlog.NewMessageFilter(test.logMsg)
			errLog := handler.FindLog(levelFilter, msgFilter)
			require.NotNil(t, errLog, "should log game result")
			require.Equal(t, test.status, errLog.AttrValue("status"))
		})
	}
}

func TestDoNotActOnCompleteGame(t *testing.T) {
	for _, status := range []types.GameStatus{types.GameStatusChallengerWon, types.GameStatusDefenderWon} {
		t.Run(status.String(), func(t *testing.T) {
			_, game, gameState, _ := setupProgressGameTest(t)
			gameState.status = status

			fetched := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "acts the first time")
			require.Equal(t, status, fetched)

			// Should not act when it knows the game is already complete
			fetched = game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "does not act after game is complete")
			require.Equal(t, status, fetched)

			// Should have replaced the act function with a noop so callCount doesn't update even when called directly
			// This allows the agent resources to be GC'd
			require.NoError(t, game.actor.Act(context.Background()))
			require.Equal(t, 1, gameState.callCount)
		})
	}
}

func TestValidateLocalNodeSync(t *testing.T) {
	_, game, gameState, syncValidator := setupProgressGameTest(t)

	game.ProgressGame(context.Background())
	require.Equal(t, 1, gameState.callCount, "acts when in sync")

	syncValidator.result = errors.New("boom")
	game.ProgressGame(context.Background())
	require.Equal(t, 1, gameState.callCount, "does not act when not in sync")
}

func TestValidatePrestate(t *testing.T) {
	tests := []struct {
		name       string
		validators []PrestateValidator
		errors     bool
	}{
		{
			name:       "SingleValidator",
			validators: []PrestateValidator{&mockValidator{}},
			errors:     false,
		},
		{
			name:       "MultipleValidators",
			validators: []PrestateValidator{&mockValidator{}, &mockValidator{}},
			errors:     false,
		},
		{
			name:       "SingleValidator_Errors",
			validators: []PrestateValidator{&mockValidator{true}},
			errors:     true,
		},
		{
			name:       "MultipleValidators_Errors",
			validators: []PrestateValidator{&mockValidator{}, &mockValidator{true}},
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

var _ PrestateValidator = (*mockValidator)(nil)

type mockValidator struct {
	err bool
}

func (m *mockValidator) Validate(_ context.Context) error {
	if m.err {
		return mockValidatorError
	}
	return nil
}

func setupProgressGameTest(t *testing.T) (*testlog.CapturingHandler, *GamePlayer, *stubGameState, *stubSyncValidator) {
	logger, logs := testlog.CaptureLogger(t, log.LevelDebug)
	gameState := &stubGameState{claimCount: 1}
	syncValidator := &stubSyncValidator{}
	game := &GamePlayer{
		actor:         gameState,
		loader:        gameState,
		logger:        logger,
		syncValidator: syncValidator,
		gameL1Head: eth.BlockID{
			Hash:   common.Hash{0x1a},
			Number: 32,
		},
	}
	return logs, game, gameState, syncValidator
}

type stubSyncValidator struct {
	result error
}

func (s *stubSyncValidator) ValidateNodeSynced(_ context.Context, _ eth.BlockID) error {
	return s.result
}

type stubGameState struct {
	status     types.GameStatus
	claimCount uint64
	callCount  int
	actErr     error
	Err        error
}

func (s *stubGameState) AdditionalStatus(_ context.Context) ([]any, error) {
	return []any{"extra", "statusValue"}, nil
}

func (s *stubGameState) Act(_ context.Context) error {
	s.callCount++
	return s.actErr
}

func (s *stubGameState) GetL1Head(_ context.Context) (common.Hash, error) {
	return common.Hash{0x1a}, nil
}

func (s *stubGameState) GetStatus(_ context.Context) (types.GameStatus, error) {
	return s.status, nil
}

func (s *stubGameState) GetClaimCount(_ context.Context) (uint64, error) {
	return s.claimCount, nil
}

func (s *stubGameState) GetAbsolutePrestateHash(_ context.Context) (common.Hash, error) {
	return common.Hash{}, s.Err
}
