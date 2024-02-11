package p2p

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ScoreBook interface {
	GetPeerScores(id peer.ID) (store.PeerScores, error)
	SetScore(id peer.ID, diff store.ScoreDiff) (store.PeerScores, error)
}

type ApplicationScorer interface {
	ApplicationScore(id peer.ID) float64
	onValidResponse(id peer.ID)
	onResponseError(id peer.ID)
	onRejectedPayload(id peer.ID)
	start()
	stop()
}

type peerApplicationScorer struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	log            log.Logger
	clock          clock.Clock
	params         *ApplicationScoreParams
	scorebook      ScoreBook
	connectedPeers func() []peer.ID

	done sync.WaitGroup
}

var _ ApplicationScorer = (*peerApplicationScorer)(nil)

func newPeerApplicationScorer(ctx context.Context, logger log.Logger, clock clock.Clock, params *ApplicationScoreParams, scorebook ScoreBook, connectedPeers func() []peer.ID) *peerApplicationScorer {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &peerApplicationScorer{
		ctx:            ctx,
		cancelFunc:     cancelFunc,
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
	_, err := s.scorebook.SetScore(id, store.IncrementValidResponses{Cap: s.params.ValidResponseCap})
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) onResponseError(id peer.ID) {
	_, err := s.scorebook.SetScore(id, store.IncrementErrorResponses{Cap: s.params.ErrorResponseCap})
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) onRejectedPayload(id peer.ID) {
	_, err := s.scorebook.SetScore(id, store.IncrementRejectedPayloads{Cap: s.params.RejectedPayloadCap})
	if err != nil {
		s.log.Error("Unable to update peer score", "peer", id, "err", err)
		return
	}
}

func (s *peerApplicationScorer) decayScores(id peer.ID) {
	_, err := s.scorebook.SetScore(id, &store.DecayApplicationScores{
		ValidResponseDecay:   s.params.ValidResponseDecay,
		ErrorResponseDecay:   s.params.ErrorResponseDecay,
		RejectedPayloadDecay: s.params.RejectedPayloadDecay,
		DecayToZero:          s.params.DecayToZero,
	})
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

func (s *peerApplicationScorer) start() {
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		ticker := s.clock.NewTicker(s.params.DecayInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.Ch():
				s.decayConnectedPeerScores()
			}
		}
	}()
}

func (s *peerApplicationScorer) stop() {
	s.cancelFunc()
	s.done.Wait()
}

type NoopApplicationScorer struct{}

func (n *NoopApplicationScorer) ApplicationScore(_ peer.ID) float64 {
	return 0
}

func (n *NoopApplicationScorer) onValidResponse(_ peer.ID) {
}

func (n *NoopApplicationScorer) onResponseError(_ peer.ID) {
}

func (n *NoopApplicationScorer) onRejectedPayload(_ peer.ID) {
}

func (n *NoopApplicationScorer) start() {
}

func (n *NoopApplicationScorer) stop() {
}

var _ ApplicationScorer = (*NoopApplicationScorer)(nil)
