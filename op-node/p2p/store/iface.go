package store

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type PeerScores struct {
	Gossip float64
}

type ScoreType int

const (
	TypeGossip ScoreType = iota
)

// ScoreDatastore defines a type-safe API for getting and setting libp2p peer score information
type ScoreDatastore interface {
	// GetPeerScores returns the current scores for the specified peer
	GetPeerScores(id peer.ID) (PeerScores, error)

	// SetScore stores the latest score for the specified peer and score type
	SetScore(id peer.ID, scoreType ScoreType, score float64) error
}

// ExtendedPeerstore defines a type-safe API to work with additional peer metadata based on a libp2p peerstore.Peerstore
type ExtendedPeerstore interface {
	peerstore.Peerstore
	ScoreDatastore
	peerstore.CertifiedAddrBook
}
