package l1

import (
	"github.com/hashicorp/golang-lru/v2/simplelru"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Cache size is quite high as retrieving data from the pre-image oracle can be quite expensive
const cacheSize = 2000

// CachingOracle is an implementation of Oracle that delegates to another implementation, adding caching of all results
type CachingOracle struct {
	oracle Oracle
	blocks *simplelru.LRU[common.Hash, eth.BlockInfo]
	txs    *simplelru.LRU[common.Hash, types.Transactions]
	rcpts  *simplelru.LRU[common.Hash, types.Receipts]
}

func NewCachingOracle(oracle Oracle) *CachingOracle {
	blockLRU, _ := simplelru.NewLRU[common.Hash, eth.BlockInfo](cacheSize, nil)
	txsLRU, _ := simplelru.NewLRU[common.Hash, types.Transactions](cacheSize, nil)
	rcptsLRU, _ := simplelru.NewLRU[common.Hash, types.Receipts](cacheSize, nil)
	return &CachingOracle{
		oracle: oracle,
		blocks: blockLRU,
		txs:    txsLRU,
		rcpts:  rcptsLRU,
	}
}

func (o *CachingOracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	block, ok := o.blocks.Get(blockHash)
	if ok {
		return block
	}
	block = o.oracle.HeaderByBlockHash(blockHash)
	o.blocks.Add(blockHash, block)
	return block
}

func (o *CachingOracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	txs, ok := o.txs.Get(blockHash)
	if ok {
		return o.HeaderByBlockHash(blockHash), txs
	}
	block, txs := o.oracle.TransactionsByBlockHash(blockHash)
	o.blocks.Add(blockHash, block)
	o.txs.Add(blockHash, txs)
	return block, txs
}

func (o *CachingOracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	rcpts, ok := o.rcpts.Get(blockHash)
	if ok {
		return o.HeaderByBlockHash(blockHash), rcpts
	}
	block, rcpts := o.oracle.ReceiptsByBlockHash(blockHash)
	o.blocks.Add(blockHash, block)
	o.rcpts.Add(blockHash, rcpts)
	return block, rcpts
}
