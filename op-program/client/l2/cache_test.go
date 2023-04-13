package l2

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Should be an Oracle implementation
var _ Oracle = (*CachingOracle)(nil)

func TestBlockByHash(t *testing.T) {
	stub, _ := newStubOracle(t)
	oracle := NewCachingOracle(stub)

	rng := rand.New(rand.NewSource(1))
	block, _ := testutils.RandomBlock(rng, 1)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block
	actual := oracle.BlockByHash(block.Hash())
	require.Equal(t, block, actual)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	actual = oracle.BlockByHash(block.Hash())
	require.Equal(t, block, actual)
}

func TestNodeByHash(t *testing.T) {
	stub, stateStub := newStubOracle(t)
	oracle := NewCachingOracle(stub)

	node := []byte{12, 3, 4}
	hash := common.Hash{0xaa}

	// Initial call retrieves from the stub
	stateStub.data[hash] = node
	actual := oracle.NodeByHash(hash)
	require.Equal(t, node, actual)

	// Later calls should retrieve from cache
	delete(stateStub.data, hash)
	actual = oracle.NodeByHash(hash)
	require.Equal(t, node, actual)
}

func TestCodeByHash(t *testing.T) {
	stub, stateStub := newStubOracle(t)
	oracle := NewCachingOracle(stub)

	node := []byte{12, 3, 4}
	hash := common.Hash{0xaa}

	// Initial call retrieves from the stub
	stateStub.code[hash] = node
	actual := oracle.CodeByHash(hash)
	require.Equal(t, node, actual)

	// Later calls should retrieve from cache
	delete(stateStub.code, hash)
	actual = oracle.CodeByHash(hash)
	require.Equal(t, node, actual)
}
