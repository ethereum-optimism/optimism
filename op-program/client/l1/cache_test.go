package l1

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/stretchr/testify/require"
)

// Should implement Oracle
var _ Oracle = (*CachingOracle)(nil)

func TestCachingOracle_HeaderByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := newStubOracle(t)
	oracle := NewCachingOracle(stub)
	block := testutils.RandomBlockInfo(rng)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block
	result := oracle.HeaderByBlockHash(block.Hash())
	require.Equal(t, block, result)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	result = oracle.HeaderByBlockHash(block.Hash())
	require.Equal(t, block, result)
}

func TestCachingOracle_TransactionsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := newStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, _ := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block
	stub.txs[block.Hash()] = block.Transactions()
	actualBlock, actualTxs := oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, block, actualBlock)
	require.Equal(t, block.Transactions(), actualTxs)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	delete(stub.txs, block.Hash())
	actualBlock, actualTxs = oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, block, actualBlock)
	require.Equal(t, block.Transactions(), actualTxs)
}

func TestCachingOracle_ReceiptsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := newStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, rcpts := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block
	stub.rcpts[block.Hash()] = rcpts
	actualBlock, actualRcpts := oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, block, actualBlock)
	require.EqualValues(t, rcpts, actualRcpts)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	delete(stub.rcpts, block.Hash())
	actualBlock, actualRcpts = oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, block, actualBlock)
	require.EqualValues(t, rcpts, actualRcpts)
}
