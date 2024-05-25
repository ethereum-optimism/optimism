package gating

import (
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

func (g *ScoringConnectionGater) checkScore(p peer.ID) (allow bool) {
	score, err := g.scores.GetPeerScore(p)
	if err != nil {
		return false
	}
	return score >= g.minScore
}

func (g *ScoringConnectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	return g.BlockingConnectionGater.InterceptPeerDial(p) && g.checkScore(p)
}

func (g *ScoringConnectionGater) InterceptAddrDial(id peer.ID, ma multiaddr.Multiaddr) (allow bool) {
	return g.BlockingConnectionGater.InterceptAddrDial(id, ma) && g.checkScore(id)
}

func (g *ScoringConnectionGater) InterceptSecured(dir network.Direction, id peer.ID, mas network.ConnMultiaddrs) (allow bool) {
	return g.BlockingConnectionGater.InterceptSecured(dir, id, mas) && g.checkScore(id)
}
