package interop

import (
	"fmt"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	l2Types "github.com/ethereum-optimism/optimism/op-program/client/l2/types"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
)

// ConsolidateOracle extends another l2.Oracle with consolidated state data.
// The consolidated state data includes data from deposits-only replacement blocks.
type ConsolidateOracle struct {
	o  l2.Oracle
	db l2.KeyValueStore
	ts *interopTypes.TransitionState
}

var _ l2.Oracle = &ConsolidateOracle{}

func NewConsolidateOracle(oracle l2.Oracle, transitionState *interopTypes.TransitionState) *ConsolidateOracle {
	return &ConsolidateOracle{
		o:  oracle,
		db: memorydb.New(),
		ts: transitionState,
	}
}

func (o *ConsolidateOracle) BlockByHash(blockHash common.Hash, chainID eth.ChainID) *types.Block {
	block := o.consolidatedBlockByHash(blockHash)
	if block != nil {
		return block
	}
	return o.o.BlockByHash(blockHash, chainID)
}

func (o *ConsolidateOracle) OutputByRoot(root common.Hash, chainID eth.ChainID) eth.Output {
	key := preimage.Keccak256Key(root).PreimageKey()
	b, err := o.db.Get(key[:])
	if err == nil {
		output, err := eth.UnmarshalOutput(b)
		if err != nil {
			panic(fmt.Errorf("invalid output %s: %w", root, err))
		}
		return output
	}
	return o.o.OutputByRoot(root, chainID)
}

func (o *ConsolidateOracle) BlockDataByHash(agreedBlockHash, blockHash common.Hash, chainID eth.ChainID) *types.Block {
	block := o.consolidatedBlockByHash(blockHash)
	if block != nil {
		return block
	}
	return o.o.BlockDataByHash(agreedBlockHash, blockHash, chainID)
}

func (o *ConsolidateOracle) ReceiptsByBlockHash(blockHash common.Hash, chainID eth.ChainID) (*types.Block, types.Receipts) {
	block := o.consolidatedBlockByHash(blockHash)
	if block != nil {
		opaqueReceipts := mpt.ReadTrie(block.ReceiptHash(), func(key common.Hash) []byte {
			k := preimage.Keccak256Key(key).PreimageKey()
			b, err := o.db.Get(k[:])
			if err != nil {
				panic(fmt.Errorf("missing receipt trie node %s: %w", key, err))
			}
			return b
		})
		txHashes := make([]common.Hash, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txHashes[i] = tx.Hash()
		}
		receipts, err := eth.DecodeRawReceipts(eth.ToBlockID(block), opaqueReceipts, txHashes)
		if err != nil {
			panic(fmt.Errorf("failed to decode receipts for block %v: %w", block.Hash(), err))
		}
		return block, receipts
	}

	return o.o.ReceiptsByBlockHash(blockHash, chainID)
}

func (o *ConsolidateOracle) NodeByHash(nodeHash common.Hash, chainID eth.ChainID) []byte {
	node, err := o.db.Get(nodeHash[:])
	if err == nil {
		return node
	}
	return o.o.NodeByHash(nodeHash, chainID)
}

func (o *ConsolidateOracle) CodeByHash(codeHash common.Hash, chainID eth.ChainID) []byte {
	code, err := o.db.Get(codeHash[:])
	if err == nil {
		return code
	}
	return o.o.CodeByHash(codeHash, chainID)
}

func (o *ConsolidateOracle) Hinter() l2Types.OracleHinter {
	return o.o.Hinter()
}

func (o *ConsolidateOracle) TransitionStateByRoot(root common.Hash) *interopTypes.TransitionState {
	return o.ts
}

func (o *ConsolidateOracle) headerByBlockHash(blockHash common.Hash) *types.Header {
	blockHashKey := preimage.Keccak256Key(blockHash).PreimageKey()
	headerRlp, err := o.db.Get(blockHashKey[:])
	if err != nil {
		return nil
	}
	var header types.Header
	if err := rlp.DecodeBytes(headerRlp, &header); err != nil {
		panic(fmt.Errorf("invalid block header %s: %w", blockHash, err))
	}
	return &header
}

func (o *ConsolidateOracle) loadTransactions(txHash common.Hash) []*types.Transaction {
	opaqueTxs := mpt.ReadTrie(txHash, func(key common.Hash) []byte {
		k := preimage.Keccak256Key(key).PreimageKey()
		b, err := o.db.Get(k[:])
		if err != nil {
			panic(fmt.Errorf("missing tx trie node %s", key))
		}
		return b
	})
	txs, err := eth.DecodeTransactions(opaqueTxs)
	if err != nil {
		panic(fmt.Errorf("failed to decode list of txs: %w", err))
	}
	return txs
}

func (o *ConsolidateOracle) consolidatedBlockByHash(blockHash common.Hash) *types.Block {
	header := o.headerByBlockHash(blockHash)
	if header == nil {
		return nil
	}
	txs := o.loadTransactions(header.TxHash)
	return types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
}

func (o *ConsolidateOracle) KeyValueStore() l2.KeyValueStore {
	return o.db
}
