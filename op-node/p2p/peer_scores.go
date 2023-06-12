package p2p

import (
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// ConfigurePeerScoring configures the peer scoring parameters for the pubsub
func ConfigurePeerScoring(gossipConf GossipSetupConfigurables, scorer Scorer, log log.Logger) []pubsub.Option {
	// If we want to completely disable scoring config here, we can use the [peerScoringParams]
	// to return early without returning any [pubsub.Option].
	peerScoreParams := gossipConf.PeerScoringParams()
	peerScoreThresholds := NewPeerScoreThresholds()
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
