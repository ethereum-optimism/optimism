package l1

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type stubOracle struct {
	t *testing.T

	// blocks maps block hash to eth.BlockInfo
	blocks map[common.Hash]*types.Header

	// txs maps block hash to transactions
	txs map[common.Hash]types.Transactions

	// rcpts maps Block hash to receipts
	rcpts map[common.Hash]types.Receipts
}

func newStubOracle(t *testing.T) *stubOracle {
	return &stubOracle{
		t:      t,
		blocks: make(map[common.Hash]*types.Header),
		txs:    make(map[common.Hash]types.Transactions),
		rcpts:  make(map[common.Hash]types.Receipts),
	}
}
func (o stubOracle) HeaderByBlockHash(blockHash common.Hash) *types.Header {
	info, ok := o.blocks[blockHash]
	if !ok {
		o.t.Fatalf("unknown block %s", blockHash)
	}
	return info
}

func (o stubOracle) TransactionsByBlockHash(blockHash common.Hash) (*types.Header, types.Transactions) {
	txs, ok := o.txs[blockHash]
	if !ok {
		o.t.Fatalf("unknown txs %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), txs
}

func (o stubOracle) ReceiptsByBlockHash(blockHash common.Hash) (*types.Header, types.Receipts) {
	rcpts, ok := o.rcpts[blockHash]
	if !ok {
		o.t.Fatalf("unknown rcpts %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), rcpts
}
