package test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type StubOracle struct {
	t *testing.T

	// Blocks maps block hash to eth.BlockInfo
	Blocks map[common.Hash]eth.BlockInfo

	// Txs maps block hash to transactions
	Txs map[common.Hash]types.Transactions

	// Rcpts maps Block hash to receipts
	Rcpts map[common.Hash]types.Receipts

	// Blobs maps indexed blob hash to l1 block ref to blob
	Blobs map[eth.IndexedBlobHash]map[eth.L1BlockRef]*eth.Blob
}

func NewStubOracle(t *testing.T) *StubOracle {
	return &StubOracle{
		t:      t,
		Blocks: make(map[common.Hash]eth.BlockInfo),
		Txs:    make(map[common.Hash]types.Transactions),
		Rcpts:  make(map[common.Hash]types.Receipts),
		Blobs:  make(map[eth.IndexedBlobHash]map[eth.L1BlockRef]*eth.Blob),
	}
}

func (o StubOracle) HeaderByBlockHash(blockHash common.Hash) eth.BlockInfo {
	info, ok := o.Blocks[blockHash]
	if !ok {
		o.t.Fatalf("unknown block %s", blockHash)
	}
	return info
}

func (o StubOracle) TransactionsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Transactions) {
	txs, ok := o.Txs[blockHash]
	if !ok {
		o.t.Fatalf("unknown txs %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), txs
}

func (o StubOracle) ReceiptsByBlockHash(blockHash common.Hash) (eth.BlockInfo, types.Receipts) {
	rcpts, ok := o.Rcpts[blockHash]
	if !ok {
		o.t.Fatalf("unknown rcpts %s", blockHash)
	}
	return o.HeaderByBlockHash(blockHash), rcpts
}

func (o StubOracle) GetBlob(ref eth.L1BlockRef, blobHash eth.IndexedBlobHash) *eth.Blob {
	o.t.Fatalf("unimplemented")
	return nil
}
