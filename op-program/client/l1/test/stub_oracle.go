package test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
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

	// L2Outputs maps L2 output roots to L2 outputs
	L2Outputs map[common.Hash]eth.Output
}

func NewStubOracle(t *testing.T) *StubOracle {
	return &StubOracle{
		t:         t,
		Blocks:    make(map[common.Hash]eth.BlockInfo),
		Txs:       make(map[common.Hash]types.Transactions),
		Rcpts:     make(map[common.Hash]types.Receipts),
		L2Outputs: make(map[common.Hash]eth.Output),
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

func (o StubOracle) L2OutputByRoot(l2OutputRoot common.Hash) eth.Output {
	output, ok := o.L2Outputs[l2OutputRoot]
	if !ok {
		o.t.Fatalf("unknown output %s", l2OutputRoot)
	}
	return output
}
