package plasma

import (
	"errors"
	"fmt"

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
	key             []byte          // the encoded commitment
	input           []byte          // the input itself if it was resolved onchain
	expiresAt       uint64          // represents the block number after which the commitment can no longer be challenged or if challenged no longer be resolved.
	blockNumber     uint64          // block where the commitment is included as calldata to the batcher inbox
	challengeStatus ChallengeStatus // latest known challenge status
	canonical       bool            // whether the commitment was derived as part of the canonical chain if canonical it will be in comms queue if not in the pendingComms queue.
}

// CommQueue is a FIFO queue of commitments ordered by block number.
// They are naturally ordered as commitments are inserted in order of traversal.
// State impl makes sure there are no duplicates and in case of retraversal we also reset the queue.
type CommQueue []*Commitment

// PendingCommQueue is a FIFO queue of commitments ordered by L1 challenge creation block.
// They are naturally ordered as commitments are inserted in order of traversal.
// When a challenge is indexed for a commitment that has not yet been derived, it is added to this queue.
type PendingCommQueue []*Commitment

// State tracks the commitment and their challenges in order of l1 inclusion.
type State struct {
	comms        CommQueue
	pendingComms PendingCommQueue
	commsByKey   map[string]*Commitment
	log          log.Logger
	metrics      Metricer
}

func NewState(log log.Logger, m Metricer) *State {
	return &State{
		comms:        make(CommQueue, 0),
		pendingComms: make(PendingCommQueue, 0),
		commsByKey:   make(map[string]*Commitment),
		log:          log,
		metrics:      m,
	}
}

// IsTracking returns whether we currently have a commitment for the given key.
// if the block number is mismatched we return false to ignore the challenge.
func (s *State) IsTracking(key []byte, bn uint64) bool {
	if c, ok := s.commsByKey[string(key)]; ok {
		return c.blockNumber == bn
	}
	// track the commitment knowing we may be in detached head and not have seen
	// the commitment in the inbox yet.
	s.TrackDetachedCommitment(key, bn)
	return true
}

// TrackDetachedCommitment is used for indexing challenges for commitments that have not yet
// been derived due to the derivation pipeline being stalled pending a commitment to be challenged.
// Memory usage is bound to L1 block space during the DA windows, so it is hard and expensive to spam.
// Note that the challenge status and expiration is updated separately after it is tracked.
func (s *State) TrackDetachedCommitment(key []byte, bn uint64) {
	c := &Commitment{
		key:         key,
		expiresAt:   bn,
		blockNumber: bn,
		canonical:   false,
	}
	s.log.Debug("tracking detached commitment", "blockNumber", c.blockNumber, "commitment", fmt.Sprintf("%x", key))
	s.pendingComms = append(s.pendingComms, c)
	s.commsByKey[string(key)] = c
}

// SetActiveChallenge switches the state of a given commitment to active challenge. Noop if
// the commitment is not tracked as we don't want to track challenges for invalid commitments.
func (s *State) SetActiveChallenge(key []byte, challengedAt uint64, resolveWindow uint64) {
	if c, ok := s.commsByKey[string(key)]; ok {
		c.expiresAt = challengedAt + resolveWindow
		c.challengeStatus = ChallengeActive
		s.metrics.RecordActiveChallenge(c.blockNumber, challengedAt, key)
	}
}

// SetResolvedChallenge switches the state of a given commitment to resolved. Noop if
// the commitment is not tracked as we don't want to track challenges for invalid commitments.
// The input posted onchain is stored in the state for later retrieval.
func (s *State) SetResolvedChallenge(key []byte, input []byte, resolvedAt uint64) {
	if c, ok := s.commsByKey[string(key)]; ok {
		c.challengeStatus = ChallengeResolved
		c.expiresAt = resolvedAt
		c.input = input
		s.metrics.RecordResolvedChallenge(key)
	}
}

// SetInputCommitment initializes a new commitment and adds it to the state.
// This is called when we see a commitment during derivation so we can refer to it later in
// challenges.
func (s *State) SetInputCommitment(key []byte, committedAt uint64, challengeWindow uint64) *Commitment {
	c := &Commitment{
		key:         key,
		expiresAt:   committedAt + challengeWindow,
		blockNumber: committedAt,
		canonical:   true,
	}
	s.log.Debug("append commitment", "expiresAt", c.expiresAt, "blockNumber", c.blockNumber)
	s.comms = append(s.comms, c)
	s.commsByKey[string(key)] = c

	return c
}

// GetOrTrackChallenge returns the commitment for the given key if it is already tracked, or
// initializes a new commitment and adds it to the state.
func (s *State) GetOrTrackChallenge(key []byte, bn uint64, challengeWindow uint64) *Commitment {
	if c, ok := s.commsByKey[string(key)]; ok {
		// if the commitment was previously tracked from a challenge event,
		// promote it to the comms queue. It will be removed from pending during pruning step.
		if !c.canonical {
			s.comms = append(s.comms, c)
			c.canonical = true
		}
		return c
	}
	return s.SetInputCommitment(key, bn, challengeWindow)
}

// GetResolvedInput returns the input bytes if the commitment was resolved onchain.
func (s *State) GetResolvedInput(key []byte) ([]byte, error) {
	if c, ok := s.commsByKey[string(key)]; ok {
		return c.input, nil
	}
	return nil, errors.New("commitment not found")
}

// ExpireChallenges walks back from the oldest commitment to find the latest l1 origin
// for which input data can no longer be challenged. It also marks any active challenges
// as expired based on the new latest l1 origin. If any active challenges are expired
// it returns an error to signal that a derivation pipeline reset is required.
func (s *State) ExpireChallenges(bn uint64) (uint64, error) {
	latest := uint64(0)
	var err error
	for i := 0; i < len(s.comms); i++ {
		c := s.comms[i]
		if c.expiresAt <= bn && c.blockNumber > latest {
			latest = c.blockNumber

			if c.challengeStatus == ChallengeActive {
				c.challengeStatus = ChallengeExpired
				s.metrics.RecordExpiredChallenge(c.key)
				err = ErrReorgRequired
			}
		} else {
			break
		}
	}
	return latest, err
}

// safely prune in case reset is deeper than the finalized l1
const commPruneMargin = 200

// Prune removes commitments once they can no longer be challenged or resolved.
// the finalized head block number is passed so we can safely remove any commitments
// with finalized block numbers.
func (s *State) Prune(bn uint64) {
	if bn > commPruneMargin {
		bn -= commPruneMargin
	} else {
		bn = 0
	}
	i := 0
	for i < len(s.comms) {
		c := s.comms[i]
		// s.comms is ordered by block number.
		if c.blockNumber < bn {
			delete(s.commsByKey, string(c.key))
			i++
		} else {
			break
		}
	}
	if i > 0 {
		s.comms = append(s.comms[:0], s.comms[i:]...)
	}
	// pending commitments are also cleared once block is finalized
	j := 0
	for j < len(s.pendingComms) {
		c := s.pendingComms[j]
		// s.pendingComms is ordered by expiration block. As a result pruning of pending commitments
		// will lag behind the pruning of the canonical commitments.
		if c.expiresAt < bn {
			// canonical commitments are evicted during the previous step
			if !c.canonical {
				delete(s.commsByKey, string(c.key))
			}
			j++
		} else {
			break
		}
	}
	if j > 0 {
		s.pendingComms = append(s.pendingComms[:0], s.pendingComms[j:]...)
	}
	s.log.Info("pruned commitments", "canonical", i, "pending", j)
}

// In case of L1 reorg, state should be cleared so we can sync all the challenge events
// from scratch.
func (s *State) Reset() {
	s.comms = s.comms[:0]
	clear(s.commsByKey)
}
