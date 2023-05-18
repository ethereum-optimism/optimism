package store

import (
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type datastore struct {
	peerstore.Peerstore
	peerstore.CertifiedAddrBook
}

type ScoreType string

const (
	gossipType ScoreType = "gossip"
)

func NewExtendedPeerstore(ps peerstore.Peerstore, store ds.Batching) (ExtendedPeerstore, error) {
	cab, ok := peerstore.GetCertifiedAddrBook(ps)
	if !ok {
		return nil, errors.New("peerstore should also be a certified address book")
	}
	return &datastore{
		Peerstore:         ps,
		CertifiedAddrBook: cab,
	}, nil
}

func (d *datastore) GetPeerScores(id peer.ID) (PeerScores, error) {
	scores := PeerScores{}
	if score, err := d.loadScoreComponent(id, gossipType); err != nil {
		return PeerScores{}, fmt.Errorf("load gossip score: %w", err)
	} else {
		scores.gossip = score
	}
	return scores, nil
}

func (d *datastore) loadScoreComponent(id peer.ID, scoreType ScoreType) (float64, error) {
	if val, err := d.Get(id, scoreKey(scoreType)); errors.Is(err, peerstore.ErrNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		score, ok := val.(float64)
		if !ok {
			return 0, fmt.Errorf("stored score of type %v was not a float64", scoreType)
		}
		return score, nil
	}
}

func (d *datastore) SetGossipScore(id peer.ID, score float64) error {
	return d.Put(id, scoreKey(gossipType), score)
}

func scoreKey(scoreType ScoreType) string {
	return fmt.Sprintf("score-%v", scoreType)
}

var _ ExtendedPeerstore = (*datastore)(nil)
