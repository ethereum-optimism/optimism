package l1

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/core/types"
)

// fetchReceipts fetches the receipts of the transactions using RPC batching, verifies if the receipts are complete and correct, and then returns results
func fetchReceipts(ctx context.Context, receiptHash common.Hash, txs types.Transactions, getBatch batchCallContextFn) (types.Receipts, error) {
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
	for i, r := range receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return nil, fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
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
