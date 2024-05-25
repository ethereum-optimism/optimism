package gating

import (
	"net"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p/gating/mocks"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	log "github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func expiryTestSetup(t *testing.T) (*clock.DeterministicClock, *mocks.ExpiryStore, *mocks.BlockingConnectionGater, *ExpiryConnectionGater) {
	mockGater := mocks.NewBlockingConnectionGater(t)
	log := testlog.Logger(t, log.LvlError)
	cl := clock.NewDeterministicClock(time.Now())
	mockExpiryStore := mocks.NewExpiryStore(t)
	gater := AddBanExpiry(mockGater, mockExpiryStore, log, cl, metrics.NoopMetrics)
	return cl, mockExpiryStore, mockGater, gater
}

func TestExpiryConnectionGater_InterceptPeerDial(t *testing.T) {
	mallory := peer.ID("malllory")
	t.Run("expired peer ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptPeerDial(mallory).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetPeerBanExpiration(mallory, time.Time{}).Return(nil)
		allow := gater.InterceptPeerDial(mallory)
		require.True(t, allow)
	})
	t.Run("active peer ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptPeerDial(mallory).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptPeerDial(mallory)
		require.False(t, allow)
	})
	t.Run("unknown expiring ban", func(t *testing.T) {
		_, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptPeerDial(mallory).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(time.Time{}, store.UnknownBanErr)
		allow := gater.InterceptPeerDial(mallory)
		require.True(t, allow)
	})
	t.Run("inner ban", func(t *testing.T) {
		_, _, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptPeerDial(mallory).Return(false)
		allow := gater.InterceptPeerDial(mallory)
		require.False(t, allow)
	})
}

func TestExpiryConnectionGater_InterceptAddrDial(t *testing.T) {
	ip := net.IPv4(1, 2, 3, 4)
	mallory := peer.ID("7y9Qv7mG2h6fnzcDkeqVsEvW2rU9PdybSZ8y1dCrB9p")
	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/9000")
	require.NoError(t, err)

	t.Run("expired IP ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(time.Time{}, store.UnknownBanErr)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetIPBanExpiration(ip.To4(), time.Time{}).Return(nil)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.True(t, allow)
	})
	t.Run("active IP ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(time.Time{}, store.UnknownBanErr)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.False(t, allow)
	})
	t.Run("unknown IP ban expiry", func(t *testing.T) {
		_, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(time.Time{}, store.UnknownBanErr)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(time.Time{}, store.UnknownBanErr)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.True(t, allow)
	})
	t.Run("inner ban", func(t *testing.T) {
		_, _, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(false)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.False(t, allow)
	})

	t.Run("expired peer ban but active ip ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetPeerBanExpiration(mallory, time.Time{}).Return(nil)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.False(t, allow)
	})
	t.Run("active peer ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptAddrDial(mallory, addr)
		require.False(t, allow)
	})
	t.Run("expired peer ban and expired ip ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAddrDial(mallory, addr).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetPeerBanExpiration(mallory, time.Time{}).Return(nil)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetIPBanExpiration(ip.To4(), time.Time{}).Return(nil)

		allow := gater.InterceptAddrDial(mallory, addr)
		require.True(t, allow)
	})
}

type localRemoteAddrs struct {
	local  multiaddr.Multiaddr
	remote multiaddr.Multiaddr
}

func (l localRemoteAddrs) LocalMultiaddr() multiaddr.Multiaddr {
	return l.local
}

func (l localRemoteAddrs) RemoteMultiaddr() multiaddr.Multiaddr {
	return l.remote
}

var _ network.ConnMultiaddrs = localRemoteAddrs{}

func TestExpiryConnectionGater_InterceptAccept(t *testing.T) {
	ip := net.IPv4(1, 2, 3, 4)
	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/9000")
	require.NoError(t, err)
	mas := localRemoteAddrs{remote: addr}

	t.Run("expired IP ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAccept(mas).Return(true)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetIPBanExpiration(ip.To4(), time.Time{}).Return(nil)
		allow := gater.InterceptAccept(mas)
		require.True(t, allow)
	})
	t.Run("active IP ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAccept(mas).Return(true)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptAccept(mas)
		require.False(t, allow)
	})
	t.Run("unknown expiry", func(t *testing.T) {
		_, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAccept(mas).Return(true)
		mockExpiryStore.EXPECT().GetIPBanExpiration(ip.To4()).Return(time.Time{}, store.UnknownBanErr)
		allow := gater.InterceptAccept(mas)
		require.True(t, allow)
	})
	t.Run("inner ban", func(t *testing.T) {
		_, _, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptAccept(mas).Return(false)
		allow := gater.InterceptAccept(mas)
		require.False(t, allow)
	})
}

func TestExpiryConnectionGater_InterceptSecured(t *testing.T) {
	mallory := peer.ID("7y9Qv7mG2h6fnzcDkeqVsEvW2rU9PdybSZ8y1dCrB9p")
	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/9000")
	require.NoError(t, err)
	mas := localRemoteAddrs{remote: addr}

	t.Run("expired peer ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptSecured(network.DirInbound, mallory, mas).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(-time.Second), nil)
		mockExpiryStore.EXPECT().SetPeerBanExpiration(mallory, time.Time{}).Return(nil)
		allow := gater.InterceptSecured(network.DirInbound, mallory, mas)
		require.True(t, allow)
	})
	t.Run("active peer ban", func(t *testing.T) {
		cl, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptSecured(network.DirInbound, mallory, mas).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(cl.Now().Add(time.Second), nil)
		allow := gater.InterceptSecured(network.DirInbound, mallory, mas)
		require.False(t, allow)
	})
	t.Run("unknown expiry", func(t *testing.T) {
		_, mockExpiryStore, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptSecured(network.DirInbound, mallory, mas).Return(true)
		mockExpiryStore.EXPECT().GetPeerBanExpiration(mallory).Return(time.Time{}, store.UnknownBanErr)
		allow := gater.InterceptSecured(network.DirInbound, mallory, mas)
		require.True(t, allow)
	})
	t.Run("inner ban", func(t *testing.T) {
		_, _, mockGater, gater := expiryTestSetup(t)
		mockGater.EXPECT().InterceptSecured(network.DirInbound, mallory, mas).Return(false)
		allow := gater.InterceptSecured(network.DirInbound, mallory, mas)
		require.False(t, allow)
	})
	t.Run("accept outbound", func(t *testing.T) {
		_, _, _, gater := expiryTestSetup(t)
		allow := gater.InterceptSecured(network.DirOutbound, mallory, mas)
		require.True(t, allow)
	})
}
