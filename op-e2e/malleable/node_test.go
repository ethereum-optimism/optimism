package malleable

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	log "github.com/ethereum/go-ethereum/log"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	require "github.com/stretchr/testify/require"

	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestMalleable_NewMalleable tests constructing a new [Malleable] node.
func TestMalleable_NewMalleable(t *testing.T) {
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	l := testlog.Logger(t, log.LvlInfo)
	cid := big.NewInt(420)

	m, err := NewMalleable(l, cid, nil, p)
	require.NoError(t, err, "failed to create new malleable node")
	require.Empty(t, m.blocksTopic.ListPeers())

	p2, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate a second p2p priv key")
	m2, err := NewMalleable(l, cid, nil, p2)
	require.NoError(t, err, "failed to create new malleable node")
	require.Empty(t, m2.blocksTopic.ListPeers())

	err = m.h.Connect(context.Background(), peer.AddrInfo{ID: m2.ID(), Addrs: m2.Addrs()})
	require.NoError(t, err, "failed to connect the hosts")
	require.Equal(t, m.h.Network().Connectedness(m2.ID()), network.Connected)
	require.Equal(t, m2.h.Network().Connectedness(m.ID()), network.Connected)
	require.Equal(t, m.h.Peerstore().Peers().Len(), 2)
	require.Equal(t, m2.h.Peerstore().Peers().Len(), 2)
}
