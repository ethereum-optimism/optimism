package l1

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type stubOracle struct {
	t *testing.T

	// blocks maps block hash to eth.BlockInfo
	blocks map[common.Hash]eth.BlockInfo

	// txs maps block hash to transactions
	txs map[common.Hash]types.Transactions

	// rcpts maps Block hash to receipts
	rcpts map[common.Hash]types.Receipts
}

func newStubOracle(t *testing.T) *stubOracle {
	return &stubOracle{
		t:      t,
		blocks: make(map[common.Hash]eth.BlockInfo),
		txs:    make(map[common.Hash]types.Transactions),
		rcpts:  make(map[common.Hash]types.Receipts),
	}
}
func (o stubOracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	info, ok := o.blocks[blockHash]
	if !ok {
		o.t.Fatalf("unknown block %s", blockHash)
	}
	return info
}

func (o stubOracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	txs, ok := o.txs[blockHash]
	if !ok {
		o.t.Fatalf("unknown txs %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), txs
}

func (o stubOracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	rcpts, ok := o.rcpts[blockHash]
	if !ok {
		o.t.Fatalf("unknown rcpts %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), rcpts
}
