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
	res, err := store.SetScore(id, &GossipScores{Total: score})
	require.NoError(t, err)

	expected := PeerScores{Gossip: GossipScores{Total: score}}
	require.Equal(t, expected, res)

	assertPeerScores(t, store, id, expected)
}

func TestUpdateGossipScore(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	score := 123.45
	setScoreRequired(t, store, id, &GossipScores{Total: 444.223})
	setScoreRequired(t, store, id, &GossipScores{Total: score})

	assertPeerScores(t, store, id, PeerScores{Gossip: GossipScores{Total: score}})
}

func TestIncrementValidResponses(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	inc := IncrementValidResponses{Cap: 2.1}
	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ValidResponses: 1}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ValidResponses: 2}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ValidResponses: 2.1}})
}

func TestIncrementErrorResponses(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	inc := IncrementErrorResponses{Cap: 2.1}
	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ErrorResponses: 1}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ErrorResponses: 2}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{ErrorResponses: 2.1}})
}

func TestIncrementRejectedPayloads(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	inc := IncrementRejectedPayloads{Cap: 2.1}
	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{RejectedPayloads: 1}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{RejectedPayloads: 2}})

	setScoreRequired(t, store, id, inc)
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{RejectedPayloads: 2.1}})
}

func TestDecayApplicationScores(t *testing.T) {
	id := peer.ID("aaaa")
	store := createMemoryStore(t)
	for i := 0; i < 10; i++ {
		setScoreRequired(t, store, id, IncrementValidResponses{Cap: 100})
		setScoreRequired(t, store, id, IncrementErrorResponses{Cap: 100})
		setScoreRequired(t, store, id, IncrementRejectedPayloads{Cap: 100})
	}
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{
		ValidResponses:   10,
		ErrorResponses:   10,
		RejectedPayloads: 10,
	}})

	setScoreRequired(t, store, id, &DecayApplicationScores{
		ValidResponseDecay:   0.8,
		ErrorResponseDecay:   0.4,
		RejectedPayloadDecay: 0.5,
		DecayToZero:          0.1,
	})
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{
		ValidResponses:   10 * 0.8,
		ErrorResponses:   10 * 0.4,
		RejectedPayloads: 10 * 0.5,
	}})

	// Should be set to exactly zero when below DecayToZero
	setScoreRequired(t, store, id, &DecayApplicationScores{
		ValidResponseDecay:   0.8,
		ErrorResponseDecay:   0.4,
		RejectedPayloadDecay: 0.5,
		DecayToZero:          5,
	})
	assertPeerScores(t, store, id, PeerScores{ReqResp: ReqRespScores{
		ValidResponses:   10 * 0.8 * 0.8, // Not yet below 5 so preserved
		ErrorResponses:   0,
		RejectedPayloads: 0,
	}})
}

func TestStoreScoresForMultiplePeers(t *testing.T) {
	id1 := peer.ID("aaaa")
	id2 := peer.ID("bbbb")
	store := createMemoryStore(t)
	score1 := 123.45
	score2 := 453.22
	setScoreRequired(t, store, id1, &GossipScores{Total: score1})
	setScoreRequired(t, store, id2, &GossipScores{Total: score2})

	assertPeerScores(t, store, id1, PeerScores{Gossip: GossipScores{Total: score1}})
	assertPeerScores(t, store, id2, PeerScores{Gossip: GossipScores{Total: score2}})
}

func TestPersistData(t *testing.T) {
	id := peer.ID("aaaa")
	score := 123.45
	backingStore := sync.MutexWrap(ds.NewMapDatastore())
	store := createPeerstoreWithBacking(t, backingStore)

	setScoreRequired(t, store, id, &GossipScores{Total: score})

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
	book, err := newScoreBook(ctx, logger, clock, store, 24*time.Hour)
	require.NoError(t, err)

	hasScoreRecorded := func(id peer.ID) bool {
		scores, err := book.GetPeerScores(id)
		require.NoError(t, err)
		return scores != PeerScores{}
	}

	firstStore := clock.Now()
	// Set some scores all 30 minutes apart so they have different expiry times
	setScoreRequired(t, book, "aaaa", &GossipScores{Total: 123.45})
	clock.AdvanceTime(30 * time.Minute)
	setScoreRequired(t, book, "bbbb", &GossipScores{Total: 123.45})
	clock.AdvanceTime(30 * time.Minute)
	setScoreRequired(t, book, "cccc", &GossipScores{Total: 123.45})
	clock.AdvanceTime(30 * time.Minute)
	setScoreRequired(t, book, "dddd", &GossipScores{Total: 123.45})
	clock.AdvanceTime(30 * time.Minute)

	// Update bbbb again which should extend its expiry
	setScoreRequired(t, book, "bbbb", &GossipScores{Total: 123.45})

	require.True(t, hasScoreRecorded("aaaa"))
	require.True(t, hasScoreRecorded("bbbb"))
	require.True(t, hasScoreRecorded("cccc"))
	require.True(t, hasScoreRecorded("dddd"))

	elapsedTime := clock.Now().Sub(firstStore)
	timeToFirstExpiry := book.book.recordExpiry - elapsedTime
	// Advance time until the score for aaaa should be pruned.
	clock.AdvanceTime(timeToFirstExpiry + 1)
	require.NoError(t, book.book.prune())
	// Clear the cache so reads have to come from the database
	book.book.cache.Purge()
	require.False(t, hasScoreRecorded("aaaa"), "should have pruned aaaa record")

	// Advance time so cccc, dddd and the original bbbb entry should be pruned
	clock.AdvanceTime(90 * time.Minute)
	require.NoError(t, book.book.prune())
	// Clear the cache so reads have to come from the database
	book.book.cache.Purge()

	require.False(t, hasScoreRecorded("cccc"), "should have pruned cccc record")
	require.False(t, hasScoreRecorded("dddd"), "should have pruned cccc record")

	require.True(t, hasScoreRecorded("bbbb"), "should not prune bbbb record")
}

func TestPruneMultipleBatches(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	logger := testlog.Logger(t, log.LvlInfo)
	clock := clock.NewDeterministicClock(time.UnixMilli(1000))
	book, err := newScoreBook(ctx, logger, clock, sync.MutexWrap(ds.NewMapDatastore()), 24*time.Hour)
	require.NoError(t, err)

	hasScoreRecorded := func(id peer.ID) bool {
		scores, err := book.GetPeerScores(id)
		require.NoError(t, err)
		return scores != PeerScores{}
	}

	// Set scores for more peers than the max batch size
	peerCount := maxPruneBatchSize*3 + 5
	for i := 0; i < peerCount; i++ {
		setScoreRequired(t, book, peer.ID(strconv.Itoa(i)), &GossipScores{Total: 123.45})
	}
	clock.AdvanceTime(book.book.recordExpiry + 1)
	require.NoError(t, book.book.prune())
	// Clear the cache so reads have to come from the database
	book.book.cache.Purge()

	for i := 0; i < peerCount; i++ {
		require.Falsef(t, hasScoreRecorded(peer.ID(strconv.Itoa(i))), "Should prune record peer %v", i)
	}
}

// Check that scores that are eligible for pruning are not returned, even if they haven't yet been removed
func TestIgnoreOutdatedScores(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	logger := testlog.Logger(t, log.LvlInfo)
	clock := clock.NewDeterministicClock(time.UnixMilli(1000))
	retentionPeriod := 24 * time.Hour
	book, err := newScoreBook(ctx, logger, clock, sync.MutexWrap(ds.NewMapDatastore()), retentionPeriod)
	require.NoError(t, err)

	setScoreRequired(t, book, "a", &GossipScores{Total: 123.45})
	clock.AdvanceTime(retentionPeriod + 1)

	// Not available from cache
	scores, err := book.GetPeerScores("a")
	require.NoError(t, err)
	require.Equal(t, scores, PeerScores{})

	book.book.cache.Purge()
	// Not available from disk
	scores, err = book.GetPeerScores("a")
	require.NoError(t, err)
	require.Equal(t, scores, PeerScores{})
}

func assertPeerScores(t *testing.T, store ExtendedPeerstore, id peer.ID, expected PeerScores) {
	result, err := store.GetPeerScores(id)
	require.NoError(t, err)
	require.Equal(t, result, expected)

	score, err := store.GetPeerScore(id)
	require.NoError(t, err)
	require.Equal(t, expected.Gossip.Total, score)
}

func createMemoryStore(t *testing.T) ExtendedPeerstore {
	store := sync.MutexWrap(ds.NewMapDatastore())
	return createPeerstoreWithBacking(t, store)
}

func createPeerstoreWithBacking(t *testing.T, store *sync.MutexDatastore) ExtendedPeerstore {
	ps, err := pstoreds.NewPeerstore(context.Background(), store, pstoreds.DefaultOpts())
	require.NoError(t, err, "Failed to create peerstore")
	logger := testlog.Logger(t, log.LvlInfo)
	c := clock.NewDeterministicClock(time.UnixMilli(100))
	eps, err := NewExtendedPeerstore(context.Background(), logger, c, ps, store, 24*time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = eps.Close()
	})
	return eps
}

func setScoreRequired(t *testing.T, store ScoreDatastore, id peer.ID, diff ScoreDiff) {
	_, err := store.SetScore(id, diff)
	require.NoError(t, err)
}
