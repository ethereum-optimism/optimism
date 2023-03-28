package malleable

import (
	"crypto/rand"
	"math/big"
	"testing"

	log "github.com/ethereum/go-ethereum/log"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	require "github.com/stretchr/testify/require"

	testlog "github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestMalleable_NewMalleable tests constructing a new [Malleable] node.
func TestMalleable_NewMalleable(t *testing.T) {
	// Create a new private key.
	p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "failed to generate new p2p priv key")

	// Create a new malleable node.
	log := testlog.Logger(t, log.LvlInfo)
	l2ChainID := big.NewInt(420)
	m, err := NewMalleable(
		log,
		l2ChainID,
		nil,
		p,
	)
	require.NoError(t, err, "failed to create new malleable node")

	// The list of peers should be empty to start
	require.Empty(t, m.blocksTopic.ListPeers())

	// TODO: test the node
}
