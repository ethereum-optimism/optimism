package l2

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type CachingOracle struct {
	oracle Oracle
	blocks map[common.Hash]*types.Block
	nodes  map[common.Hash][]byte
	codes  map[common.Hash][]byte
}

func NewCachingOracle(oracle Oracle) *CachingOracle {
	return &CachingOracle{
		oracle: oracle,
		blocks: make(map[common.Hash]*types.Block),
		nodes:  make(map[common.Hash][]byte),
		codes:  make(map[common.Hash][]byte),
	}
}

func (o CachingOracle) NodeByHash(nodeHash common.Hash) []byte {
	node, ok := o.nodes[nodeHash]
	if ok {
		return node
	}
	node = o.oracle.NodeByHash(nodeHash)
	o.nodes[nodeHash] = node
	return node
}

func (o CachingOracle) CodeByHash(codeHash common.Hash) []byte {
	code, ok := o.codes[codeHash]
	if ok {
		return code
	}
	code = o.oracle.CodeByHash(codeHash)
	o.codes[codeHash] = code
	return code
}

func (o CachingOracle) BlockByHash(blockHash common.Hash) *types.Block {
	block, ok := o.blocks[blockHash]
	if ok {
		return block
	}
	block = o.oracle.BlockByHash(blockHash)
	o.blocks[blockHash] = block
	return block
}
