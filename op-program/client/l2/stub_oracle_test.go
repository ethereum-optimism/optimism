package l2

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

type stubBlockOracle struct {
	t      *testing.T
	blocks map[common.Hash]*types.Block
	StateOracle
}

func newStubOracle(t *testing.T) (*stubBlockOracle, *stubStateOracle) {
	stateOracle := newStubStateOracle(t)
	blockOracle := stubBlockOracle{
		t:           t,
		blocks:      make(map[common.Hash]*types.Block),
		StateOracle: stateOracle,
	}
	return &blockOracle, stateOracle
}

func newStubOracleWithBlocks(t *testing.T, chain []*types.Block, db ethdb.Database) *stubBlockOracle {
	blocks := make(map[common.Hash]*types.Block, len(chain))
	for _, block := range chain {
		blocks[block.Hash()] = block
	}
	return &stubBlockOracle{
		blocks:      blocks,
		StateOracle: &kvStateOracle{t: t, source: db},
	}
}

func (o stubBlockOracle) BlockByHash(blockHash common.Hash) *types.Block {
	block, ok := o.blocks[blockHash]
	if !ok {
		o.t.Fatalf("requested unknown block %s", blockHash)
	}
	return block
}

// kvStateOracle loads data from a source ethdb.KeyValueStore
type kvStateOracle struct {
	t      *testing.T
	source ethdb.KeyValueStore
}

func (o *kvStateOracle) NodeByHash(nodeHash common.Hash) []byte {
	val, err := o.source.Get(nodeHash.Bytes())
	if err != nil {
		o.t.Fatalf("error retrieving node %v: %v", nodeHash, err)
	}
	return val
}

func (o *kvStateOracle) CodeByHash(hash common.Hash) []byte {
	return rawdb.ReadCode(o.source, hash)
}

func newStubStateOracle(t *testing.T) *stubStateOracle {
	return &stubStateOracle{
		t:    t,
		data: make(map[common.Hash][]byte),
		code: make(map[common.Hash][]byte),
	}
}

// Stub StateOracle implementation that reads from simple maps
type stubStateOracle struct {
	t    *testing.T
	data map[common.Hash][]byte
	code map[common.Hash][]byte
}

func (o *stubStateOracle) NodeByHash(nodeHash common.Hash) []byte {
	data, ok := o.data[nodeHash]
	if !ok {
		o.t.Fatalf("no value for node %v", nodeHash)
	}
	return data
}

func (o *stubStateOracle) CodeByHash(hash common.Hash) []byte {
	data, ok := o.code[hash]
	if !ok {
		o.t.Fatalf("no value for code %v", hash)
	}
	return data
}
