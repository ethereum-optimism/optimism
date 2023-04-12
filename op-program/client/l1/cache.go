package l1

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CachingOracle is an implementation of Oracle that delegates to another implementation, adding caching of all results
type CachingOracle struct {
	oracle Oracle
	blocks map[common.Hash]eth.BlockInfo
	txs    map[common.Hash]types.Transactions
	rcpts  map[common.Hash]types.Receipts
}

func NewCachingOracle(oracle Oracle) *CachingOracle {
	return &CachingOracle{
		oracle: oracle,
		blocks: make(map[common.Hash]eth.BlockInfo),
		txs:    make(map[common.Hash]types.Transactions),
		rcpts:  make(map[common.Hash]types.Receipts),
	}
}

func (o CachingOracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	block, ok := o.blocks[blockHash]
	if ok {
		return block
	}
	block = o.oracle.HeaderByBlockHash(blockHash)
	o.blocks[blockHash] = block
	return block
}

func (o CachingOracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	txs, ok := o.txs[blockHash]
	if ok {
		return o.HeaderByBlockHash(blockHash), txs
	}
	block, txs := o.oracle.TransactionsByBlockHash(blockHash)
	o.blocks[blockHash] = block
	o.txs[blockHash] = txs
	return block, txs
}

func (o CachingOracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	rcpts, ok := o.rcpts[blockHash]
	if ok {
		return o.HeaderByBlockHash(blockHash), rcpts
	}
	block, rcpts := o.oracle.ReceiptsByBlockHash(blockHash)
	o.blocks[blockHash] = block
	o.rcpts[blockHash] = rcpts
	return block, rcpts
}
