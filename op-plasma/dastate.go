package plasma

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// ErrReorgRequired is returned when a commitment was derived but for which the challenge expired.
// This requires a reorg to rederive without the input even if the input was previously available.
var ErrReorgRequired = errors.New("reorg required")

type ChallengeStatus uint8

const (
	ChallengeUninitialized ChallengeStatus = iota
	ChallengeActive
	ChallengeResolved
	ChallengeExpired
)

// Commitment keeps track of the onchain state of an input commitment.
type Commitment struct {
	comm               CommitmentData
	inclusionBlock     eth.L1BlockRef // block where the commitment is included as calldata to the batcher inbox.
	challengeWindowEnd uint64         // represents the block number after which the commitment can no longer be challenged.
}

type Challenge struct {
	comm                     CommitmentData  // the specific commitment which was challenged
	commInclusionBlockNumber uint64          // block where the commitment is included as calldata to the batcher inbox
	resolveWindowEnd         uint64          // block number at which the challenge must be resolved by
	input                    []byte          // the input itself if it was resolved onchain
	challengeStatus          ChallengeStatus // status of the challenge based on the highest processed action
}

func (c *Challenge) key() string {
	return challengeKey(c.comm, c.commInclusionBlockNumber)
}

func challengeKey(comm CommitmentData, inclusionBlockNumber uint64) string {
	return fmt.Sprintf("%d%x", inclusionBlockNumber, comm.Encode())
}

// State tracks the commitment and their challenges in order of l1 inclusion.
// Commitments & challenges are tracked in their L1 inclusion order. Commitments are split into un-expired & expired queues.
// When commitments are moved from the un-expired to expired queues, if there is an active challenge the DA Manager is informed
// that a commitment became invalid. When commitments + challenges expire in a block which is finalized, they are removed from
// the state.
// In the special case of a L2 reorg, challenges are still tracked but commitments are removed. This will allow the plasma fetcher
// to find the expired challenge.
type State struct {
	commitments        []Commitment          // commitments where the challenge/resolve period has not expired yet
	expiredCommitments []Commitment          // commitments where the challenge/resolve period has expired but not finalized
	challenges         []*Challenge          // challenges ordered by L1 inclusion
	expiredChallenges  []*Challenge          // challenges ordered by L1 inclusion
	challengesMap      map[string]*Challenge // challenges by seralized comm + block number for easy lookup
	cfg                Config
	log                log.Logger
	metrics            Metricer
}

func NewState(log log.Logger, m Metricer, cfg Config) *State {
	return &State{
		commitments:        make([]Commitment, 0),
		expiredCommitments: make([]Commitment, 0),
		challenges:         make([]*Challenge, 0),
		expiredChallenges:  make([]*Challenge, 0),
		challengesMap:      make(map[string]*Challenge),
		cfg:                cfg,
		log:                log,
		metrics:            m,
	}
}

// ClearCommitments removes all tracked commitments but not challenges.
// This should be used to retain the challenge state when performing a L2 reorg
func (s *State) ClearCommitments() {
	s.commitments = s.commitments[:0]
	s.expiredCommitments = s.expiredCommitments[:0]
}

// Reset clears the state. It should be used when a L1 reorg occurs.
func (s *State) Reset() {
	s.commitments = s.commitments[:0]
	s.expiredCommitments = s.expiredCommitments[:0]
	s.challenges = s.challenges[:0]
	s.expiredChallenges = s.expiredChallenges[:0]
	clear(s.challengesMap)
}

// CreateChallenge creates & tracks a challenge. It will overwrite earlier challenges if the
// same commitment is challenged again.
func (s *State) CreateChallenge(comm CommitmentData, inclusionBlock eth.BlockID, commBlockNumber uint64) {
	c := &Challenge{
		comm:                     comm,
		commInclusionBlockNumber: commBlockNumber,
		resolveWindowEnd:         inclusionBlock.Number + s.cfg.ResolveWindow,
		challengeStatus:          ChallengeActive,
	}
	s.challenges = append(s.challenges, c)
	s.challengesMap[c.key()] = c
}

// ResolveChallenge marks a challenge as resolved. It will return an error if there was not a corresponding challenge.
func (s *State) ResolveChallenge(comm CommitmentData, inclusionBlock eth.BlockID, commBlockNumber uint64, input []byte) error {
	c, ok := s.challengesMap[challengeKey(comm, commBlockNumber)]
	if !ok {
		return errors.New("challenge was not tracked")
	}
	c.input = input
	c.challengeStatus = ChallengeResolved
	return nil
}

// TrackCommitment stores a commitment in the State
func (s *State) TrackCommitment(comm CommitmentData, inclusionBlock eth.L1BlockRef) {
	c := Commitment{
		comm:               comm,
		inclusionBlock:     inclusionBlock,
		challengeWindowEnd: inclusionBlock.Number + s.cfg.ChallengeWindow,
	}
	s.commitments = append(s.commitments, c)
}

// GetChallenge looks up a challenge against commitment + inclusion block.
func (s *State) GetChallenge(comm CommitmentData, commBlockNumber uint64) (*Challenge, bool) {
	challenge, ok := s.challengesMap[challengeKey(comm, commBlockNumber)]
	return challenge, ok
}

// GetChallenge looks up a challenge against commitment + inclusion block.
func (s *State) GetChallengeStatus(comm CommitmentData, commBlockNumber uint64) ChallengeStatus {
	challenge, ok := s.GetChallenge(comm, commBlockNumber)
	if ok {
		return challenge.challengeStatus
	}
	return ChallengeUninitialized
}

// NoCommitments returns true iff it is not tracking any commitments or challenges.
func (s *State) NoCommitments() bool {
	return len(s.challenges) == 0 && len(s.expiredChallenges) == 0 && len(s.commitments) == 0 && len(s.expiredCommitments) == 0
}

// ExpireCommitments moves commitments from the could be challenged/resolved state to the have been resolved as valid or not but not yet finalized state.
// If a commitment was found to be invalid (via a challenge marking it as Expired), we return ErrExpiredChallenge to indicate that a L2 reorg
// should be performed.
func (s *State) ExpireCommitments(origin eth.BlockID) error {
	var err error
	for len(s.commitments) > 0 {
		c := s.commitments[0]
		challenge, ok := s.GetChallenge(c.comm, c.inclusionBlock.Number)

		// Determine when the commitment's challenge window expires
		expiresAt := c.challengeWindowEnd
		if ok {
			expiresAt = challenge.resolveWindowEnd
		}

		// If the commitment expires the in future, return early
		if expiresAt > origin.Number {
			return err
		}
		s.log.Info("Expiring commitment", "comm", c.comm, "commInclusionBlockNumber", c.inclusionBlock.Number, "origin", origin, "challenged", ok)

		// If it has expired, move the commitment to the expired queue
		s.expiredCommitments = append(s.expiredCommitments, c)
		s.commitments = s.commitments[1:]
		// If there was a challenge which was not resolved, expire the commitment.
		if ok && challenge.challengeStatus != ChallengeResolved {
			err = ErrReorgRequired
		}
	}
	return err
}

// ExpireChallenges marks challenges as expired. It must be called for every block because there is no contract event to expire challenges.
func (s *State) ExpireChallenges(origin eth.BlockID) {
	for len(s.challenges) > 0 {
		c := s.challenges[0]
		// If the challenge can still be resolved, return early
		if c.resolveWindowEnd > origin.Number {
			return
		}
		s.log.Info("Expiring challenge", "comm", c.comm, "commInclusionBlockNumber", c.commInclusionBlockNumber, "origin", origin)

		// Move the challenge to the expired queue
		s.expiredChallenges = append(s.expiredChallenges, c)
		s.challenges = s.challenges[1:]
		// Mark the challenge as expired if it was not resolved
		if c.challengeStatus == ChallengeActive {
			c.challengeStatus = ChallengeExpired
		}
	}
}

// Prune removes challenges & commitments which have been expired in a block which is finalized.
// It returns the inclusion block number of a commitment which has been finalized.
func (s *State) Prune(finalizedBlock eth.BlockID) eth.L1BlockRef {
	var ret eth.L1BlockRef
	for len(s.expiredCommitments) > 0 {
		c := s.expiredCommitments[0]
		challenge, ok := s.GetChallenge(c.comm, c.inclusionBlock.Number)

		// Determine when the commitment expires
		expiresAt := c.challengeWindowEnd
		if ok {
			expiresAt = challenge.resolveWindowEnd
		}

		// If the commitment expires the in future, stop processing commitments
		if expiresAt > finalizedBlock.Number {
			break
		}
		// Remove the finalized commitment
		s.expiredCommitments = s.expiredCommitments[1:]
		// Return the inclusion block which is passed as the finalized block to the finality controller
		ret = c.inclusionBlock
	}

	for len(s.expiredChallenges) > 0 {
		c := s.expiredChallenges[0]
		// If the challenge expires the in future, stop processing challenges
		if c.resolveWindowEnd > finalizedBlock.Number {
			break
		}
		// Remove the finalized challenge
		s.expiredChallenges = s.expiredChallenges[1:]
		delete(s.challengesMap, c.key())
	}
	return ret
}
