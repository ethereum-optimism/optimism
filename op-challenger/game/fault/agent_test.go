package fault

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
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

func createClaimsWithClaimants(t *testing.T, d types.Depth) []types.Claim {
	claimBuilder := test.NewClaimBuilder(t, d, alphabet.NewTraceProvider(big.NewInt(0), d))
	rootClaim := claimBuilder.CreateRootClaim()
	claim1 := rootClaim
	claim1.Claimant = common.BigToAddress(big.NewInt(1))
	claim2 := claimBuilder.AttackClaim(claim1)
	claim2.Claimant = common.BigToAddress(big.NewInt(2))
	claim3 := claimBuilder.AttackClaim(claim2)
	claim3.Claimant = common.BigToAddress(big.NewInt(3))
	return []types.Claim{claim1, claim2, claim3}
}

func TestAgent_SelectiveClaimResolution(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                 string
		callResolveStatus    gameTypes.GameStatus
		selective            bool
		claimants            []common.Address
		claims               []types.Claim
		expectedResolveCount int
	}{
		{
			name:                 "NonSelectiveEmptyClaimants",
			callResolveStatus:    gameTypes.GameStatusDefenderWon,
			selective:            false,
			claimants:            []common.Address{},
			claims:               createClaimsWithClaimants(t, types.Depth(4)),
			expectedResolveCount: 3,
		},
		{
			name:                 "NonSelectiveWithClaimants",
			callResolveStatus:    gameTypes.GameStatusDefenderWon,
			selective:            false,
			claimants:            []common.Address{common.BigToAddress(big.NewInt(1))},
			claims:               createClaimsWithClaimants(t, types.Depth(4)),
			expectedResolveCount: 3,
		},
		{
			name:              "SelectiveEmptyClaimants",
			callResolveStatus: gameTypes.GameStatusDefenderWon,
			selective:         true,
			claimants:         []common.Address{},
			claims:            createClaimsWithClaimants(t, types.Depth(4)),
		},
		{
			name:                 "SelectiveWithClaimants",
			callResolveStatus:    gameTypes.GameStatusDefenderWon,
			selective:            true,
			claimants:            []common.Address{common.BigToAddress(big.NewInt(1))},
			claims:               createClaimsWithClaimants(t, types.Depth(4)),
			expectedResolveCount: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			agent, claimLoader, responder := setupTestAgent(t)
			agent.selective = test.selective
			agent.claimants = test.claimants
			claimLoader.maxLoads = 1
			claimLoader.claims = test.claims
			responder.callResolveStatus = test.callResolveStatus

			require.NoError(t, agent.Act(ctx))

			require.Equal(t, test.expectedResolveCount, responder.callResolveClaimCount, "should check if game is resolvable")
			require.Equal(t, test.expectedResolveCount, responder.resolveClaimCount, "should check if game is resolvable")
		})
	}
}

func TestLoadClaimsWhenGameNotResolvable(t *testing.T) {
	// Checks that if the game isn't resolvable, that the agent continues on to start checking claims
	agent, claimLoader, responder := setupTestAgent(t)
	responder.callResolveErr = errors.New("game is not resolvable")
	responder.callResolveClaimErr = errors.New("claim is not resolvable")
	depth := types.Depth(4)
	claimBuilder := test.NewClaimBuilder(t, depth, alphabet.NewTraceProvider(big.NewInt(0), depth))

	claimLoader.claims = []types.Claim{
		claimBuilder.CreateRootClaim(),
	}

	require.NoError(t, agent.Act(context.Background()))

	require.EqualValues(t, 2, claimLoader.callCount, "should load claims for unresolvable game")
	require.EqualValues(t, responder.callResolveClaimCount, 1, "should check if claim is resolvable")
	require.Zero(t, responder.resolveClaimCount, "should not send resolveClaim")
}

func setupTestAgent(t *testing.T) (*Agent, *stubClaimLoader, *stubResponder) {
	logger := testlog.Logger(t, log.LevelInfo)
	claimLoader := &stubClaimLoader{}
	depth := types.Depth(4)
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	agent := NewAgent(metrics.NoopMetrics, claimLoader, depth, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{})
	return agent, claimLoader, responder
}

type stubClaimLoader struct {
	callCount int
	maxLoads  int
	claims    []types.Claim
}

func (s *stubClaimLoader) GetAllClaims(_ context.Context, _ rpcblock.Block) ([]types.Claim, error) {
	s.callCount++
	if s.callCount > s.maxLoads && s.maxLoads != 0 {
		return []types.Claim{}, nil
	}
	return s.claims, nil
}

type stubResponder struct {
	l                 sync.Mutex
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
	s.l.Lock()
	defer s.l.Unlock()
	s.callResolveCount++
	return s.callResolveStatus, s.callResolveErr
}

func (s *stubResponder) Resolve() error {
	s.l.Lock()
	defer s.l.Unlock()
	s.resolveCount++
	return s.resolveErr
}

func (s *stubResponder) CallResolveClaim(ctx context.Context, clainIdx uint64) error {
	s.l.Lock()
	defer s.l.Unlock()
	s.callResolveClaimCount++
	return s.callResolveClaimErr
}

func (s *stubResponder) ResolveClaim(clainIdx uint64) error {
	s.l.Lock()
	defer s.l.Unlock()
	s.resolveClaimCount++
	return nil
}

func (s *stubResponder) PerformAction(ctx context.Context, response types.Action) error {
	return nil
}
