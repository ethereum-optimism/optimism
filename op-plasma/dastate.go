package plasma

import (
	"container/heap"
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

// CommQueue is a priority queue of commitments ordered by block number.
type CommQueue []*Commitment

var _ heap.Interface = (*CommQueue)(nil)

func (c CommQueue) Len() int { return len(c) }

// we want the first item in the queue to have the lowest block number
func (c CommQueue) Less(i, j int) bool {
	return c[i].blockNumber < c[j].blockNumber
}

func (c CommQueue) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *CommQueue) Push(x any) {
	*c = append(*c, x.(*Commitment))
}

func (c *CommQueue) Pop() any {
	old := *c
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*c = old[0 : n-1]
	return item
}

// State tracks the commitment and their challenges in order of l1 inclusion.
type State struct {
	activeComms  CommQueue
	expiredComms CommQueue
	commsByKey   map[string]*Commitment
	log          log.Logger
	metrics      Metricer
	finalized    uint64
}

func NewState(log log.Logger, m Metricer) *State {
	return &State{
		activeComms:  make(CommQueue, 0),
		expiredComms: make(CommQueue, 0),
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
	heap.Push(&s.activeComms, c)
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
	heap.Push(&s.activeComms, c)
	s.commsByKey[string(key)] = c

	return c
}

// GetOrTrackChallenge returns the commitment for the given key if it is already tracked, or
// initializes a new commitment and adds it to the state.
func (s *State) GetOrTrackChallenge(key []byte, bn uint64, challengeWindow uint64) *Commitment {
	if c, ok := s.commsByKey[string(key)]; ok {
		// commitments previously introduced by challenge events are marked as canonical
		c.canonical = true
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
	var err error
	for s.activeComms.Len() > 0 && s.activeComms[0].expiresAt <= bn && s.activeComms[0].blockNumber > s.finalized {
		// move from the active to the expired queue
		c := heap.Pop(&s.activeComms).(*Commitment)
		heap.Push(&s.expiredComms, c)

		if c.canonical {
			// advance finalized head only if the commitment was derived as part of the canonical chain
			s.finalized = c.blockNumber
		}

		// active mark as expired so it is skipped in the derivation pipeline
		if c.challengeStatus == ChallengeActive {
			c.challengeStatus = ChallengeExpired

			// only reorg if canonical. If the pipeline is behind, it will just
			// get skipped once it catches up. If it is spam, it will be pruned
			// with no effect.
			if c.canonical {
				err = ErrReorgRequired
				s.metrics.RecordExpiredChallenge(c.key)
			}
		}
	}

	return s.finalized, err
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
	for s.expiredComms.Len() > 0 && s.expiredComms[0].blockNumber < bn {
		c := heap.Pop(&s.expiredComms).(*Commitment)
		s.log.Debug("prune commitment", "expiresAt", c.expiresAt, "blockNumber", c.blockNumber)
		delete(s.commsByKey, string(c.key))
	}
}

// In case of L1 reorg, state should be cleared so we can sync all the challenge events
// from scratch.
func (s *State) Reset() {
	s.activeComms = s.activeComms[:0]
	s.expiredComms = s.expiredComms[:0]
	s.finalized = 0
	clear(s.commsByKey)
}
