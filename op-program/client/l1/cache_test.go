package l1

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-program/client/l1/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// Should implement Oracle
var _ Oracle = (*CachingOracle)(nil)

func TestCachingOracle_HeaderByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := test.NewStubOracle(t)
	oracle := NewCachingOracle(stub)
	block := testutils.RandomBlockInfo(rng)

	// Initial call retrieves from the stub
	stub.Blocks[block.Hash()] = block
	result := oracle.HeaderByBlockHash(block.Hash())
	require.Equal(t, block, result)

	// Later calls should retrieve from cache
	delete(stub.Blocks, block.Hash())
	result = oracle.HeaderByBlockHash(block.Hash())
	require.Equal(t, block, result)
}

func TestCachingOracle_TransactionsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := test.NewStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, _ := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.Blocks[block.Hash()] = eth.BlockToInfo(block)
	stub.Txs[block.Hash()] = block.Transactions()
	actualBlock, actualTxs := oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, eth.BlockToInfo(block), actualBlock)
	require.Equal(t, block.Transactions(), actualTxs)

	// Later calls should retrieve from cache
	delete(stub.Blocks, block.Hash())
	delete(stub.Txs, block.Hash())
	actualBlock, actualTxs = oracle.TransactionsByBlockHash(block.Hash())
	require.Equal(t, eth.BlockToInfo(block), actualBlock)
	require.Equal(t, block.Transactions(), actualTxs)
}

func TestCachingOracle_ReceiptsByBlockHash(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := test.NewStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, rcpts := testutils.RandomBlock(rng, 3)

	// Initial call retrieves from the stub
	stub.Blocks[block.Hash()] = eth.BlockToInfo(block)
	stub.Rcpts[block.Hash()] = rcpts
	actualBlock, actualRcpts := oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, eth.BlockToInfo(block), actualBlock)
	require.EqualValues(t, rcpts, actualRcpts)

	// Later calls should retrieve from cache
	delete(stub.Blocks, block.Hash())
	delete(stub.Rcpts, block.Hash())
	actualBlock, actualRcpts = oracle.ReceiptsByBlockHash(block.Hash())
	require.Equal(t, eth.BlockToInfo(block), actualBlock)
	require.EqualValues(t, rcpts, actualRcpts)
}

func TestCachingOracle_L2OutputByRoot(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stub := test.NewStubOracle(t)
	oracle := NewCachingOracle(stub)
	block, _ := testutils.RandomBlock(rng, 3)

	var storageRoot [32]byte
	rng.Read(storageRoot[:])
	l2Output := &eth.OutputV0{
		StateRoot:                eth.Bytes32(block.Root()),
		MessagePasserStorageRoot: storageRoot,
		BlockHash:                block.Hash(),
	}
	l2OutputRoot := common.Hash(eth.OutputRoot(l2Output))

	// Initial call retrieves from the stub
	stub.L2Outputs[l2OutputRoot] = l2Output
	result := oracle.L2OutputByRoot(l2OutputRoot)
	require.Equal(t, l2Output.Marshal(), result)

	// Later calls should retrieve from cache
	delete(stub.L2Outputs, l2OutputRoot)
	result = oracle.L2OutputByRoot(l2OutputRoot)
	require.Equal(t, l2Output.Marshal(), result)
}
