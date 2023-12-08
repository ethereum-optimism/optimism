package fault

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestDoNotMakeMovesWhenGameIsResolvable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		callResolveStatus gameTypes.GameStatus
	}{
		{
			name:              "DefenderWon",
			callResolveStatus: gameTypes.GameStatusDefenderWon,
		},
		{
			name:              "ChallengerWon",
			callResolveStatus: gameTypes.GameStatusChallengerWon,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			agent, claimLoader, responder := setupTestAgent(t)
			responder.callResolveStatus = test.callResolveStatus

			require.NoError(t, agent.Act(ctx))

			require.Equal(t, 1, responder.callResolveCount, "should check if game is resolvable")
			require.Equal(t, 1, claimLoader.callCount, "should fetch claims once for resolveClaim")

			require.EqualValues(t, 1, responder.resolveCount, "should resolve winning game")
		})
	}
}

func TestLoadClaimsWhenGameNotResolvable(t *testing.T) {
	// Checks that if the game isn't resolvable, that the agent continues on to start checking claims
	agent, claimLoader, responder := setupTestAgent(t)
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

func setupTestAgent(t *testing.T) (*Agent, *stubClaimLoader, *stubResponder) {
	logger := testlog.Logger(t, log.LvlInfo)
	claimLoader := &stubClaimLoader{}
	depth := 4
	provider := alphabet.NewTraceProvider("abcd", uint64(depth))
	responder := &stubResponder{}
	agent := NewAgent(metrics.NoopMetrics, claimLoader, depth, trace.NewSimpleTraceAccessor(provider), responder, logger)
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
