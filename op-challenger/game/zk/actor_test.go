package zk

import (
	"context"
	"errors"
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

func TestActor_DoNothingIfAlreadyChallenged(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	rootProvider.root = common.Hash{0xba, 0xd0} // Disagree but already challenged
	contract.challenge(t)
	verifyNoChallenge(t, actor, contract, sender)
}

func TestActor_ChallengeIncorrectProposal(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	rootProvider.root = common.Hash{0xba, 0xd0}
	contract.proposalHash = common.Hash{0x11}
	contract.l2SequenceNumber = uint64(28492)
	verifyChallenge(t, actor, contract, sender)
}

func TestActor_ChallengeProposalBeyondCurrentUnsafeHead(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	rootProvider.root = common.Hash{0xba, 0xd0}
	rootProvider.outputErr = mockNotFoundRPCError()
	contract.proposalHash = rootProvider.root
	contract.l2SequenceNumber = rootProvider.rootBlockNum
	verifyChallenge(t, actor, contract, sender)
}

func TestActor_Resolve(t *testing.T) {
	// Could just call `gameOver()` and check parent.
	t.Run("ParentNotResolved", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		// Child is resolvable but still has to wait until the parent is resolved.
		contract.setDeadlineExpired()
		contract.setParentStatus(types.GameStatusInProgress)
		// Invalid proposal hash but shouldn't challenge as deadline has expired
		contract.proposalHash = common.Hash{0xba, 0xd0}
		err := actor.Act(context.Background())
		require.NoError(t, err)
		require.Empty(t, sender.sentData, "should not resolve game before parent")
	})

	t.Run("InChallengePeriodButParentInvalid", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		// Child is resolvable but still has to wait until the parent is resolved.
		contract.setDeadlineNotReached()
		contract.setParentStatus(types.GameStatusChallengerWon)
		// Invalid proposal hash but shouldn't challenge as game can be resolved
		contract.proposalHash = common.Hash{0xba, 0xd0}
		verifyResolved(t, actor, sender)
	})

	t.Run("Unchallenged-DeadlinePassed", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		contract.setDeadlineExpired()
		verifyResolved(t, actor, sender)
	})

	t.Run("Challenged-DeadlinePassed", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		contract.challenge(t)
		// When challenged, the deadline is set to the deadline for proving which has expired
		contract.setDeadlineExpired()
		verifyResolved(t, actor, sender)
	})

	t.Run("Proven-Challenged", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		contract.challenge(t)
		contract.prove(t, common.Address{0xaa})
		contract.setDeadlineNotReached()
		verifyResolved(t, actor, sender)
	})

	t.Run("Proven-Unchallenged", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		contract.prove(t, common.Address{0xaa})
		contract.setDeadlineNotReached()
		verifyResolved(t, actor, sender)
	})

	t.Run("Resolved", func(t *testing.T) {
		actor, _, contract, sender := setupActorTest(t)
		contract.markResolved()
		err := actor.Act(context.Background())
		require.NoError(t, err)
		require.Empty(t, sender.sentData, "should not act on resolved game")
	})
}

func verifyResolved(t *testing.T, actor *Actor, sender *stubTxSender) {
	err := actor.Act(context.Background())
	require.NoError(t, err)
	require.Len(t, sender.sentData, 1, "should have sent resolve tx")
	require.Equal(t, resolveData, sender.sentData[0], "tx should have used resolve data")
}

func TestActor_DoNotChallengeCorrectProposal(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	contract.proposalHash = rootProvider.root
	contract.l2SequenceNumber = rootProvider.rootBlockNum
	verifyNoChallenge(t, actor, contract, sender)
}

func verifyNoChallenge(t *testing.T, actor *Actor, contract *stubContract, sender *stubTxSender) {
	err := actor.Act(context.Background())
	require.NoError(t, err)
	require.False(t, contract.txCreated, "should not challenge already challenged game")
	require.Empty(t, sender.sentData, "should not send challenge tx")
}

func verifyChallenge(t *testing.T, actor *Actor, contract *stubContract, sender *stubTxSender) {
	err := actor.Act(context.Background())
	require.NoError(t, err)
	require.True(t, contract.txCreated, "should not challenge already challenged game")
	require.Len(t, sender.sentData, 1, "should not send challenge tx")
	require.Equal(t, challengeData, sender.sentData[0], "should have sent expected challenge transaction")
}

func setupActorTest(t *testing.T) (*Actor, *stubRootProvider, *stubContract, *stubTxSender) {
	logger := testlog.Logger(t, log.LvlInfo)
	l1Head := eth.BlockID{
		Hash:   common.Hash{0x12},
		Number: 785,
	}
	rootBlockNum := uint64(28492)
	rootProvider := &stubRootProvider{
		root:         common.Hash{0x11},
		rootBlockNum: rootBlockNum,
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
	return actor, rootProvider, contract, txSender
}

type stubRootProvider struct {
	outputErr    error
	rootBlockNum uint64
	root         common.Hash
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

func (s *stubContract) challenge(t *testing.T) {
	require.Equal(t, contracts.ProposalStatusUnchallenged, s.proposalStatus, "game not in challengable state")
	s.proposalStatus = contracts.ProposalStatusChallenged
}

func (s *stubContract) prove(t *testing.T, prover common.Address) {
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
