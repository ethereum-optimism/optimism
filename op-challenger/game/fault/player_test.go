package fault

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockTraceProviderError = fmt.Errorf("mock trace provider error")
	mockLoaderError        = fmt.Errorf("mock loader error")
)

func TestProgressGame_LogErrorFromAct(t *testing.T) {
	handler, game, actor := setupProgressGameTest(t, true)
	actor.actErr = errors.New("boom")
	status := game.ProgressGame(context.Background())
	require.Equal(t, gameTypes.GameStatusInProgress, status)
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
		name            string
		status          gameTypes.GameStatus
		agreeWithOutput bool
		logLevel        log.Lvl
		logMsg          string
	}{
		{
			name:            "GameLostAsDefender",
			status:          gameTypes.GameStatusChallengerWon,
			agreeWithOutput: false,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
		},
		{
			name:            "GameLostAsChallenger",
			status:          gameTypes.GameStatusDefenderWon,
			agreeWithOutput: true,
			logLevel:        log.LvlError,
			logMsg:          "Game lost",
		},
		{
			name:            "GameWonAsDefender",
			status:          gameTypes.GameStatusDefenderWon,
			agreeWithOutput: false,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
		},
		{
			name:            "GameWonAsChallenger",
			status:          gameTypes.GameStatusChallengerWon,
			agreeWithOutput: true,
			logLevel:        log.LvlInfo,
			logMsg:          "Game won",
		},
		{
			name:            "GameInProgress",
			status:          gameTypes.GameStatusInProgress,
			agreeWithOutput: true,
			logLevel:        log.LvlInfo,
			logMsg:          "Game info",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			handler, game, gameState := setupProgressGameTest(t, test.agreeWithOutput)
			gameState.status = test.status

			status := game.ProgressGame(context.Background())
			require.Equal(t, 1, gameState.callCount, "should perform next actions")
			require.Equal(t, test.status, status)
			errLog := handler.FindLog(test.logLevel, test.logMsg)
			require.NotNil(t, errLog, "should log game result")
			require.Equal(t, test.status, errLog.GetContextValue("status"))
		})
	}
}

func TestDoNotActOnCompleteGame(t *testing.T) {
	for _, status := range []gameTypes.GameStatus{gameTypes.GameStatusChallengerWon, gameTypes.GameStatusDefenderWon} {
		t.Run(status.String(), func(t *testing.T) {
			_, game, gameState := setupProgressGameTest(t, true)
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

// TestValidateAbsolutePrestate tests that the absolute prestate is validated
// correctly by the service component.
func TestValidateAbsolutePrestate(t *testing.T) {
	t.Run("ValidPrestates", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		prestateHash := crypto.Keccak256(prestate)
		prestateHash[0] = mipsevm.VMStatusUnfinished
		mockTraceProvider := newMockTraceProvider(false, prestate)
		mockLoader := newMockPrestateLoader(false, common.BytesToHash(prestateHash))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.NoError(t, err)
	})

	t.Run("TraceProviderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		mockTraceProvider := newMockTraceProvider(true, prestate)
		mockLoader := newMockPrestateLoader(false, common.BytesToHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.ErrorIs(t, err, mockTraceProviderError)
	})

	t.Run("LoaderErrors", func(t *testing.T) {
		prestate := []byte{0x00, 0x01, 0x02, 0x03}
		mockTraceProvider := newMockTraceProvider(false, prestate)
		mockLoader := newMockPrestateLoader(true, common.BytesToHash(prestate))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.ErrorIs(t, err, mockLoaderError)
	})

	t.Run("PrestateMismatch", func(t *testing.T) {
		mockTraceProvider := newMockTraceProvider(false, []byte{0x00, 0x01, 0x02, 0x03})
		mockLoader := newMockPrestateLoader(false, common.BytesToHash([]byte{0x00}))
		err := ValidateAbsolutePrestate(context.Background(), mockTraceProvider, mockLoader)
		require.Error(t, err)
	})
}

func setupProgressGameTest(t *testing.T, agreeWithProposedRoot bool) (*testlog.CapturingHandler, *GamePlayer, *stubGameState) {
	logger := testlog.Logger(t, log.LvlDebug)
	handler := &testlog.CapturingHandler{
		Delegate: logger.GetHandler(),
	}
	logger.SetHandler(handler)
	gameState := &stubGameState{claimCount: 1}
	game := &GamePlayer{
		act:                     gameState.Act,
		agreeWithProposedOutput: agreeWithProposedRoot,
		loader:                  gameState,
		logger:                  logger,
	}
	return handler, game, gameState
}

type stubGameState struct {
	status     gameTypes.GameStatus
	claimCount uint64
	callCount  int
	actErr     error
	Err        error
}

func (s *stubGameState) Act(ctx context.Context) error {
	s.callCount++
	return s.actErr
}

func (s *stubGameState) GetGameStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	return s.status, nil
}

func (s *stubGameState) GetClaimCount(ctx context.Context) (uint64, error) {
	return s.claimCount, nil
}

type mockTraceProvider struct {
	prestateErrors bool
	prestate       []byte
}

func newMockTraceProvider(prestateErrors bool, prestate []byte) *mockTraceProvider {
	return &mockTraceProvider{
		prestateErrors: prestateErrors,
		prestate:       prestate,
	}
}
func (m *mockTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	panic("not implemented")
}
func (m *mockTraceProvider) GetStepData(ctx context.Context, i uint64) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	panic("not implemented")
}
func (m *mockTraceProvider) AbsolutePreState(ctx context.Context) ([]byte, error) {
	if m.prestateErrors {
		return nil, mockTraceProviderError
	}
	return m.prestate, nil
}
func (m *mockTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (common.Hash, error) {
	prestate, err := m.AbsolutePreState(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	hash := common.BytesToHash(crypto.Keccak256(prestate))
	hash[0] = mipsevm.VMStatusUnfinished
	return hash, nil
}

type mockLoader struct {
	prestateError bool
	prestate      common.Hash
}

func newMockPrestateLoader(prestateError bool, prestate common.Hash) *mockLoader {
	return &mockLoader{
		prestateError: prestateError,
		prestate:      prestate,
	}
}
func (m *mockLoader) FetchAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	if m.prestateError {
		return common.Hash{}, mockLoaderError
	}
	return m.prestate, nil
}
