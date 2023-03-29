package malleable

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	log "github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
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

	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	m, err := NewMalleable(l, cid, nil, p, OnUnsafeL2Payload)
	require.NoError(t, err, "failed to create new malleable node")

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

	err = m.h.Connect(context.Background(), peer.AddrInfo{ID: m2.ID(), Addrs: m2.Addrs()})
	require.NoError(t, err, "failed to connect the hosts")

	testKey := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	priv, err := ethCrypto.HexToECDSA(testKey)
	require.NoError(t, err, "failed to build private key from hex string")
	signer := &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(priv)}
	randomTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(420),
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Data:      []byte("hello world"),
	})
	// executionPayloadFixedPart := 32 + 20 + 32 + 32 + 256 + 32 + 8 + 8 + 8 + 8 + 4 + 32 + 32 + 4
	payload := &eth.ExecutionPayload{
		ParentHash:    common.HexToHash("0x5c698f13940a2153440c6d19660878bc90219d9298fdcf37365aa8d88d40fc42"),
		FeeRecipient:  common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(1),
		GasLimit:      eth.Uint64Quantity(1337),
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(1),
		ExtraData:     []byte("hello world"), // make([]byte, 100000), // math.MaxUint32-executionPayloadFixedPart),
		BaseFeePerGas: *uint256.NewInt(7),
		BlockHash:     common.HexToHash("0x5c698f13940a2153440c6d19660878bc90219d9298fdcf37365aa8d88d40fc42"),
		Transactions: []eth.Data{
			randomTx.Data(),
		},
	}

	m.PublishL2Payload(context.Background(), payload, signer)

	// Wait for the payload to be gossiped
	time.Sleep(1 * time.Second)

	// This should have been received
	require.Equal(t, receivedBlockNumber, eth.Uint64Quantity(1))

	// Check that the payload was received by the other node at the other end of the pubsub topic.
	require.Equal(t, m.h.Network().Connectedness(m2.ID()), network.Connected)
	require.Equal(t, m2.h.Network().Connectedness(m.ID()), network.Connected)
	require.Equal(t, m.h.Peerstore().Peers().Len(), 2)
	require.Equal(t, m2.h.Peerstore().Peers().Len(), 2)
}
