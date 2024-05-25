package store

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"
)

func TestGetUnknownPeerBan(t *testing.T) {
	book := createMemoryPeerBanBook(t)
	defer book.Close()
	exp, err := book.GetPeerBanExpiration("a")
	require.Same(t, UnknownBanErr, err)
	require.Equal(t, time.Time{}, exp)
}

func TestRoundTripPeerBan(t *testing.T) {
	book := createMemoryPeerBanBook(t)
	defer book.Close()
	expiry := time.Unix(2484924, 0)
	require.NoError(t, book.SetPeerBanExpiration("a", expiry))
	result, err := book.GetPeerBanExpiration("a")
	require.NoError(t, err)
	require.Equal(t, result, expiry)
}

func createMemoryPeerBanBook(t *testing.T) *peerBanBook {
	store := sync.MutexWrap(ds.NewMapDatastore())
	logger := testlog.Logger(t, log.LvlInfo)
	c := clock.NewDeterministicClock(time.UnixMilli(100))
	book, err := newPeerBanBook(context.Background(), logger, c, store)
	require.NoError(t, err)
	return book
}
