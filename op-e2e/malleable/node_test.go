package malleable

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	log "github.com/ethereum/go-ethereum/log"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	require "github.com/stretchr/testify/require"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestMalleable_NewMalleable tests constructing a new [Malleable] node.
func TestMalleable_NewMalleable(t *testing.T) {
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	l := testlog.Logger(t, log.LvlInfo)
	cid := big.NewInt(420)

	m, err := NewMalleable(l, cid, nil, p, OnUnsafeL2Payload)
	require.NoError(t, err, "failed to create new malleable node")
	require.Empty(t, m.blocksTopic.ListPeers())

	p2, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate a second p2p priv key")
	m2, err := NewMalleable(l, cid, nil, p2, OnUnsafeL2Payload)
	require.NoError(t, err, "failed to create new malleable node")
	require.Empty(t, m2.blocksTopic.ListPeers())

	err = m.h.Connect(context.Background(), peer.AddrInfo{ID: m2.ID(), Addrs: m2.Addrs()})
	require.NoError(t, err, "failed to connect the hosts")
	require.Equal(t, m.h.Network().Connectedness(m2.ID()), network.Connected)
	require.Equal(t, m2.h.Network().Connectedness(m.ID()), network.Connected)
	require.Equal(t, m.h.Peerstore().Peers().Len(), 2)
	require.Equal(t, m2.h.Peerstore().Peers().Len(), 2)
}

func OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
	return nil
}

// TestMalleable_PublishPayload tests publishing an [eth.ExecutionPayload]
// through the [Malleable] node to the [pubsub.Topic].
func TestMalleable_PublishPayload(t *testing.T) {
	cid := big.NewInt(420)
	l := testlog.Logger(t, log.LvlInfo)

	// Construct the first malleable node
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	m, err := NewMalleable(l, cid, nil, p, OnUnsafeL2Payload)
	require.NoError(t, err, "failed to create new malleable node")

	// Construct the second malleable node
	receivedBlockNumber := eth.Uint64Quantity(0)
	m2BlocksCallback := func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
		receivedBlockNumber = payload.BlockNumber
		require.Equal(t, payload.BlockNumber, eth.Uint64Quantity(1))
		return nil
	}
	p2, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate a second p2p priv key")
	m2, err := NewMalleable(l, cid, nil, p2, m2BlocksCallback)
	require.NoError(t, err, "failed to create new malleable node")

	// Connect the nodes
	err = m.h.Connect(context.Background(), peer.AddrInfo{ID: m2.ID(), Addrs: m2.Addrs()})
	require.NoError(t, err, "failed to connect the hosts")

	// Construct a payload
	testKey := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	priv, err := ethCrypto.HexToECDSA(testKey)
	require.NoError(t, err, "failed to build private key from hex string")
	signer := &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(priv)}
	payload := &eth.ExecutionPayload{
		BlockNumber: eth.Uint64Quantity(1),
	}

	// Publish the payload and give it time to be received
	require.Zero(t, receivedBlockNumber)
	m.PublishL2Payload(context.Background(), payload, signer)
	time.Sleep(1 * time.Second)
	require.Equal(t, receivedBlockNumber, eth.Uint64Quantity(1))
}
