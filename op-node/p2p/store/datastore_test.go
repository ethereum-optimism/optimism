package store

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	"github.com/stretchr/testify/require"
)

func TestGetEmptyScoreComponents(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	result, err := store.GetPeerScores(id)
	require.NoError(t, err)
	require.Equal(t, result, PeerScores{})
}

func TestRoundTripGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	err := store.SetGossipScore(id, score)
	require.NoError(t, err)

	elements, err := store.GetPeerScores(id)
	require.NoError(t, err)
	require.Equal(t, elements, PeerScores{gossip: score})
}

func TestUpdateGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	require.NoError(t, store.SetGossipScore(id, 444.223))
	require.NoError(t, store.SetGossipScore(id, score))

	result, err := store.GetPeerScores(id)
	require.NoError(t, err)
	require.Equal(t, result, PeerScores{gossip: score})
}

func createMemoryStore(t *testing.T) ExtendedPeerstore {
	store := sync.MutexWrap(ds.NewMapDatastore())
	ps, err := pstoreds.NewPeerstore(context.Background(), store, pstoreds.DefaultOpts())
	require.NoError(t, err, "Failed to create peerstore")
	eps, err := NewExtendedPeerstore(ps, store)
	require.NoError(t, err)
	return eps
}
