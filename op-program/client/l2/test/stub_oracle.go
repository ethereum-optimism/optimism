package test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Same as l2.StateOracle but need to use our own copy to avoid dependency loops
type stateOracle interface {
	NodeByHash(nodeHash common.Hash) []byte
	CodeByHash(codeHash common.Hash) []byte
}

type StubBlockOracle struct {
	t      *testing.T
	Blocks map[common.Hash]*types.Block
	stateOracle
}

func NewStubOracle(t *testing.T) (*StubBlockOracle, *StubStateOracle) {
	stateOracle := NewStubStateOracle(t)
	blockOracle := StubBlockOracle{
		t:           t,
		Blocks:      make(map[common.Hash]*types.Block),
		stateOracle: stateOracle,
	}
	return &blockOracle, stateOracle
}

func NewStubOracleWithBlocks(t *testing.T, chain []*types.Block, db ethdb.Database) *StubBlockOracle {
	blocks := make(map[common.Hash]*types.Block, len(chain))
	for _, block := range chain {
		blocks[block.Hash()] = block
	}
	return &StubBlockOracle{
		Blocks:      blocks,
		stateOracle: &KvStateOracle{t: t, Source: db},
	}
}

func (o StubBlockOracle) BlockByHash(blockHash common.Hash) *types.Block {
	block, ok := o.Blocks[blockHash]
	if !ok {
		o.t.Fatalf("requested unknown block %s", blockHash)
	}
	return block
}

// KvStateOracle loads data from a source ethdb.KeyValueStore
type KvStateOracle struct {
	t      *testing.T
	Source ethdb.KeyValueStore
}

func NewKvStateOracle(t *testing.T, db ethdb.KeyValueStore) *KvStateOracle {
	return &KvStateOracle{
		t:      t,
		Source: db,
	}
}

func (o *KvStateOracle) NodeByHash(nodeHash common.Hash) []byte {
	val, err := o.Source.Get(nodeHash.Bytes())
	if err != nil {
		o.t.Fatalf("error retrieving node %v: %v", nodeHash, err)
	}
	return val
}

func (o *KvStateOracle) CodeByHash(hash common.Hash) []byte {
	return rawdb.ReadCode(o.Source, hash)
}

func NewStubStateOracle(t *testing.T) *StubStateOracle {
	return &StubStateOracle{
		t:    t,
		Data: make(map[common.Hash][]byte),
		Code: make(map[common.Hash][]byte),
	}
}

// StubStateOracle is a StateOracle implementation that reads from simple maps
type StubStateOracle struct {
	t    *testing.T
	Data map[common.Hash][]byte
	Code map[common.Hash][]byte
}

func (o *StubStateOracle) NodeByHash(nodeHash common.Hash) []byte {
	data, ok := o.Data[nodeHash]
	if !ok {
		o.t.Fatalf("no value for node %v", nodeHash)
	}
	return data
}

func (o *StubStateOracle) CodeByHash(hash common.Hash) []byte {
	data, ok := o.Code[hash]
	if !ok {
		o.t.Fatalf("no value for code %v", hash)
	}
	return data
}
