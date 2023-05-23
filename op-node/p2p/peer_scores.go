package p2p

import (
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
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
	banEnabled := gossipConf.BanPeers()
	peerGater := NewPeerGater(g, log, banEnabled)
	opts := []pubsub.Option{}
	eps, ok := h.Peerstore().(store.ExtendedPeerstore)
	if !ok {
		log.Warn("Disabling peer scoring. Peerstore does not support peer scores")
		return opts
	}
	scorer := NewScorer(peerGater, eps, m, gossipConf.PeerBandScorer(), log)
	// Check the app specific score since libp2p doesn't export it's [validate] function :/
	if peerScoreParams != nil && peerScoreParams.AppSpecificScore != nil {
		opts = []pubsub.Option{
			pubsub.WithPeerScore(peerScoreParams, &peerScoreThresholds),
			pubsub.WithPeerScoreInspect(scorer.SnapshotHook(), peerScoreInspectFrequency),
		}
	} else {
		log.Warn("Proceeding with no peer scoring...\nMissing AppSpecificScore in peer scoring params")
	}
	return opts
}
