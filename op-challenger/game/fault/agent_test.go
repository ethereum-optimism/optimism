package fault

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-service/clock"
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

var l1Time = time.UnixMilli(100)

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

func TestDoNotMakeMovesWhenL2BlockNumberChallenged(t *testing.T) {
	ctx := context.Background()

	agent, claimLoader, responder := setupTestAgent(t)
	claimLoader.blockNumChallenged = true

	require.NoError(t, agent.Act(ctx))

	require.Equal(t, 1, responder.callResolveCount, "should check if game is resolvable")
	require.Equal(t, 1, claimLoader.callCount, "should fetch claims only once for resolveClaim")
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

	for _, tCase := range tests {
		tCase := tCase
		t.Run(tCase.name, func(t *testing.T) {
			agent, claimLoader, responder := setupTestAgent(t)
			agent.selective = tCase.selective
			agent.claimants = tCase.claimants
			claimLoader.maxLoads = 1
			if tCase.selective {
				claimLoader.maxLoads = 0
			}
			claimLoader.claims = tCase.claims
			responder.callResolveStatus = tCase.callResolveStatus

			require.NoError(t, agent.Act(ctx))

			require.Equal(t, tCase.expectedResolveCount, responder.callResolveClaimCount, "should check if game is resolvable")
			require.Equal(t, tCase.expectedResolveCount, responder.resolveClaimCount, "should check if game is resolvable")
			if tCase.selective {
				require.Equal(t, 0, responder.callResolveCount, "should not resolve game in selective mode")
				require.Equal(t, 0, responder.resolveCount, "should not resolve game in selective mode")
			}
		})
	}
}

func TestSkipAttemptingToResolveClaimsWhenClockNotExpired(t *testing.T) {
	agent, claimLoader, responder := setupTestAgent(t)
	responder.callResolveErr = errors.New("game is not resolvable")
	responder.callResolveClaimErr = errors.New("claim is not resolvable")
	depth := types.Depth(4)
	claimBuilder := test.NewClaimBuilder(t, depth, alphabet.NewTraceProvider(big.NewInt(0), depth))

	rootTime := l1Time.Add(-agent.maxClockDuration - 5*time.Minute)
	gameBuilder := claimBuilder.GameBuilder(test.WithClock(rootTime, 0))
	gameBuilder.Seq().
		Attack(test.WithClock(rootTime.Add(5*time.Minute), 5*time.Minute)).
		Defend(test.WithClock(rootTime.Add(7*time.Minute), 2*time.Minute)).
		Attack(test.WithClock(rootTime.Add(11*time.Minute), 4*time.Minute))
	claimLoader.claims = gameBuilder.Game.Claims()

	require.NoError(t, agent.Act(context.Background()))

	// Currently tries to resolve the first two claims because their clock's have expired, but doesn't detect that
	// they have unresolvable children.
	require.Equal(t, 2, responder.callResolveClaimCount)
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
	gameDuration := 3 * time.Minute
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(l1Time)
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, 0)
	return agent, claimLoader, responder
}

type stubClaimLoader struct {
	callCount          int
	maxLoads           int
	claims             []types.Claim
	blockNumChallenged bool
}

func (s *stubClaimLoader) IsL2BlockNumberChallenged(_ context.Context, _ rpcblock.Block) (bool, error) {
	return s.blockNumChallenged, nil
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
	resolvedClaims        []uint64
}

func (s *stubResponder) CallResolve(_ context.Context) (gameTypes.GameStatus, error) {
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

func (s *stubResponder) CallResolveClaim(_ context.Context, idx uint64) error {
	s.l.Lock()
	defer s.l.Unlock()
	if slices.Contains(s.resolvedClaims, idx) {
		return errors.New("already resolved")
	}
	s.callResolveClaimCount++
	return s.callResolveClaimErr
}

func (s *stubResponder) ResolveClaims(claims ...uint64) error {
	s.l.Lock()
	defer s.l.Unlock()
	s.resolveClaimCount += len(claims)
	s.resolvedClaims = append(s.resolvedClaims, claims...)
	return nil
}

func (s *stubResponder) PerformAction(_ context.Context, _ types.Action) error {
	return nil
}

// TestResponseDelay tests the response delay functionality using deterministic clock
func TestResponseDelay(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{
			name:  "NoDelay",
			delay: 0,
		},
		{
			name:  "ShortDelay",
			delay: 5 * time.Second,
		},
		{
			name:  "LongDelay",
			delay: 2 * time.Minute,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up agent with deterministic clock
			logger := testlog.Logger(t, log.LevelInfo)
			claimLoader := &stubClaimLoader{}
			depth := types.Depth(4)
			gameDuration := 3 * time.Minute
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
			l1Clock := clock.NewDeterministicClock(l1Time)

			// Create agent with the test response delay
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, test.delay)

			// Set up game state with a claim to respond to
			claimLoader.claims = []types.Claim{
				{
					ClaimData: types.ClaimData{
						Value:    common.Hash{},
						Position: types.NewPositionFromGIndex(big.NewInt(1)),
					},
					Clock: types.Clock{
						Duration:  time.Minute,
						Timestamp: l1Time,
					},
					ContractIndex: 0,
				},
			}

			// Track time before performing action
			startTime := systemClock.Now()

			// Create an action that will trigger the delay
			action := types.Action{
				Type:        types.ActionTypeMove,
				ParentClaim: claimLoader.claims[0],
				IsAttack:    true,
				Value:       common.Hash{0x01},
			}

			// Perform action in a goroutine so we can control clock advancement
			var wg sync.WaitGroup
			wg.Add(1)

			done := make(chan struct{})
			go func() {
				agent.performAction(ctx, &wg, action)
				close(done)
			}()

			// Advance clock by the expected delay amount
			if test.delay > 0 {
				// First select: Verify the action is waiting for the delay (polling check)
				select {
				case <-done:
					t.Fatal("Action completed before delay period")
				case <-time.After(50 * time.Millisecond):
					// Expected - still waiting for delay
				}

				// Advance the deterministic clock by the delay amount
				systemClock.AdvanceTime(test.delay)
			}

			// Second select: Wait for action to complete after clock advancement
			select {
			case <-done:
				// Expected completion
			case <-time.After(500 * time.Millisecond):
				t.Fatal("Action did not complete in reasonable time")
			}

			wg.Wait()

			// Verify the elapsed time matches expected delay
			elapsed := systemClock.Since(startTime)
			require.Equal(t, test.delay, elapsed, "Delay should match expected duration")
		})
	}
}

// TestResponseDelayContextCancellation tests that context cancellation interrupts the delay
func TestResponseDelayContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up agent with long delay and deterministic clock
	logger := testlog.Logger(t, log.LevelInfo)
	claimLoader := &stubClaimLoader{}
	depth := types.Depth(4)
	gameDuration := 3 * time.Minute
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(l1Time)

	longDelay := 5 * time.Minute
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, longDelay)

	// Set up game state
	claimLoader.claims = []types.Claim{
		{
			ClaimData: types.ClaimData{
				Value:    common.Hash{},
				Position: types.NewPositionFromGIndex(big.NewInt(1)),
			},
			Clock: types.Clock{
				Duration:  time.Minute,
				Timestamp: l1Time,
			},
			ContractIndex: 0,
		},
	}

	startTime := systemClock.Now()

	action := types.Action{
		Type:        types.ActionTypeMove,
		ParentClaim: claimLoader.claims[0],
		IsAttack:    true,
		Value:       common.Hash{0x01},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {
		agent.performAction(ctx, &wg, action)
		close(done)
	}()

	// First select: Verify the action is waiting for the delay (polling check)
	select {
	case <-done:
		t.Fatal("Action completed before delay period and cancellation")
	case <-time.After(50 * time.Millisecond):
		// Expected - still waiting for delay
	}

	// Cancel the context (simulates timeout or shutdown)
	cancel()

	// Action should complete quickly after cancellation
	select {
	case <-done:
		// Expected completion due to cancellation
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Action did not complete quickly after cancellation")
	}

	wg.Wait()

	// Verify that elapsed time is much less than the full delay
	elapsed := systemClock.Since(startTime)
	require.Less(t, elapsed, longDelay, "Should not wait full delay when context is cancelled")
	require.Less(t, elapsed, 250*time.Millisecond, "Should complete quickly after cancellation")
}

// TestResponseDelayDifferentActionTypes tests that delay applies to all action types
func TestResponseDelayDifferentActionTypes(t *testing.T) {
	actionTypes := []struct {
		name       string
		actionType types.ActionType
	}{
		{"Move", types.ActionTypeMove},
		{"Step", types.ActionTypeStep},
		{"ChallengeL2BlockNumber", types.ActionTypeChallengeL2BlockNumber},
	}

	for _, actionTest := range actionTypes {
		actionTest := actionTest
		t.Run(actionTest.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up agent with deterministic clock and response delay
			logger := testlog.Logger(t, log.LevelInfo)
			claimLoader := &stubClaimLoader{}
			depth := types.Depth(4)
			gameDuration := 3 * time.Minute
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
			l1Clock := clock.NewDeterministicClock(l1Time)

			responseDelay := 3 * time.Second
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, responseDelay)

			// Set up game state
			claimLoader.claims = []types.Claim{
				{
					ClaimData: types.ClaimData{
						Value:    common.Hash{},
						Position: types.NewPositionFromGIndex(big.NewInt(1)),
					},
					Clock: types.Clock{
						Duration:  time.Minute,
						Timestamp: l1Time,
					},
					ContractIndex: 0,
				},
			}

			startTime := systemClock.Now()

			// Create action of specific type
			action := types.Action{
				Type:        actionTest.actionType,
				ParentClaim: claimLoader.claims[0],
				IsAttack:    true,
				Value:       common.Hash{0x01},
			}

			var wg sync.WaitGroup
			wg.Add(1)

			done := make(chan struct{})
			go func() {
				agent.performAction(ctx, &wg, action)
				close(done)
			}()

			// First select: Verify the action is waiting for the delay (polling check)
			select {
			case <-done:
				t.Fatal("Action completed before delay period")
			case <-time.After(50 * time.Millisecond):
				// Expected - still waiting for delay
			}

			// Advance clock by delay amount
			systemClock.AdvanceTime(responseDelay)

			// Second select: Wait for action to complete after clock advancement
			select {
			case <-done:
				// Expected completion
			case <-time.After(500 * time.Millisecond):
				t.Fatal("Action did not complete after delay")
			}

			wg.Wait()

			// Verify delay was applied correctly for this action type
			elapsed := systemClock.Since(startTime)
			require.Equal(t, responseDelay, elapsed, "Delay should apply to action type %s", actionTest.name)
		})
	}
}
