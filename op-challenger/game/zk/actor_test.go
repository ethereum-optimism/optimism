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
	"github.com/stretchr/testify/require"
)

var (
	challengeData = "challenge"
	resolveData   = "resolve"
	l1Time        = time.Unix(9892842, 0)
)

// zkTestL1Head is the game's committed L1 head. currentL1 must be strictly greater for the node to
// count as synced past it (the sync gate skips while currentL1 <= l1Head).
const zkTestL1Head = uint64(785)

type zkTestStubs struct {
	rootProvider *stubSuperRootProvider
	contract     *stubContract
	sender       *stubTxSender
}

func TestActor(t *testing.T) {
	// Super root: matches proposal, mismatches, not yet cross-safe, absent (Data nil), rpc error
	// In challenge period, ChallengePeriodExpired, In proof period, ProvenWithoutChallenge, ProvenAfterChallenge, ProofPeriodExpired, Resolved
	// No parent, parent in progress, parent valid, parent invalid
	tests := []struct {
		name      string
		setup     func(t *testing.T, stubs *zkTestStubs)
		challenge bool
		resolve   bool
		expectErr bool
	}{
		{
			name: "DoNotChallengeValidSuperRootProposal",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.setDeadlineNotReached()
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootTimestamp
			},
		},
		{
			name: "ChallengeMismatchedSuperRoot",
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
			name: "ErrorWhenSuperRootUnavailable",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.rootProvider.outputErr = errors.New("connection refused")
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootTimestamp
			},
			expectErr: true,
		},
		{
			name: "ChallengeProposalWithNoSuperRootAtTimestamp",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.rootProvider.dataNil = true
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootTimestamp
			},
			challenge: true,
		},
		{
			name: "ChallengeStillUnsafeProposal",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = stubs.rootProvider.root
				stubs.contract.l2SequenceNumber = stubs.rootProvider.rootTimestamp
				stubs.rootProvider.safeTimestamp = stubs.rootProvider.rootTimestamp - 1
			},
			challenge: true,
		},
		{
			// Behind the game L1 head and the proposal is invalid: the sync gate must win, so the
			// actor skips rather than challenges on a stale view.
			name: "WaitWhenNotSyncedPastGameL1Head",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				stubs.rootProvider.currentL1 = eth.BlockID{Number: zkTestL1Head}
			},
		},
		{
			// Behind the game L1 head: the challenge is sync-skipped, but ungated resolution still
			// fires off the invalid parent.
			name: "ResolveWhileNotSyncedPastGameL1Head",
			setup: func(t *testing.T, stubs *zkTestStubs) {
				stubs.contract.proposalHash = common.Hash{0xba, 0xd0}
				stubs.rootProvider.currentL1 = eth.BlockID{Number: zkTestL1Head}
				stubs.contract.setParentStatus(types.GameStatusChallengerWon)
			},
			resolve: true,
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
			if tt.expectErr {
				require.Error(t, err)
				require.Empty(t, stubs.sender.sentData)
				return
			}
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

// TestActor_CanonicalUnprovableProposal_WarnsButDoesNotChallenge exercises the Decision-2
// divergence: a canonical, cross-safe proposal that is not provable within the game l1Head is
// accepted with a warning, never challenged. Steps 6 and 7 are otherwise both 0-tx, so the log is
// the only observable difference.
func TestActor_CanonicalUnprovableProposal_WarnsButDoesNotChallenge(t *testing.T) {
	const warnMsg = "not provable within game l1Head"

	t.Run("WarnsWhenUnprovableWithinGameL1Head", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		actor, stubs := newZKActor(t, logger)
		stubs.rootProvider.verifiedRequiredL1 = eth.BlockID{Number: zkTestL1Head + 1}
		require.NoError(t, actor.Act(context.Background()))
		require.Empty(t, stubs.sender.sentData)
		require.NotNil(t, logs.FindLog(testlog.NewMessageContainsFilter(warnMsg)),
			"expected warning for canonical-but-unprovable proposal")
	})

	t.Run("NoWarnWhenProvableWithinGameL1Head", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		actor, stubs := newZKActor(t, logger)
		// Default verifiedRequiredL1 (== l1Head) is provable within the game.
		require.NoError(t, actor.Act(context.Background()))
		require.Empty(t, stubs.sender.sentData)
		require.Nil(t, logs.FindLog(testlog.NewMessageContainsFilter(warnMsg)),
			"step-7 valid proposal must not warn")
	})
}

func setupActorTest(t *testing.T) (*Actor, *zkTestStubs) {
	return newZKActor(t, testlog.Logger(t, log.LvlInfo))
}

func newZKActor(t *testing.T, logger log.Logger) (*Actor, *zkTestStubs) {
	l1Head := eth.BlockID{
		Hash:   common.Hash{0x12},
		Number: zkTestL1Head,
	}
	rootTimestamp := uint64(28492)
	rootProvider := &stubSuperRootProvider{
		root:          common.Hash{0x11},
		rootTimestamp: rootTimestamp,
		safeTimestamp: rootTimestamp + 10,
		// Synced past l1Head and provable within it, so the default proposal is valid.
		currentL1:          eth.BlockID{Number: zkTestL1Head + 1},
		verifiedRequiredL1: eth.BlockID{Number: zkTestL1Head},
	}
	// Default to a valid proposal
	contract := &stubContract{
		proposalHash:     rootProvider.root,
		l2SequenceNumber: rootProvider.rootTimestamp,
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

type stubSuperRootProvider struct {
	outputErr          error
	rootTimestamp      uint64
	root               common.Hash
	safeTimestamp      uint64
	currentL1          eth.BlockID
	verifiedRequiredL1 eth.BlockID
	dataNil            bool
}

func (s *stubSuperRootProvider) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	if s.outputErr != nil {
		return eth.SuperRootAtTimestampResponse{}, s.outputErr
	}
	if timestamp != s.rootTimestamp {
		return eth.SuperRootAtTimestampResponse{}, errors.New("unexpected super root request")
	}
	resp := eth.SuperRootAtTimestampResponse{
		CurrentSafeTimestamp: s.safeTimestamp,
		CurrentL1:            s.currentL1,
	}
	if !s.dataNil {
		resp.Data = &eth.SuperRootResponseData{
			SuperRoot:          eth.Bytes32(s.root),
			VerifiedRequiredL1: s.verifiedRequiredL1,
		}
	}
	return resp, nil
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
