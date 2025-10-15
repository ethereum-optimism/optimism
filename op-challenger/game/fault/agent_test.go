package fault

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	faulttest "github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var l1Time = time.UnixMilli(100)

// newStubClaimLoaderWithDefaults creates a stubClaimLoader with sensible defaults
// for basic delay tests (prevents clock extension from triggering)
func newStubClaimLoaderWithDefaults() *stubClaimLoader {
	return &stubClaimLoader{
		// A large clock extension value used to prevent clock
		// extension from triggering during basic delay tests
		clockExtension: 1 * time.Hour,
	}
}

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
	claimBuilder := faulttest.NewClaimBuilder(t, d, alphabet.NewTraceProvider(big.NewInt(0), d))
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
	claimBuilder := faulttest.NewClaimBuilder(t, depth, alphabet.NewTraceProvider(big.NewInt(0), depth))

	rootTime := l1Time.Add(-agent.maxClockDuration - 5*time.Minute)
	gameBuilder := claimBuilder.GameBuilder(faulttest.WithClock(rootTime, 0))
	gameBuilder.Seq().
		Attack(faulttest.WithClock(rootTime.Add(5*time.Minute), 5*time.Minute)).
		Defend(faulttest.WithClock(rootTime.Add(7*time.Minute), 2*time.Minute)).
		Attack(faulttest.WithClock(rootTime.Add(11*time.Minute), 4*time.Minute))
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
	claimBuilder := faulttest.NewClaimBuilder(t, depth, alphabet.NewTraceProvider(big.NewInt(0), depth))

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
	gameDuration := 24 * time.Hour
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(l1Time)
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, 0, 0)
	return agent, claimLoader, responder
}

type stubClaimLoader struct {
	callCount          int
	maxLoads           int
	claims             []types.Claim
	blockNumChallenged bool
	clockExtension     time.Duration
	clockExtensionErr  error
	splitDepth         types.Depth
	maxGameDepth       types.Depth
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

func (s *stubClaimLoader) GetClockExtension(_ context.Context) (time.Duration, error) {
	if s.clockExtensionErr != nil {
		return 0, s.clockExtensionErr
	}
	// Return a reasonable default if not set
	if s.clockExtension == 0 {
		return 5 * time.Minute, nil // Default clock extension
	}
	return s.clockExtension, nil
}

func (s *stubClaimLoader) GetSplitDepth(_ context.Context) (types.Depth, error) {
	if s.splitDepth != 0 {
		return s.splitDepth, nil
	}
	return types.Depth(30), nil // Reasonable default for tests
}

func (s *stubClaimLoader) GetMaxGameDepth(_ context.Context) (types.Depth, error) {
	if s.maxGameDepth != 0 {
		return s.maxGameDepth, nil
	}
	return types.Depth(73), nil // Reasonable default for tests
}

func (s *stubClaimLoader) GetOracle(_ context.Context) (contracts.PreimageOracleContract, error) {
	return &stubPreimageOracleContract{}, nil
}

// stubPreimageOracleContract implements the PreimageOracleContract interface for testing
type stubPreimageOracleContract struct{}

func (s *stubPreimageOracleContract) ChallengePeriod(_ context.Context) (uint64, error) {
	return 86400, nil // 1 day in seconds - reasonable default for tests
}

// Add minimal implementations for other required methods (if any)
func (s *stubPreimageOracleContract) Addr() common.Address { return common.Address{} }
func (s *stubPreimageOracleContract) AddGlobalDataTx(*types.PreimageOracleData) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *stubPreimageOracleContract) InitLargePreimage(*big.Int, uint32, uint32) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *stubPreimageOracleContract) AddLeaves(*big.Int, *big.Int, []byte, []common.Hash, bool) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *stubPreimageOracleContract) MinLargePreimageSize(context.Context) (uint64, error) {
	return 0, nil
}
func (s *stubPreimageOracleContract) CallSqueeze(context.Context, common.Address, *big.Int, keccakTypes.StateSnapshot, keccakTypes.Leaf, merkle.Proof, keccakTypes.Leaf, merkle.Proof) error {
	return nil
}
func (s *stubPreimageOracleContract) Squeeze(common.Address, *big.Int, keccakTypes.StateSnapshot, keccakTypes.Leaf, merkle.Proof, keccakTypes.Leaf, merkle.Proof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *stubPreimageOracleContract) GetActivePreimages(context.Context, common.Hash) ([]keccakTypes.LargePreimageMetaData, error) {
	return nil, nil
}
func (s *stubPreimageOracleContract) GetProposalMetadata(context.Context, rpcblock.Block, ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error) {
	return nil, nil
}
func (s *stubPreimageOracleContract) GetProposalTreeRoot(context.Context, rpcblock.Block, keccakTypes.LargePreimageIdent) (common.Hash, error) {
	return common.Hash{}, nil
}
func (s *stubPreimageOracleContract) GetInputDataBlocks(context.Context, rpcblock.Block, keccakTypes.LargePreimageIdent) ([]uint64, error) {
	return nil, nil
}
func (s *stubPreimageOracleContract) DecodeInputData([]byte) (*big.Int, keccakTypes.InputData, error) {
	return nil, keccakTypes.InputData{}, nil
}
func (s *stubPreimageOracleContract) GlobalDataExists(context.Context, *types.PreimageOracleData) (bool, error) {
	return false, nil
}
func (s *stubPreimageOracleContract) GetGlobalData(context.Context, *types.PreimageOracleData) ([32]byte, error) {
	return [32]byte{}, nil
}
func (s *stubPreimageOracleContract) ChallengeTx(keccakTypes.LargePreimageIdent, keccakTypes.Challenge) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *stubPreimageOracleContract) GetMinBondLPP(context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

// createStubGame creates a mock game for testing performAction calls
func createStubGame(claims []types.Claim) types.Game {
	if len(claims) == 0 {
		// Create a default root claim for tests
		claims = []types.Claim{
			faulttest.NewClaimBuilder(nil, types.Depth(4), alphabet.NewTraceProvider(big.NewInt(0), types.Depth(4))).CreateRootClaim(),
		}
	}
	return types.NewGameState(claims, types.Depth(4))
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

	performActionCount int
	performActionErr   error // If set, PerformAction will return this error
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
	s.l.Lock()
	defer s.l.Unlock()
	s.performActionCount++
	return s.performActionErr
}

func (s *stubResponder) PerformedActionCount() int {
	s.l.Lock()
	defer s.l.Unlock()
	return s.performActionCount
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
			name:  "Delay",
			delay: 20 * time.Hour, // Less than extension threshold (24h - 1h = 23h)
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up agent with deterministic clock
			logger := testlog.Logger(t, log.LevelInfo)
			claimLoader := newStubClaimLoaderWithDefaults()
			depth := types.Depth(4)
			gameDuration := 24 * time.Hour // Large value to avoid clock extension triggering
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
			l1Clock := clock.NewDeterministicClock(l1Time)

			// Create agent with the test response delay
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, test.delay, 0)

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
				agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
				close(done)
			}()

			if test.delay > 0 {
				// Wait for the action delay to begin waiting
				require.True(t, systemClock.WaitForNewPendingTaskWithTimeout(30*time.Second))
				require.Zero(t, responder.PerformedActionCount(), "Action should not have completed before delay period")

				systemClock.AdvanceTime(test.delay)
			}
			// Verify the action completes
			select {
			case <-done:
				// Expected completion due to cancellation
			case <-time.After(30 * time.Second):
				t.Fatal("Action did not complete quickly after cancellation")
			}
			// And verify the wait group is done for good measure
			wg.Wait()

			require.Equal(t, 1, responder.PerformedActionCount(), "Action should have completed after delay period")
		})
	}
}

// TestResponseDelayContextCancellation tests that context cancellation interrupts the delay
func TestResponseDelayContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up agent with long delay and deterministic clock
	logger := testlog.Logger(t, log.LevelInfo)
	claimLoader := newStubClaimLoaderWithDefaults()
	depth := types.Depth(4)
	gameDuration := 24 * time.Hour
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(l1Time)

	longDelay := 5 * time.Minute
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, longDelay, 0)

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
		agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
		close(done)
	}()

	// Verify the action is waiting for the delay
	systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)

	// Cancel the context (simulates timeout or shutdown)
	cancel()

	// Action should complete even though the clock didn't progress
	select {
	case <-done:
		// Expected completion due to cancellation
	case <-time.After(30 * time.Second):
		t.Fatal("Action did not complete quickly after cancellation")
	}

	// And verify the wait group is done for good measure
	wg.Wait()
	require.Zero(t, responder.PerformedActionCount(), "Action should not have completed")
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
			claimLoader := newStubClaimLoaderWithDefaults()
			depth := types.Depth(4)
			gameDuration := 24 * time.Hour // Large value to avoid clock extension triggering
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
			l1Clock := clock.NewDeterministicClock(l1Time)

			responseDelay := 3 * time.Hour
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, responseDelay, 0)

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
				agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
				close(done)
			}()

			// First select: Verify the action is waiting for the delay (polling check)
			systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)
			require.Zero(t, responder.PerformedActionCount(), "Action was performed before delay")

			// Advance clock by delay amount
			systemClock.AdvanceTime(responseDelay)

			// Second select: Wait for action to complete after clock advancement
			select {
			case <-done:
				// Expected completion
			case <-time.After(30 * time.Second):
				t.Fatal("Action did not complete after delay")
			}
			// Verify the wait group is done for good measure
			wg.Wait()

			// Verify the action was performed
			require.Equal(t, 1, responder.PerformedActionCount(), "Action was not performed after delay")
		})
	}
}

// TestResponseDelayAfter tests the response delay activation threshold functionality
func TestResponseDelayAfter(t *testing.T) {
	tests := []struct {
		name               string
		responseDelay      time.Duration
		responseDelayAfter uint64
		actionsToPerform   int
	}{
		{
			name:               "DelayFromFirstResponse",
			responseDelay:      2 * time.Hour,
			responseDelayAfter: 0, // Apply delay from first response
			actionsToPerform:   3,
		},
		{
			name:               "DelayAfterFirstResponse",
			responseDelay:      2 * time.Hour,
			responseDelayAfter: 1, // Skip first response, delay subsequent ones
			actionsToPerform:   3,
		},
		{
			name:               "DelayAfterSecondResponse",
			responseDelay:      2 * time.Hour,
			responseDelayAfter: 2, // Skip first two responses
			actionsToPerform:   4,
		},
		{
			name:               "DelayNeverActivates",
			responseDelay:      2 * time.Hour,
			responseDelayAfter: 5, // Threshold higher than actions performed
			actionsToPerform:   3,
		},
		{
			name:               "NoDelayConfigured",
			responseDelay:      0, // No delay configured
			responseDelayAfter: 0,
			actionsToPerform:   3,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up agent with deterministic clock
			logger := testlog.Logger(t, log.LevelInfo)
			claimLoader := newStubClaimLoaderWithDefaults()
			depth := types.Depth(4)
			gameDuration := 24 * time.Hour // Large value to avoid clock extension triggering
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
			l1Clock := clock.NewDeterministicClock(l1Time)

			// Create agent with test parameters
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, test.responseDelay, test.responseDelayAfter)

			// Set up initial game state
			claimBuilder := faulttest.NewClaimBuilder(t, depth, provider)
			baseClaim := claimBuilder.CreateRootClaim()
			// Fix timestamp to be realistic
			baseClaim.Clock = types.Clock{
				Duration:  0,             // Root claim starts with no accumulated time
				Timestamp: l1Clock.Now(), // Use current time
			}
			claimLoader.claims = []types.Claim{baseClaim}

			// Perform actions and verify delay behavior
			for i := 0; i < test.actionsToPerform; i++ {
				action := types.Action{
					Type:        types.ActionTypeMove,
					ParentClaim: baseClaim,
					IsAttack:    true,
					Value:       common.Hash{byte(i + 1)}, // Unique value for each action
				}

				var wg sync.WaitGroup
				wg.Add(1)

				done := make(chan struct{})
				go func() {
					agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
					close(done)
				}()

				// Calculate if delay should be applied: response count >= threshold AND delay > 0
				shouldHaveDelay := uint64(i) >= test.responseDelayAfter && test.responseDelay > 0

				if shouldHaveDelay {
					systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)
					require.Equal(t, i, responder.PerformedActionCount(), "Action was performed before delay")

					// Advance clock by delay amount
					systemClock.AdvanceTime(test.responseDelay)
				}

				// Wait for completion
				select {
				case <-done:
					// Expected completion
				case <-time.After(30 * time.Second):
					t.Fatalf("Action %d did not complete after delay", i+1)
				}
				wg.Wait()

				// Verify response count incremented (assuming successful response)
				expectedCount := uint64(i + 1)
				require.Equal(t, expectedCount, agent.responseCount.Load(), "Response count should increment after action %d", expectedCount)
			}
		})
	}
}

// TestResponseDelayAfterWithFailedActions tests that failed actions don't increment response count
func TestResponseDelayAfterWithFailedActions(t *testing.T) {
	ctx := context.Background()

	// Set up agent with delay after 1 response
	logger := testlog.Logger(t, log.LevelInfo)
	claimLoader := newStubClaimLoaderWithDefaults()
	depth := types.Depth(4)
	gameDuration := 24 * time.Hour
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(l1Time)

	responseDelay := 2 * time.Hour
	responseDelayAfter := uint64(1) // Delay after first successful response
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, gameDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, responseDelay, responseDelayAfter)

	// Set up game state
	claimBuilder := faulttest.NewClaimBuilder(t, depth, provider)
	baseClaim := claimBuilder.CreateRootClaim()
	// Fix timestamp to be realistic
	baseClaim.Clock = types.Clock{
		Duration:  0,             // Root claim starts with no accumulated time
		Timestamp: l1Clock.Now(), // Use current time
	}
	claimLoader.claims = []types.Claim{baseClaim}

	action := types.Action{
		Type:        types.ActionTypeMove,
		ParentClaim: baseClaim,
		IsAttack:    true,
		Value:       common.Hash{0x01},
	}

	// First action: make responder fail
	responder.performActionErr = errors.New("simulated action failure")

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {
		agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
		close(done)
	}()

	// Should complete without needing to advance the clock (no delay since responseCount < responseDelayAfter)
	select {
	case <-done:
		// Expected immediate completion
	case <-time.After(30 * time.Second):
		t.Fatal("Failed action took too long")
	}
	wg.Wait()

	require.Equal(t, uint64(0), agent.responseCount.Load(), "Failed action should not increment response count")

	// Second action: make responder succeed
	responder.performActionErr = nil

	wg.Add(1)
	done = make(chan struct{})
	go func() {
		agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
		close(done)
	}()

	// Should complete without needing to advance the clock (no delay since responseCount is still 0)
	select {
	case <-done:
		// Expected immediate completion
	case <-time.After(30 * time.Second):
		t.Fatal("Successful action took too long")
	}
	wg.Wait()

	// Should be no delay but response count should increment
	require.Equal(t, uint64(1), agent.responseCount.Load(), "Successful action should increment response count")

	// Third action: should now have delay applied
	wg.Add(1)
	done = make(chan struct{})
	go func() {
		agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
		close(done)
	}()

	// Should be waiting for delay now (responseCount >= responseDelayAfter)
	systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)
	// Note: 2 attempts have been made - one failed, one successful and the third is delayed.
	require.Equal(t, 2, responder.PerformedActionCount(), "Should not have performed action without delay")

	// Advance clock by delay amount
	systemClock.AdvanceTime(responseDelay)

	// Wait for completion
	select {
	case <-done:
		// Expected completion
	case <-time.After(30 * time.Second):
		t.Fatal("Action did not complete after delay")
	}

	wg.Wait()

	require.Equal(t, 3, responder.PerformedActionCount(), "Should have performed action after delay")
	require.Equal(t, uint64(2), agent.responseCount.Load(), "Response count should be 2 after second successful action")
}

// TestResponseDelayClockExtension tests that delays are skipped during clock extension periods
func TestResponseDelayClockExtension(t *testing.T) {
	// Common test configuration
	const (
		responseDelay      = 30 * time.Second // Reasonable delay that fits in remaining time
		responseDelayAfter = 0
		maxClockDuration   = 10 * time.Minute
		clockExtension     = 1 * time.Minute
		baseTimestamp      = 100000 // milliseconds since Unix epoch
	)
	extensionThreshold := maxClockDuration - clockExtension // 9 minutes

	tests := []struct {
		name                string
		parentClockDuration time.Duration // Previous accumulated time
		timeSinceCreation   time.Duration // Additional time since claim created
	}{
		{
			name:                "NoExtension_WithDelay",
			parentClockDuration: 3 * time.Minute,
			timeSinceCreation:   1 * time.Minute, // Total: 4min < 9min threshold
		},
		{
			name:                "InExtension_SkipDelay",
			parentClockDuration: 8 * time.Minute,
			timeSinceCreation:   2 * time.Minute, // Total: 10min > 9min threshold
		},
		{
			name:                "ExactlyAtThreshold_InExtension_SkipDelay",
			parentClockDuration: 8 * time.Minute,
			timeSinceCreation:   1*time.Minute + 1*time.Microsecond, // Total: just over 9min
		},
		{
			name:                "JustBelowThreshold_WithDelay_WaitDelay",
			parentClockDuration: 8 * time.Minute,
			timeSinceCreation:   20 * time.Second, // Total: 8min20s + 30s delay = 8min50s < 9min threshold
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			// Set up agent with deterministic clock
			logger := testlog.Logger(t, log.LevelInfo)
			claimLoader := &stubClaimLoader{
				clockExtension: clockExtension,
			}
			depth := types.Depth(4)
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}
			currentTime := time.UnixMilli(baseTimestamp).Add(test.timeSinceCreation)
			systemClock := clock.NewDeterministicClock(currentTime)
			l1Clock := clock.NewDeterministicClock(currentTime)

			// Create agent with test parameters
			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, maxClockDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, responseDelay, responseDelayAfter)

			// Set up proper parent-child relationship for chess clock calculation
			claimBuilder := faulttest.NewClaimBuilder(t, depth, provider)

			// Create a grandparent claim (root claim) that has the accumulated time from previous moves
			grandparentClaim := claimBuilder.CreateRootClaim(faulttest.WithClock(
				currentTime.Add(-test.timeSinceCreation).Add(-time.Duration(test.parentClockDuration.Nanoseconds())),
				test.parentClockDuration,
			))
			grandparentClaim.ContractIndex = 0 // Root claim

			// Create parent claim as an attack on the grandparent (so it's NOT a root claim)
			parentClaim := claimBuilder.AttackClaim(grandparentClaim, faulttest.WithClock(
				currentTime.Add(-test.timeSinceCreation),
				0, // This will be calculated by ChessClock
			))
			parentClaim.ContractIndex = 1 // Set contract index

			// Calculate total chess clock time using the same logic as the contract
			// This should be grandparent.Duration + time since parent was created
			totalChessClockTime := test.parentClockDuration + test.timeSinceCreation
			expectDelay := totalChessClockTime <= extensionThreshold
			claimLoader.claims = []types.Claim{grandparentClaim, parentClaim}

			// Create action with the parent claim
			action := types.Action{
				Type:        types.ActionTypeMove,
				ParentClaim: parentClaim,
				IsAttack:    true,
				Value:       common.Hash{0x01},
			}

			// Perform action and measure timing
			var wg sync.WaitGroup
			wg.Add(1)

			done := make(chan struct{})
			go func() {
				agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
				close(done)
			}()

			if expectDelay {
				// Should be waiting for delay
				systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)
				require.Equal(t, 0, responder.PerformedActionCount(), "Should not have performed action without delay")

				// Advance clock by delay amount
				systemClock.AdvanceTime(responseDelay)
			}

			// Wait for completion
			select {
			case <-done:
				// Expected completion
			case <-time.After(30 * time.Second):
				t.Fatal("Action did not complete in expected time")
			}
			wg.Wait()

			require.Equal(t, 1, responder.PerformedActionCount(), "Should have performed action after delay")
		})
	}
}

// TestResponseDelayTimeoutPrevention tests delay timeout prevention logic
func TestResponseDelayTimeoutPrevention(t *testing.T) {
	const (
		responseDelayAfter = 0
		maxClockDuration   = 10 * time.Minute
		clockExtension     = 2 * time.Minute
	)

	tests := []struct {
		name                string
		parentClockDuration time.Duration
		responseDelay       time.Duration
		expectDelay         bool
		description         string
	}{
		{
			name:                "DelayFitsInExtensionBuffer_ShouldSkip",
			parentClockDuration: 8*time.Minute + 30*time.Second, // Past threshold but delay fits
			responseDelay:       1 * time.Minute,                // Fits in 2min extension
			expectDelay:         false,                          // Should skip due to extension period
			description:         "When in extension period, should skip delay regardless of timeout risk",
		},
		{
			name:                "DelayWouldTimeout_ShouldSkip",
			parentClockDuration: 9*time.Minute + 30*time.Second, // Already in extension (threshold 8min)
			responseDelay:       3 * time.Minute,                // Large delay
			expectDelay:         false,                          // Should skip due to being in extension
			description:         "Should skip delay when already in extension period",
		},
		{
			name:                "DelayWouldEnterExtensionPeriod_ShouldSkip",
			parentClockDuration: 6 * time.Minute, // Not in extension (8min threshold)
			responseDelay:       3 * time.Minute, // Would push us to 9min > 8min threshold
			expectDelay:         false,           // Should skip to avoid extension period
			description:         "Should skip delay when it would cause entry into extension period",
		},
		{
			name:                "BeforeThreshold_ShouldDelay",
			parentClockDuration: 5 * time.Minute,  // Well before threshold, 5min remaining
			responseDelay:       30 * time.Second, // Short delay that fits in remaining time
			expectDelay:         true,             // Should apply delay
			description:         "Should apply delay when well before extension threshold and delay fits",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			logger := testlog.Logger(t, log.LevelInfo)

			claimLoader := &stubClaimLoader{
				clockExtension: clockExtension,
			}
			depth := types.Depth(4)
			provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
			responder := &stubResponder{}

			// Set up timing so parentClockDuration calculation works
			currentTime := time.UnixMilli(100000)
			systemClock := clock.NewDeterministicClock(currentTime)
			l1Clock := clock.NewDeterministicClock(currentTime)

			agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, maxClockDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, test.responseDelay, responseDelayAfter)

			// Create claims with proper parent-child relationship for chess clock calculation
			claimBuilder := faulttest.NewClaimBuilder(t, depth, provider)
			timeSinceCreation := 1 * time.Minute // Fixed component

			// Create grandparent claim (root claim) that has the accumulated time from previous moves
			grandparentClaim := claimBuilder.CreateRootClaim(faulttest.WithClock(
				currentTime.Add(-timeSinceCreation).Add(-time.Duration(test.parentClockDuration.Nanoseconds())),
				test.parentClockDuration,
			))
			grandparentClaim.ContractIndex = 0 // Root claim

			// Create parent claim as an attack on the grandparent (so it's NOT a root claim)
			parentClaim := claimBuilder.AttackClaim(grandparentClaim, faulttest.WithClock(
				currentTime.Add(-timeSinceCreation),
				0, // This will be calculated by ChessClock
			))
			parentClaim.ContractIndex = 1 // Set contract index

			claimLoader.claims = []types.Claim{grandparentClaim, parentClaim}

			action := types.Action{
				Type:        types.ActionTypeMove,
				ParentClaim: parentClaim,
				IsAttack:    true,
				Value:       common.Hash{0x01},
			}

			// Perform action and check timing
			var wg sync.WaitGroup
			wg.Add(1)
			done := make(chan struct{})
			go func() {
				agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
				close(done)
			}()

			if test.expectDelay {
				// Should wait for delay
				systemClock.WaitForNewPendingTaskWithTimeout(30 * time.Second)
				require.Equal(t, 0, responder.PerformedActionCount(), "Should be waiting for delay")

				// Advance clock and complete
				systemClock.AdvanceTime(test.responseDelay)
			}

			// Wait for completion - using longer timeout for CI reliability
			select {
			case <-done:
				// Expected completion
			case <-time.After(30 * time.Second):
				t.Fatal("Action did not complete - this indicates a test logic error")
			}
			wg.Wait()

			require.Equal(t, 1, responder.PerformedActionCount(), test.description)
		})
	}
}

// TestResponseDelayClockExtensionError tests error handling when clock extension detection fails
func TestResponseDelayClockExtensionError(t *testing.T) {
	ctx := context.Background()

	// Set up agent with claimLoader that returns an error for clock extension
	logger := testlog.Logger(t, log.LevelInfo)
	claimLoader := &stubClaimLoader{
		clockExtensionErr: errors.New("failed to get clock extension"),
	}
	depth := types.Depth(4)
	provider := alphabet.NewTraceProvider(big.NewInt(0), depth)
	responder := &stubResponder{}
	systemClock := clock.NewDeterministicClock(time.UnixMilli(120200))
	l1Clock := clock.NewDeterministicClock(time.UnixMilli(120200))

	responseDelay := 2 * time.Hour
	maxClockDuration := 10 * time.Minute // Use a reasonable default for error test
	agent := NewAgent(metrics.NoopMetrics, systemClock, l1Clock, claimLoader, depth, maxClockDuration, trace.NewSimpleTraceAccessor(provider), responder, logger, false, []common.Address{}, responseDelay, 0)

	// Set up game state
	claimBuilder := faulttest.NewClaimBuilder(t, depth, provider)
	baseClaim := claimBuilder.CreateRootClaim()
	claimLoader.claims = []types.Claim{baseClaim}

	action := types.Action{
		Type:        types.ActionTypeMove,
		ParentClaim: baseClaim,
		IsAttack:    true,
		Value:       common.Hash{0x01},
	}

	// Perform action - should still apply delay despite error
	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {
		agent.performAction(ctx, &wg, createStubGame(claimLoader.claims), action)
		close(done)
	}()

	// Should complete without needing to advance clock (no delay applied for safety when extension detection fails)
	select {
	case <-done:
		// Expected - immediate completion
	case <-time.After(30 * time.Second):
		t.Fatal("Action did not complete immediately when extension detection fails")
	}
	wg.Wait()

	require.Equal(t, 1, responder.PerformedActionCount(), "Should have performed action")
}
