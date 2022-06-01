package l1

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/core/types"
)

// fetchReceipts fetches the receipts of the transactions using RPC batching, verifies if the receipts are complete and correct, and then returns results
func fetchReceipts(ctx context.Context, block eth.BlockID, receiptHash common.Hash, txs types.Transactions, getBatch batchCallContextFn) (types.Receipts, error) {
	if len(txs) == 0 {
		if receiptHash != types.EmptyRootHash {
			return nil, fmt.Errorf("no transactions, but got non-empty receipt trie root: %s", receiptHash)
		}
		return nil, nil
	}

	receipts := make([]*types.Receipt, len(txs))
	receiptRequests := make([]rpc.BatchElem, len(txs))
	for i := 0; i < len(txs); i++ {
		receipts[i] = new(types.Receipt)
		receiptRequests[i] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txs[i].Hash()},
			Result: &receipts[i], // receipt may become nil, double pointer is intentional
		}
	}
	if err := getBatch(ctx, receiptRequests); err != nil {
		return nil, fmt.Errorf("failed to fetch batch of receipts: %v", err)
	}

	// We don't trust the RPC to provide consistent cached receipt info that we use for critical rollup derivation work.
	// Let's check everything quickly.
	logIndex := uint(0)
	for i, r := range receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return nil, fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
		}
		if r.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("receipt %d has unexpected tx index %d", i, r.TransactionIndex)
		}
		if r.BlockNumber.Uint64() != block.Number {
			return nil, fmt.Errorf("receipt %d has unexpected block number %d, expected %d", i, r.BlockNumber, block.Number)
		}
		if r.BlockHash != block.Hash {
			return nil, fmt.Errorf("receipt %d has unexpected block hash %s, expected %s", i, r.BlockHash, block.Hash)
		}
		for j, log := range r.Logs {
			if log.Index != logIndex {
				return nil, fmt.Errorf("log %d (%d of tx %d) has unexpected log index %d", logIndex, j, i, log.Index)
			}
			if log.TxIndex != uint(i) {
				return nil, fmt.Errorf("log %d has unexpected tx index %d", log.Index, log.TxIndex)
			}
			if log.BlockHash != block.Hash {
				return nil, fmt.Errorf("log %d of block %s has unexpected block hash %s", log.Index, block.Hash, log.BlockHash)
			}
			if log.BlockNumber != block.Number {
				return nil, fmt.Errorf("log %d of block %d has unexpected block number %d", log.Index, block.Number, log.BlockNumber)
			}
			if h := txs[i].Hash(); log.TxHash != h {
				return nil, fmt.Errorf("log %d of tx %s has unexpected tx hash %s", log.Index, h, log.TxHash)
			}
			if log.Removed {
				return nil, fmt.Errorf("canonical log (%d) must never be removed due to reorg", log.Index)
			}
			logIndex++
		}
	}

	// Sanity-check: external L1-RPC sources are notorious for not returning all receipts,
	// or returning them out-of-order. Verify the receipts against the expected receipt-hash.
	hasher := trie.NewStackTrie(nil)
	computed := types.DeriveSha(types.Receipts(receipts), hasher)
	if receiptHash != computed {
		return nil, fmt.Errorf("failed to fetch list of receipts: expected receipt root %s but computed %s from retrieved receipts", receiptHash, computed)
	}
	return receipts, nil
}
