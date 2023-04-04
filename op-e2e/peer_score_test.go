package op_e2e

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	log "github.com/ethereum/go-ethereum/log"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	require "github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	malleable "github.com/ethereum-optimism/optimism/op-e2e/malleable"
	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestMalleable_PeerScoreUpdated tests publishing an [eth.ExecutionPayload]
// through the [Malleable] node to the [pubsub.Topic].
func TestMalleable_PeerScoreUpdated(t *testing.T) {
	// Setup the default system first
	cfg := DefaultSystemConfig(t)
	cfg.P2PTopology = map[string][]string{
		"verifier": {"sequencer"},
	}
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	honestNodeId := sys.RollupNodes["verifier"].P2P().Host().ID()
	honestNodeAddrs := sys.RollupNodes["verifier"].P2P().Host().Addrs()

	// Grab the chain id from the system config
	cid := cfg.DeployConfig.L2ChainID
	l := testlog.Logger(t, log.LvlInfo)
	onUnsafeL2Payload := func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
		return nil
	}

	// Construct a new malleable node.
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")
	m, err := malleable.NewMalleable(l, big.NewInt(int64(cid)), nil, p, onUnsafeL2Payload)
	require.NoError(t, err, "failed to create new malleable node")
	malleableNodeId := m.ID()

	// Connect the Malleable node to the honest rollup node
	err = m.Connect(context.Background(), peer.AddrInfo{ID: honestNodeId, Addrs: honestNodeAddrs})
	require.NoError(t, err, "failed to connect the hosts")
	require.Equal(t, m.Network().Connectedness(m.ID()), network.Connected)

	// Construct an execution payload
	testKey := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	priv, err := ethCrypto.HexToECDSA(testKey)
	require.NoError(t, err, "failed to build private key from hex string")
	signer := &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(priv)}
	randomTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(int64(cid)),
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Data:      []byte("hello world"),
	})
	payload := &eth.ExecutionPayload{
		ParentHash:   common.HexToHash("0x5c698f13940a2153440c6d19660878bc90219d9298fdcf37365aa8d88d40fc42"),
		FeeRecipient: common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
		StateRoot:    eth.Bytes32{},
		ReceiptsRoot: eth.Bytes32{},
		LogsBloom:    eth.Bytes256{},
		PrevRandao:   eth.Bytes32{},
		BlockNumber:  eth.Uint64Quantity(1),
		GasLimit:     eth.Uint64Quantity(1337),
		GasUsed:      0,
		Timestamp:    eth.Uint64Quantity(1),
		ExtraData:    []byte("hello world"),
		// BaseFeePerGas: *uint256.NewInt(7),
		BlockHash: common.HexToHash("0x5c698f13940a2153440c6d19660878bc90219d9298fdcf37365aa8d88d40fc42"),
		Transactions: []eth.Data{
			randomTx.Data(),
		},
	}

	for {
		// Publish the payload
		m.PublishL2Payload(context.Background(), payload, signer)
		time.Sleep(1 * time.Second)
		// Get the peer score of the honest node
		connected := slices.Contains(sys.RollupNodes["verifier"].P2P().Host().Peerstore().Peers(), malleableNodeId)
		fmt.Printf("Is malleable node still connected to honest node: %v\n", connected)
	}
}
