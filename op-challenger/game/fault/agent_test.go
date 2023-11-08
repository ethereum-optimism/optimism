package fault

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestShouldResolve tests the resolution logic.
func TestShouldResolve(t *testing.T) {
	t.Run("AgreeWithProposedOutput", func(t *testing.T) {
		agent, _, _ := setupTestAgent(t, true)
		require.False(t, agent.shouldResolve(gameTypes.GameStatusDefenderWon))
		require.True(t, agent.shouldResolve(gameTypes.GameStatusChallengerWon))
		require.False(t, agent.shouldResolve(gameTypes.GameStatusInProgress))
	})

	t.Run("DisagreeWithProposedOutput", func(t *testing.T) {
		agent, _, _ := setupTestAgent(t, false)
		require.True(t, agent.shouldResolve(gameTypes.GameStatusDefenderWon))
		require.False(t, agent.shouldResolve(gameTypes.GameStatusChallengerWon))
		require.False(t, agent.shouldResolve(gameTypes.GameStatusInProgress))
	})
}

func TestDoNotMakeMovesWhenGameIsResolvable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                    string
		agreeWithProposedOutput bool
		callResolveStatus       gameTypes.GameStatus
		shouldResolve           bool
	}{
		{
			name:                    "Agree_Losing",
			agreeWithProposedOutput: true,
			callResolveStatus:       gameTypes.GameStatusDefenderWon,
			shouldResolve:           false,
		},
		{
			name:                    "Agree_Winning",
			agreeWithProposedOutput: true,
			callResolveStatus:       gameTypes.GameStatusChallengerWon,
			shouldResolve:           true,
		},
		{
			name:                    "Disagree_Losing",
			agreeWithProposedOutput: false,
			callResolveStatus:       gameTypes.GameStatusChallengerWon,
			shouldResolve:           false,
		},
		{
			name:                    "Disagree_Winning",
			agreeWithProposedOutput: false,
			callResolveStatus:       gameTypes.GameStatusDefenderWon,
			shouldResolve:           true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			agent, claimLoader, responder := setupTestAgent(t, test.agreeWithProposedOutput)
			responder.callResolveStatus = test.callResolveStatus

			require.NoError(t, agent.Act(ctx))

			require.Equal(t, 1, responder.callResolveCount, "should check if game is resolvable")
			require.Equal(t, 1, claimLoader.callCount, "should fetch claims once for resolveClaim")

			if test.shouldResolve {
				require.EqualValues(t, 1, responder.resolveCount, "should resolve winning game")
			} else {
				require.Zero(t, responder.resolveCount, "should not resolve losing game")
			}
		})
	}
}

func TestLoadClaimsWhenGameNotResolvable(t *testing.T) {
	// Checks that if the game isn't resolvable, that the agent continues on to start checking claims
	agent, claimLoader, responder := setupTestAgent(t, false)
	responder.callResolveErr = errors.New("game is not resolvable")
	responder.callResolveClaimErr = errors.New("claim is not resolvable")
	depth := 4
	claimBuilder := test.NewClaimBuilder(t, depth, alphabet.NewTraceProvider("abcdefg", uint64(depth)))

	claimLoader.claims = []types.Claim{
		claimBuilder.CreateRootClaim(true),
	}

	require.NoError(t, agent.Act(context.Background()))

	require.EqualValues(t, 2, claimLoader.callCount, "should load claims for unresolvable game")
	require.EqualValues(t, responder.callResolveClaimCount, 1, "should check if claim is resolvable")
	require.Zero(t, responder.resolveClaimCount, "should not send resolveClaim")
}

func setupTestAgent(t *testing.T, agreeWithProposedOutput bool) (*Agent, *stubClaimLoader, *stubResponder) {
	logger := testlog.Logger(t, log.LvlInfo)
	claimLoader := &stubClaimLoader{}
	depth := 4
	trace := alphabet.NewTraceProvider("abcd", uint64(depth))
	responder := &stubResponder{}
	updater := &stubUpdater{}
	agent := NewAgent(metrics.NoopMetrics, claimLoader, depth, trace, responder, updater, agreeWithProposedOutput, logger)
	return agent, claimLoader, responder
}

type stubClaimLoader struct {
	callCount int
	claims    []types.Claim
}

func (s *stubClaimLoader) GetAllClaims(ctx context.Context) ([]types.Claim, error) {
	s.callCount++
	return s.claims, nil
}

type stubResponder struct {
	callResolveCount  int
	callResolveStatus gameTypes.GameStatus
	callResolveErr    error

	resolveCount int
	resolveErr   error

	callResolveClaimCount int
	callResolveClaimErr   error
	resolveClaimCount     int
}

func (s *stubResponder) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	s.callResolveCount++
	return s.callResolveStatus, s.callResolveErr
}

func (s *stubResponder) Resolve(ctx context.Context) error {
	s.resolveCount++
	return s.resolveErr
}

func (s *stubResponder) CallResolveClaim(ctx context.Context, clainIdx uint64) error {
	s.callResolveClaimCount++
	return s.callResolveClaimErr
}

func (s *stubResponder) ResolveClaim(ctx context.Context, clainIdx uint64) error {
	s.resolveClaimCount++
	return nil
}

func (s *stubResponder) PerformAction(ctx context.Context, response types.Action) error {
	return nil
}

type stubUpdater struct {
}

func (s *stubUpdater) UpdateOracle(ctx context.Context, data *types.PreimageOracleData) error {
	panic("Not implemented")
}
