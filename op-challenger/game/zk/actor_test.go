package zk

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var challengeData = []byte{0x99, 0x98}

func TestActor_DoNothingIfAlreadyChallenged(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	rootProvider.root = common.Hash{0xba, 0xd0} // Disagree but already challenged
	contract.challenged = true
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

func TestActor_DoNotChallengeCorrectProposal(t *testing.T) {
	actor, rootProvider, contract, sender := setupActorTest(t)
	contract.challenged = false
	contract.proposalHash = rootProvider.root
	contract.l2SequenceNumber = rootProvider.rootBlockNum
	verifyNoChallenge(t, actor, contract, sender)
}

func verifyNoChallenge(t *testing.T, actor *Actor, contract *stubContract, sender *stubTxSender) {
	err := actor.Act(context.Background())
	require.NoError(t, err)
	require.False(t, contract.txCreated, "should not challenge already challenged game")
	require.Empty(t, sender.sent, "should not send challenge tx")
}

func verifyChallenge(t *testing.T, actor *Actor, contract *stubContract, sender *stubTxSender) {
	err := actor.Act(context.Background())
	require.NoError(t, err)
	require.True(t, contract.txCreated, "should not challenge already challenged game")
	require.Len(t, sender.sent, 1, "should not send challenge tx")
	require.Equal(t, challengeData, sender.sent[0].TxData, "should have sent expected challenge transaction")
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
	}
	txSender := &stubTxSender{}
	creator := ActorCreator(rootProvider, contract, txSender)
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
	challenged       bool
	txCreated        bool
	proposalHash     common.Hash
	l2SequenceNumber uint64
}

func (s *stubContract) CanChallenge(_ context.Context) (bool, error) {
	return !s.challenged, nil
}

func (s *stubContract) ChallengeTx(_ context.Context) (txmgr.TxCandidate, error) {
	s.txCreated = true
	return txmgr.TxCandidate{
		TxData: challengeData,
	}, nil
}

func (s *stubContract) GetProposal(_ context.Context) (common.Hash, uint64, error) {
	return s.proposalHash, s.l2SequenceNumber, nil
}

type stubTxSender struct {
	sent    []txmgr.TxCandidate
	sendErr error
}

func (s *stubTxSender) SendAndWaitSimple(_ string, candidates ...txmgr.TxCandidate) error {
	s.sent = append(s.sent, candidates...)
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
