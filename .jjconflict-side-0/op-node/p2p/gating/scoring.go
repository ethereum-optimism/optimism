package gating

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

//go:generate mockery --name Scores --output mocks/ --with-expecter=true
type Scores interface {
	GetPeerScore(id peer.ID) (float64, error)
}

// ScoringConnectionGater enhances a ConnectionGater by enforcing a minimum score for peer connections
type ScoringConnectionGater struct {
	BlockingConnectionGater
	scores   Scores
	minScore float64
}

func AddScoring(gater BlockingConnectionGater, scores Scores, minScore float64) *ScoringConnectionGater {
	return &ScoringConnectionGater{BlockingConnectionGater: gater, scores: scores, minScore: minScore}
}

func (g *ScoringConnectionGater) checkScore(p peer.ID) (allow bool, score float64) {
	score, err := g.scores.GetPeerScore(p)
	if err != nil {
		return false, score
	}
	return score >= g.minScore, score
}

func (g *ScoringConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	if !g.BlockingConnectionGater.InterceptPeerDial(p) {
		return false
	}
	check, score := g.checkScore(p)
	if !check {
		log.Warn("peer has failed checkScore", "peer_id", p, "score", score, "min_score", g.minScore)
	}
	return check
}

func (g *ScoringConnectionGater) InterceptAddrDial(id peer.ID, ma multiaddr.Multiaddr) (allow bool) {
	if !g.BlockingConnectionGater.InterceptAddrDial(id, ma) {
		return false
	}
	check, score := g.checkScore(id)
	if !check {
		log.Warn("peer has failed checkScore", "peer_id", id, "score", score, "min_score", g.minScore)
	}
	return check
}

func (g *ScoringConnectionGater) InterceptSecured(dir network.Direction, id peer.ID, mas network.ConnMultiaddrs) (allow bool) {
	if !g.BlockingConnectionGater.InterceptSecured(dir, id, mas) {
		return false
	}
	check, score := g.checkScore(id)
	if !check {
		log.Warn("peer has failed checkScore", "peer_id", id, "score", score, "min_score", g.minScore)
	}
	return check
}
