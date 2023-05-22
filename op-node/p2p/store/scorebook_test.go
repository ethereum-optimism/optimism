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
	assertPeerScores(t, store, id, PeerScores{})
}

func TestRoundTripGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	err := store.SetScore(id, TypeGossip, score)
	require.NoError(t, err)

	assertPeerScores(t, store, id, PeerScores{Gossip: score})
}

func TestUpdateGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	require.NoError(t, store.SetScore(id, TypeGossip, 444.223))
	require.NoError(t, store.SetScore(id, TypeGossip, score))

	assertPeerScores(t, store, id, PeerScores{Gossip: score})
}

func TestStoreScoresForMultiplePeers(t *testing.T) {
	id1 := peer.ID("aaaa")
	id2 := peer.ID("bbbb")
	store := createMemoryStore(t)
	score1 := 123.45
	score2 := 453.22
	require.NoError(t, store.SetScore(id1, TypeGossip, score1))
	require.NoError(t, store.SetScore(id2, TypeGossip, score2))

	assertPeerScores(t, store, id1, PeerScores{Gossip: score1})
	assertPeerScores(t, store, id2, PeerScores{Gossip: score2})
}

func TestPersistData(t *testing.T) {
	id := peer.ID("aaaa")
	score := 123.45
	backingStore := sync.MutexWrap(ds.NewMapDatastore())
	store := createPeerstoreWithBacking(t, backingStore)

	require.NoError(t, store.SetScore(id, TypeGossip, score))

	// Close and recreate a new store from the same backing
	require.NoError(t, store.Close())
	store = createPeerstoreWithBacking(t, backingStore)

	assertPeerScores(t, store, id, PeerScores{Gossip: score})
}

func TestUnknownScoreType(t *testing.T) {
	store := createMemoryStore(t)
	err := store.SetScore("aaaa", 92832, 244.24)
	require.ErrorContains(t, err, "unknown score type")
}

func assertPeerScores(t *testing.T, store ExtendedPeerstore, id peer.ID, expected PeerScores) {
	result, err := store.GetPeerScores(id)
	require.NoError(t, err)
	require.Equal(t, result, expected)
}

func createMemoryStore(t *testing.T) ExtendedPeerstore {
	store := sync.MutexWrap(ds.NewMapDatastore())
	return createPeerstoreWithBacking(t, store)
}

func createPeerstoreWithBacking(t *testing.T, store *sync.MutexDatastore) ExtendedPeerstore {
	ps, err := pstoreds.NewPeerstore(context.Background(), store, pstoreds.DefaultOpts())
	require.NoError(t, err, "Failed to create peerstore")
	eps, err := NewExtendedPeerstore(context.Background(), ps, store)
	require.NoError(t, err)
	return eps
}
