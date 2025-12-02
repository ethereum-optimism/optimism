package p2p

import (
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// ConfigurePeerScoring configures the peer scoring parameters for the pubsub
func ConfigurePeerScoring(gossipConf GossipSetupConfigurables, scorer Scorer, log log.Logger) []pubsub.Option {
	// If we want to completely disable scoring config here, we can use the [peerScoringParams]
	// to return early without returning any [pubsub.Option].
	scoreParams := gossipConf.PeerScoringParams()
	opts := []pubsub.Option{}
	if scoreParams != nil {
		peerScoreThresholds := NewPeerScoreThresholds()
		// Create copy of params before modifying the AppSpecificScore
		params := scoreParams.PeerScoring
		params.AppSpecificScore = scorer.ApplicationScore
		opts = []pubsub.Option{
			pubsub.WithPeerScore(&params, &peerScoreThresholds),
			pubsub.WithPeerScoreInspect(scorer.SnapshotHook(), peerScoreInspectFrequency),
		}
	} else {
		log.Info("Peer scoring disabled")
	}
	return opts
}
