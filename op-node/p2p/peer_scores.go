package p2p

import (
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
)

// ConfigurePeerScoring configures the peer scoring parameters for the pubsub
func ConfigurePeerScoring(h host.Host, g ConnectionGater, gossipConf GossipSetupConfigurables, m GossipMetricer, log log.Logger) []pubsub.Option {
	// If we want to completely disable scoring config here, we can use the [peerScoringParams]
	// to return early without returning any [pubsub.Option].
	peerScoreParams := gossipConf.PeerScoringParams()
	peerScoreThresholds := NewPeerScoreThresholds()
	scorer := NewScorer(g, h.Peerstore(), m, log)
	opts := []pubsub.Option{
		pubsub.WithPeerScore(peerScoreParams, &peerScoreThresholds),
		pubsub.WithPeerScoreInspect(scorer.SnapshotHook(), peerScoreInspectFrequency),
	}
	return opts
}
