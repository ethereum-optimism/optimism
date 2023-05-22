package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
)

type scoreBook struct {
	ctx   context.Context
	store ds.Batching
	cache *lru.Cache[peer.ID, PeerScores]
	sync.RWMutex
}

var scoresBase = ds.NewKey("/peers/scores")

const (
	scoreDataV0    = "0"
	scoreCacheSize = 100
)

func newScoreBook(ctx context.Context, store ds.Batching) (*scoreBook, error) {
	cache, err := lru.New[peer.ID, PeerScores](scoreCacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}
	return &scoreBook{
		ctx:   ctx,
		store: store,
		cache: cache,
	}, nil
}

func (d *scoreBook) GetPeerScores(id peer.ID) (PeerScores, error) {
	d.RLock()
	defer d.RUnlock()
	return d.getPeerScoresNoLock(id)
}

func (d *scoreBook) getPeerScoresNoLock(id peer.ID) (PeerScores, error) {
	scores, ok := d.cache.Get(id)
	if ok {
		return scores, nil
	}
	data, err := d.store.Get(d.ctx, scoreKey(id))
	if errors.Is(err, ds.ErrNotFound) {
		return PeerScores{}, nil
	} else if err != nil {
		return PeerScores{}, fmt.Errorf("load scores for peer %v: %w", id, err)
	}
	scores, err = deserializeScoresV0(data)
	if err != nil {
		return PeerScores{}, fmt.Errorf("invalid score data for peer %v: %w", id, err)
	}
	d.cache.Add(id, scores)
	return scores, nil
}

func (d *scoreBook) SetScore(id peer.ID, scoreType ScoreType, score float64) error {
	d.Lock()
	defer d.Unlock()
	scores, err := d.getPeerScoresNoLock(id)
	if err != nil {
		return err
	}
	switch scoreType {
	case TypeGossip:
		scores.Gossip = score
	default:
		return fmt.Errorf("unknown score type: %v", scoreType)
	}
	data, err := serializeScoresV0(scores)
	if err != nil {
		return fmt.Errorf("encode scores for peer %v: %w", id, err)
	}
	err = d.store.Put(d.ctx, scoreKey(id), data)
	if err != nil {
		return fmt.Errorf("storing updated scores for peer %v: %w", id, err)
	}
	d.cache.Add(id, scores)
	return nil
}

func scoreKey(id peer.ID) ds.Key {
	return scoresBase.ChildString(base32.RawStdEncoding.EncodeToString([]byte(id))).ChildString(scoreDataV0)
}
