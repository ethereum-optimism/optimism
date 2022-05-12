package p2p

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/internal/testlog"
	"github.com/ethereum/go-ethereum/log"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	tswarm "github.com/libp2p/go-libp2p-swarm/testing"
	yamux "github.com/libp2p/go-libp2p-yamux"
	lconf "github.com/libp2p/go-libp2p/config"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
)

func TestingConfig(t *testing.T) *Config {
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	mtpt, err := lconf.MuxerConstructor(yamux.DefaultTransport)
	require.NoError(t, err)
	mux := lconf.MsMuxC{MuxC: mtpt, ID: "/yamux/1.0.0"}

	return &Config{
		Priv:                (*ecdsa.PrivateKey)((p).(*crypto.Secp256k1PrivateKey)),
		DisableP2P:          false,
		NoDiscovery:         true, // we statically peer during most tests.
		ListenIP:            net.IP{127, 0, 0, 1},
		ListenTCPPort:       0, // bind to any available port
		StaticPeers:         nil,
		HostMux:             []lconf.MsMuxC{mux},
		NoTransportSecurity: true,
		PeersLo:             1,
		PeersHi:             10,
		PeersGrace:          time.Second * 10,
		NAT:                 false,
		UserAgent:           "optimism-testing",
		TimeoutNegotiation:  time.Second * 2,
		TimeoutAccept:       time.Second * 2,
		TimeoutDial:         time.Second * 2,
		Store:               sync.MutexWrap(ds.NewMapDatastore()),
		ConnGater: func(conf *Config) (connmgr.ConnectionGater, error) {
			return tswarm.DefaultMockConnectionGater(), nil
		},
		ConnMngr: DefaultConnManager,
	}
}

// Simplified p2p test, to debug/test basic libp2p things with
func TestP2PSimple(t *testing.T) {
	confA := TestingConfig(t)
	confB := TestingConfig(t)
	hostA, err := confA.Host(testlog.Logger(t, log.LvlError).New("host", "A"))
	require.NoError(t, err, "failed to launch host A")
	defer hostA.Close()
	hostB, err := confB.Host(testlog.Logger(t, log.LvlError).New("host", "B"))
	require.NoError(t, err, "failed to launch host B")
	defer hostB.Close()
	err = hostA.Connect(context.Background(), peer.AddrInfo{ID: hostB.ID(), Addrs: hostB.Addrs()})
	require.NoError(t, err, "failed to connect to peer B from peer A")
	require.Equal(t, hostB.Network().Connectedness(hostA.ID()), network.Connected)
}

// Full setup, using negotiated transport security and muxes
func TestP2PFull(t *testing.T) {
	pA, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	pB, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")

	mplexC, err := mplexC()
	require.NoError(t, err)
	yamuxC, err := yamuxC()
	require.NoError(t, err)
	noiseC, err := noiseC()
	require.NoError(t, err)
	tlsC, err := tlsC()
	require.NoError(t, err)

	confA := Config{
		Priv:                (*ecdsa.PrivateKey)((pA).(*crypto.Secp256k1PrivateKey)),
		DisableP2P:          false,
		NoDiscovery:         true,
		ListenIP:            net.IP{127, 0, 0, 1},
		ListenTCPPort:       0, // bind to any available port
		StaticPeers:         nil,
		HostMux:             []lconf.MsMuxC{yamuxC, mplexC},
		HostSecurity:        []lconf.MsSecC{noiseC, tlsC},
		NoTransportSecurity: false,
		PeersLo:             1,
		PeersHi:             10,
		PeersGrace:          time.Second * 10,
		NAT:                 false,
		UserAgent:           "optimism-testing",
		TimeoutNegotiation:  time.Second * 2,
		TimeoutAccept:       time.Second * 2,
		TimeoutDial:         time.Second * 2,
		Store:               sync.MutexWrap(ds.NewMapDatastore()),
		ConnGater:           DefaultConnGater,
		ConnMngr:            DefaultConnManager,
	}
	// copy config A, and change the settings for B
	confB := confA
	confB.Priv = (*ecdsa.PrivateKey)((pB).(*crypto.Secp256k1PrivateKey))
	confB.Store = sync.MutexWrap(ds.NewMapDatastore())
	// TODO: maybe swap the order of sec/mux preferences, to test that negotiation works

	hostA, err := confA.Host(testlog.Logger(t, log.LvlError).New("host", "A"))
	require.NoError(t, err)
	defer hostA.Close()

	conns := make(chan network.Conn, 1)
	hostA.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			conns <- conn
			t.Log("connected")
		}})

	// Set up B to connect statically
	confB.StaticPeers, err = peer.AddrInfoToP2pAddrs(&peer.AddrInfo{ID: hostA.ID(), Addrs: hostA.Addrs()})
	require.NoError(t, err)

	hostB, err := confB.Host(testlog.Logger(t, log.LvlError).New("host", "B"))
	require.NoError(t, err)
	defer hostB.Close()

	select {
	case <-time.After(time.Second):
		t.Fatal("failed to connect new host")
	case c := <-conns:
		require.Equal(t, hostB.ID(), c.RemotePeer())
	}
}

// Most tests should use mocknets instead of using the actual local host network
func TestP2PMocknet(t *testing.T) {
	mnet, err := mocknet.FullMeshConnected(3)
	require.NoError(t, err, "failed to setup mocknet")
	defer mnet.Close()
	hosts := mnet.Hosts()
	hostA, hostB, hostC := hosts[0], hosts[1], hosts[2]
	require.Equal(t, hostA.Network().Connectedness(hostB.ID()), network.Connected)
	require.Equal(t, hostA.Network().Connectedness(hostC.ID()), network.Connected)
	require.Equal(t, hostB.Network().Connectedness(hostC.ID()), network.Connected)
}
