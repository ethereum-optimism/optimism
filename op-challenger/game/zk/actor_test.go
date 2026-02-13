package zk

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	challengeData = "challenge"
	resolveData   = "resolve"
	l1Time        = time.Unix(9892842, 0)
)

type zkTestStubs struct {
	rootProvider *stubRootProvider
	contract     *stubContract
	sender       *stubTxSender
}

func TestActor(t *testing.T) {
	// Output root: Valid, Invalid
	// Safety: Safe, Unsafe, Beyond unsafe
	// In challenge period, ChallengePeriodExpired, In proof period, ProvenWithoutChallenge, ProvenAfterChallenge, ProofPeriodExpired, Resolved
	// No parent, parent in progress, parent valid, parent invalid
	tests := []struct {
		name      string
		setup     func(t *testing.T, stubs *zkTestStubs)
		challenge bool
		resolve   bool
	}{
		{
			name: "DoNotChallengeCorrectProposal",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineNotReached()
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootBlockNum
			},
		},
		{
			name: "ChallengeIncorrectProposal",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
			},
			challenge: true,
		},
		{
			name: "DoNothingIfAlreadyChallenged",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.rootProvider.root = common.Hash{0xba, 0xd0} // Disagree but already challenged
				stubs.contract.challenge(t)
			},
		},
		{
			name: "ChallengeProposalBeyondCurrentUnsafeHead",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.rootProvider.root = common.Hash{0xba, 0xd0}
				stubs.rootProvider.outputErr = mockNotFoundRPCError()
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootBlockNum
			},
			challenge: true,
		},
		{
			name: "ChallengeCurrentlyUnsafeProposal",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootBlockNum
				stubs.rootProvider.safeBlockNum = stubs.rootProvider.rootBlockNum - 1
			},
			challenge: true,
		},
		{
			name: "ChallengeUnresolvableGameWithNoParent",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				stubs.contract.parentIndex = math.MaxUint32
			},
			challenge: true,
		},
		{
			name: "ResolveGameWithNoParent",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				stubs.contract.parentIndex = math.MaxUint32
			},
			resolve: true,
		},
		{
			name: "DoNothingWhenDeadlineExpiredButParentNotResolved",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
				// Proposal is invalid but can't challenge because the deadline is expired
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				// And can't resolve because the parent is still unresolved
				stubs.contract.setParentStatus(types.GameStatusInProgress)
			},
		},
		{
			name: "InChallengePeriodWithInvalidParent",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				// Game should be challenged
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				stubs.contract.setDeadlineNotReached()
				// And is immediately resolvable because the parent is invalid
				stubs.contract.setParentStatus(types.GameStatusChallengerWon)
			},
			challenge: true,
			resolve:   true,
		},
		{
			name: "UnchallengedWithDeadlineExpired",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
			},
			resolve: true,
		},
		{
			name: "ChallengedWithDeadlineExpired",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
				stubs.contract.challenge(t)
			},
			resolve: true,
		},
		{
			name: "ChallengedAndProvenWithDeadlineExpired",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
				stubs.contract.challenge(t)
				stubs.contract.prove(t)
			},
			resolve: true,
		},
		{
			name: "ChallengedAndProvenWithDeadlineNotReached",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineNotReached()
				stubs.contract.challenge(t)
				stubs.contract.prove(t)
			},
			resolve: true,
		},
		{
			name: "UnchallengedAndProvenWithDeadlineExpired",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineExpired()
				stubs.contract.prove(t)
			},
			resolve: true,
		},
		{
			name: "UnchallengedAndProvenWithDeadlineNotReached",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineNotReached()
				stubs.contract.prove(t)
			},
			resolve: true,
		},
		{
			name: "AlreadyResolved",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineNotReached()
				stubs.contract.markResolved()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor, stubs := setupActorTest(t)
			if tt.setup != nil {
				tt.setup(t, stubs)
			}
			err := actor.Act(context.Background())
			require.NoError(t, err)
			expectedTxCount := 0
			if tt.challenge {
				require.Contains(t, stubs.sender.sentData, challengeData)
				expectedTxCount++
			}
			if tt.resolve {
				require.Contains(t, stubs.sender.sentData, resolveData)
				expectedTxCount++
			}
			require.Len(t, stubs.sender.sentData, expectedTxCount)
		})
	}
}

func setupActorTest(t *testing.T) (*Actor, *zkTestStubs) {
	logger := testlog.Logger(t, log.LvlInfo)
	l1Head := eth.BlockID{
		Hash:   common.Hash{0x12},
		Number: 785,
	}
	rootBlockNum := uint64(28492)
	rootProvider := &stubRootProvider{
		root:         common.Hash{0x11},
		rootBlockNum: rootBlockNum,
		safeBlockNum: rootBlockNum + 10,
	}
	// Default to a valid proposal
	contract := &stubContract{
		proposalHash:     rootProvider.root,
		l2SequenceNumber: rootProvider.rootBlockNum,
		parentStatus:     types.GameStatusDefenderWon,
		parentIndex:      482,
	}
	contract.setDeadlineNotReached()
	txSender := &stubTxSender{}
	l1Clock := clock.NewDeterministicClock(l1Time)
	// Simplify the tests by using the same stub for the game and the dispute game factory
	creator := ActorCreator(l1Clock, rootProvider, contract, contract, txSender)
	genericActor, err := creator(context.Background(), logger, l1Head)
	require.NoError(t, err, "failed to create actor")
	actor, ok := genericActor.(*Actor)
	require.True(t, ok, "actor is not of expected type")
	return actor, &zkTestStubs{
		rootProvider: rootProvider,
		contract:     contract,
		sender:       txSender,
	}
}

type stubRootProvider struct {
	outputErr    error
	rootBlockNum uint64
	root         common.Hash
	safeBlockNum uint64
}

func (s *stubRootProvider) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	if s.outputErr != nil {
		return nil, s.outputErr
	}
	if blockNum != s.rootBlockNum {
		return nil, errors.New("unexpected output request")
	}
	return &eth.OutputResponse{
		OutputRoot: eth.Bytes32(s.root),
		Status: &eth.SyncStatus{
			SafeL2: eth.L2BlockRef{
				Number: s.safeBlockNum,
			},
		},
	}, nil
}

type stubContract struct {
	parentIndex      uint32
	parentStatus     types.GameStatus
	proposalStatus   contracts.ProposalStatus
	deadline         time.Time
	txCreated        bool
	proposalHash     common.Hash
	l2SequenceNumber uint64
}

func (s *stubContract) Addr() common.Address {
	return common.Address{0x67, 0x67, 0x67}
}

func (s *stubContract) challenge(t *testing.T) {
	require.Equal(t, contracts.ProposalStatusUnchallenged, s.proposalStatus, "game not in challengable state")
	s.proposalStatus = contracts.ProposalStatusChallenged
}

func (s *stubContract) prove(t *testing.T) {
	if s.proposalStatus == contracts.ProposalStatusUnchallenged {
		s.proposalStatus = contracts.ProposalStatusUnchallengedAndValidProofProvided
		return
	}
	require.Equal(t, contracts.ProposalStatusChallenged, s.proposalStatus, "game not in provable state")
	s.proposalStatus = contracts.ProposalStatusChallengedAndValidProofProvided
}

func (s *stubContract) setDeadlineExpired() {
	s.deadline = l1Time.Add(-1 * time.Second)
}

func (s *stubContract) setDeadlineNotReached() {
	s.deadline = l1Time.Add(1 * time.Second)
}

func (s *stubContract) markResolved() {
	s.proposalStatus = contracts.ProposalStatusResolved
}

func (s *stubContract) setParentStatus(status types.GameStatus) {
	s.parentStatus = status
}

func (s *stubContract) GetGameStatus(_ context.Context, idx uint64) (types.GameStatus, error) {
	if idx != uint64(s.parentIndex) {
		return 0, errors.New("unexpected parent index")
	}
	if idx == math.MaxUint32 {
		return 0, errors.New("execution reverted") // no such game
	}
	return s.parentStatus, nil
}

func (s *stubContract) GetChallengerMetadata(_ context.Context, _ rpcblock.Block) (contracts.ChallengerMetadata, error) {
	return contracts.ChallengerMetadata{
		ParentIndex:      s.parentIndex,
		ProposalStatus:   s.proposalStatus,
		ProposedRoot:     s.proposalHash,
		L2SequenceNumber: s.l2SequenceNumber,
		Deadline:         s.deadline,
	}, nil
}

func (s *stubContract) ChallengeTx(_ context.Context) (txmgr.TxCandidate, error) {
	s.txCreated = true
	return txmgr.TxCandidate{
		TxData: []byte(challengeData),
	}, nil
}

func (s *stubContract) ResolveTx() (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{
		TxData: []byte(resolveData),
	}, nil
}

func (s *stubContract) GetProposal(_ context.Context) (common.Hash, uint64, error) {
	return s.proposalHash, s.l2SequenceNumber, nil
}

type stubTxSender struct {
	sentData []string
	sendErr  error
}

func (s *stubTxSender) SendAndWaitSimple(_ string, candidates ...txmgr.TxCandidate) error {
	for _, candidate := range candidates {
		s.sentData = append(s.sentData, string(candidate.TxData))
	}
	if s.sendErr != nil {
		return s.sendErr
	}
	return nil
}

// mockNotFoundRPCError creates a minimal rpc.Error that reports a "not found" message
// to exercise the JSON-RPC application error path in the enricher.
func mockNotFoundRPCError() rpc.Error { return testRPCError{msg: "not found", code: -32000} }

type testRPCError struct {
	msg  string
	code int
}

func (e testRPCError) Error() string  { return e.msg }
func (e testRPCError) ErrorCode() int { return e.code }
