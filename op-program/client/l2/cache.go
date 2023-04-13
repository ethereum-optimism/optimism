package l2

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hashicorp/golang-lru/v2/simplelru"
)

const blockCacheSize = 2_000
const nodeCacheSize = 100_000
const codeCacheSize = 10_000

type CachingOracle struct {
	oracle Oracle
	blocks *simplelru.LRU[common.Hash, *types.Block]
	nodes  *simplelru.LRU[common.Hash, []byte]
	codes  *simplelru.LRU[common.Hash, []byte]
}

func NewCachingOracle(oracle Oracle) *CachingOracle {
	blockLRU, _ := simplelru.NewLRU[common.Hash, *types.Block](blockCacheSize, nil)
	nodeLRU, _ := simplelru.NewLRU[common.Hash, []byte](nodeCacheSize, nil)
	codeLRU, _ := simplelru.NewLRU[common.Hash, []byte](codeCacheSize, nil)
	return &CachingOracle{
		oracle: oracle,
		blocks: blockLRU,
		nodes:  nodeLRU,
		codes:  codeLRU,
	}
}

func (o *CachingOracle) NodeByHash(nodeHash common.Hash) []byte {
	node, ok := o.nodes.Get(nodeHash)
	if ok {
		return node
	}
	node = o.oracle.NodeByHash(nodeHash)
	o.nodes.Add(nodeHash, node)
	return node
}

func (o *CachingOracle) CodeByHash(codeHash common.Hash) []byte {
	code, ok := o.codes.Get(codeHash)
	if ok {
		return code
	}
	code = o.oracle.CodeByHash(codeHash)
	o.codes.Add(codeHash, code)
	return code
}

func (o *CachingOracle) BlockByHash(blockHash common.Hash) *types.Block {
	block, ok := o.blocks.Get(blockHash)
	if ok {
		return block
	}
	block = o.oracle.BlockByHash(blockHash)
	o.blocks.Add(blockHash, block)
	return block
}
