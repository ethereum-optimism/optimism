package p2p

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ApplicationScoreParams struct {
	ValidResponseCap    float64
	ValidResponseWeight float64
	ValidResponseDecay  float64

	ErrorResponseCap    float64
	ErrorResponseWeight float64
	ErrorResponseDecay  float64

	RejectedPayloadCap    float64
	RejectedPayloadWeight float64
	RejectedPayloadDecay  float64

	DecayToZero float64
	DecayPeriod time.Duration
}

// TODO: Validation of params
// decay must be between 0 and 1
// rewards must be weighted positive, penalties weighted negative

type scoreBook interface {
	GetPeerScores(id peer.ID) (store.PeerScores, error)
	SetScore(id peer.ID, diff store.ScoreDiff) (store.PeerScores, error)
}

type peerApplicationScorer struct {
	log            log.Logger
	clock          clock.Clock
	params         *ApplicationScoreParams
	scorebook      scoreBook
	connectedPeers func() []peer.ID

	done sync.WaitGroup
}

func newPeerApplicationScorer(logger log.Logger, clock clock.Clock, params *ApplicationScoreParams, scorebook scoreBook, connectedPeers func() []peer.ID) *peerApplicationScorer {
	return &peerApplicationScorer{
		log:            logger,
		clock:          clock,
		params:         params,
		scorebook:      scorebook,
		connectedPeers: connectedPeers,
	}
}

func (s *peerApplicationScorer) ApplicationScore(id peer.ID) float64 {
	scores, err := s.scorebook.GetPeerScores(id)
	if err != nil {
		s.log.Error("Failed to load peer scores", "peer", id, "err", err)
		return 0
	}
	score := scores.ReqResp.ValidResponses * s.params.ValidResponseWeight
	score += scores.ReqResp.ErrorResponses * s.params.ErrorResponseWeight
	score += scores.ReqResp.RejectedPayloads * s.params.RejectedPayloadWeight
	return score
}

func (s *peerApplicationScorer) onValidResponse(id peer.ID) {
	_, err := s.scorebook.SetScore(id, store.NewReqRespScoresDiff(func(target *store.ReqRespScores) {
		target.ValidResponses = math.Min(target.ValidResponses+1, s.params.ValidResponseCap)
	}))
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) onResponseError(id peer.ID) {
	_, err := s.scorebook.SetScore(id, store.NewReqRespScoresDiff(func(target *store.ReqRespScores) {
		target.ErrorResponses = math.Min(target.ErrorResponses+1, s.params.ErrorResponseCap)
	}))
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) onRejectedPayload(id peer.ID) {
	_, err := s.scorebook.SetScore(id, store.NewReqRespScoresDiff(func(target *store.ReqRespScores) {
		target.RejectedPayloads = math.Min(target.RejectedPayloads+1, s.params.RejectedPayloadCap)
	}))
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) decayScores(id peer.ID) {
	decay := func(value float64, decay float64) float64 {
		value *= decay
		if value < s.params.DecayToZero {
			return 0
		}
		return value
	}
	_, err := s.scorebook.SetScore(id, store.NewReqRespScoresDiff(func(target *store.ReqRespScores) {
		target.ValidResponses = decay(target.ValidResponses, s.params.ValidResponseDecay)
		target.ErrorResponses = decay(target.ErrorResponses, s.params.ErrorResponseDecay)
		target.RejectedPayloads = decay(target.RejectedPayloads, s.params.RejectedPayloadDecay)
	}))
	if err != nil {
		s.log.Error("Unable to decay peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) decayConnectedPeerScores() {
	for _, id := range s.connectedPeers() {
		s.decayScores(id)
	}
}

func (s *peerApplicationScorer) start(ctx context.Context) {
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		ticker := s.clock.NewTicker(s.params.DecayPeriod)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.Ch():
				s.decayConnectedPeerScores()
			}
		}
	}()
}
