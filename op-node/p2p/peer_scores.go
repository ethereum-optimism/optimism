package p2p

import (
	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// ConfigurePeerScoring configures the peer scoring parameters for the pubsub
func ConfigurePeerScoring(g ConnectionGater, ps store.ExtendedPeerstore, gossipConf GossipSetupConfigurables, m GossipMetricer, log log.Logger) []pubsub.Option {
	// If we want to completely disable scoring config here, we can use the [peerScoringParams]
	// to return early without returning any [pubsub.Option].
	peerScoreParams := gossipConf.PeerScoringParams()
	peerScoreThresholds := NewPeerScoreThresholds()
	banEnabled := gossipConf.BanPeers()
	peerGater := NewPeerGater(g, log, banEnabled)
	scorer := NewScorer(peerGater, ps, m, gossipConf.PeerBandScorer(), log)
	opts := []pubsub.Option{}
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
