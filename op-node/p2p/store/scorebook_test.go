package store

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
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
	err := store.SetScore(id, &GossipScores{Total: score})
	require.NoError(t, err)

	assertPeerScores(t, store, id, PeerScores{Gossip: GossipScores{Total: score}})
}

func TestUpdateGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	require.NoError(t, store.SetScore(id, &GossipScores{Total: 444.223}))
	require.NoError(t, store.SetScore(id, &GossipScores{Total: score}))

	assertPeerScores(t, store, id, PeerScores{Gossip: GossipScores{Total: score}})
}

func TestStoreScoresForMultiplePeers(t *testing.T) {
	id1 := peer.ID("aaaa")
	id2 := peer.ID("bbbb")
	store := createMemoryStore(t)
	score1 := 123.45
	score2 := 453.22
	require.NoError(t, store.SetScore(id1, &GossipScores{Total: score1}))
	require.NoError(t, store.SetScore(id2, &GossipScores{Total: score2}))

	assertPeerScores(t, store, id1, PeerScores{Gossip: GossipScores{Total: score1}})
	assertPeerScores(t, store, id2, PeerScores{Gossip: GossipScores{Total: score2}})
}

func TestPersistData(t *testing.T) {
	id := peer.ID("aaaa")
	score := 123.45
	backingStore := sync.MutexWrap(ds.NewMapDatastore())
	store := createPeerstoreWithBacking(t, backingStore)

	require.NoError(t, store.SetScore(id, &GossipScores{Total: score}))

	// Close and recreate a new store from the same backing
	require.NoError(t, store.Close())
	store = createPeerstoreWithBacking(t, backingStore)

	assertPeerScores(t, store, id, PeerScores{Gossip: GossipScores{Total: score}})
}

func TestCloseCompletes(t *testing.T) {
	store := createMemoryStore(t)
	require.NoError(t, store.Close())
}

func TestPrune(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	logger := testlog.Logger(t, log.LvlInfo)
	store := sync.MutexWrap(ds.NewMapDatastore())
	clock := clock.NewDeterministicClock(time.UnixMilli(1000))
	book, err := newScoreBook(ctx, logger, clock, store)
	require.NoError(t, err)

	hasScoreRecorded := func(id peer.ID) bool {
		scores, err := book.GetPeerScores(id)
		require.NoError(t, err)
		return scores != PeerScores{}
	}

	firstStore := clock.Now()
	// Set some scores all 30 minutes apart so they have different expiry times
	require.NoError(t, book.SetScore("aaaa", &GossipScores{Total: 123.45}))
	clock.AdvanceTime(30 * time.Minute)
	require.NoError(t, book.SetScore("bbbb", &GossipScores{Total: 123.45}))
	clock.AdvanceTime(30 * time.Minute)
	require.NoError(t, book.SetScore("cccc", &GossipScores{Total: 123.45}))
	clock.AdvanceTime(30 * time.Minute)
	require.NoError(t, book.SetScore("dddd", &GossipScores{Total: 123.45}))
	clock.AdvanceTime(30 * time.Minute)

	// Update bbbb again which should extend its expiry
	require.NoError(t, book.SetScore("bbbb", &GossipScores{Total: 123.45}))

	require.True(t, hasScoreRecorded("aaaa"))
	require.True(t, hasScoreRecorded("bbbb"))
	require.True(t, hasScoreRecorded("cccc"))
	require.True(t, hasScoreRecorded("dddd"))

	elapsedTime := clock.Now().Sub(firstStore)
	timeToFirstExpiry := expiryPeriod - elapsedTime
	// Advance time until the score for aaaa should be pruned.
	clock.AdvanceTime(timeToFirstExpiry + 1)
	require.NoError(t, book.prune())
	// Clear the cache so reads have to come from the database
	book.cache.Purge()
	require.False(t, hasScoreRecorded("aaaa"), "should have pruned aaaa record")

	// Advance time so cccc, dddd and the original bbbb entry should be pruned
	clock.AdvanceTime(90 * time.Minute)
	require.NoError(t, book.prune())
	// Clear the cache so reads have to come from the database
	book.cache.Purge()

	require.False(t, hasScoreRecorded("cccc"), "should have pruned cccc record")
	require.False(t, hasScoreRecorded("dddd"), "should have pruned cccc record")

	require.True(t, hasScoreRecorded("bbbb"), "should not prune bbbb record")
}

func TestPruneMultipleBatches(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	logger := testlog.Logger(t, log.LvlInfo)
	clock := clock.NewDeterministicClock(time.UnixMilli(1000))
	book, err := newScoreBook(ctx, logger, clock, sync.MutexWrap(ds.NewMapDatastore()))
	require.NoError(t, err)

	hasScoreRecorded := func(id peer.ID) bool {
		scores, err := book.GetPeerScores(id)
		require.NoError(t, err)
		return scores != PeerScores{}
	}

	// Set scores for more peers than the max batch size
	peerCount := maxPruneBatchSize*3 + 5
	for i := 0; i < peerCount; i++ {
		require.NoError(t, book.SetScore(peer.ID(strconv.Itoa(i)), &GossipScores{Total: 123.45}))
	}
	clock.AdvanceTime(expiryPeriod + 1)
	require.NoError(t, book.prune())
	// Clear the cache so reads have to come from the database
	book.cache.Purge()

	for i := 0; i < peerCount; i++ {
		require.Falsef(t, hasScoreRecorded(peer.ID(strconv.Itoa(i))), "Should prune record peer %v", i)
	}
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
	logger := testlog.Logger(t, log.LvlInfo)
	clock := clock.NewDeterministicClock(time.UnixMilli(100))
	eps, err := NewExtendedPeerstore(context.Background(), logger, clock, ps, store)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = eps.Close()
	})
	return eps
}
