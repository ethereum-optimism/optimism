package store

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"
)

func TestGetUnknownIPBan(t *testing.T) {
	book := createMemoryIPBanBook(t)
	defer book.Close()
	exp, err := book.GetIPBanExpiration(net.IPv4(1, 2, 3, 4))
	require.Same(t, ErrUnknownBan, err)
	require.Equal(t, time.Time{}, exp)
}

func TestRoundTripIPBan(t *testing.T) {
	book := createMemoryIPBanBook(t)
	defer book.Close()
	expiry := time.Unix(2484924, 0)
	ip := net.IPv4(1, 2, 3, 4)
	require.NoError(t, book.SetIPBanExpiration(ip, expiry))
	result, err := book.GetIPBanExpiration(ip)
	require.NoError(t, err)
	require.Equal(t, result, expiry)
}

func createMemoryIPBanBook(t *testing.T) *ipBanBook {
	store := sync.MutexWrap(ds.NewMapDatastore())
	logger := testlog.Logger(t, log.LevelInfo)
	c := clock.NewDeterministicClock(time.UnixMilli(100))
	book, err := newIPBanBook(context.Background(), logger, c, store)
	require.NoError(t, err)
	return book
}
