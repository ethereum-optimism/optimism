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
	header := testutils.RandomHeader(rng)

	// Initial call retrieves from the stub
	stub.blocks[header.Hash()] = header
	result := oracle.HeaderByBlockHash(header.Hash())
	require.Equal(t, header, result)

	// Later calls should retrieve from cache
	delete(stub.blocks, header.Hash())
	result = oracle.HeaderByBlockHash(header.Hash())
	require.Equal(t, header, result)
}

func TestCachingOracle_TransactionsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := newStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, _ := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block.Header()
	stub.txs[block.Hash()] = block.Transactions()
	actualHeader, actualTxs := oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, block.Header(), actualHeader)
	require.Equal(t, block.Transactions(), actualTxs)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	delete(stub.txs, block.Hash())
	actualHeader, actualTxs = oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, block.Header(), actualHeader)
	require.Equal(t, block.Transactions(), actualTxs)
}

func TestCachingOracle_ReceiptsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := newStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, rcpts := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.blocks[block.Hash()] = block.Header()
	stub.rcpts[block.Hash()] = rcpts
	actualHeader, actualRcpts := oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, block.Header(), actualHeader)
	require.EqualValues(t, rcpts, actualRcpts)

	// Later calls should retrieve from cache
	delete(stub.blocks, block.Hash())
	delete(stub.rcpts, block.Hash())
	actualHeader, actualRcpts = oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, block.Header(), actualHeader)
	require.EqualValues(t, rcpts, actualRcpts)
}
