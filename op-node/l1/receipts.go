package l1

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/core/types"
)

// ReceiptsFetcher fetches the receipts of the transactions using RPC batching in iterative calls,
// and then verifies if the receipts are complete and correct, and then returns results.
type ReceiptsFetcher struct {
	iterBatchCall *IterativeBatchCall
	receipts      []*types.Receipt
	receiptHash   common.Hash
	block         eth.BlockID
	txs           types.Transactions
	batchSize     uint
}

// Fetch fetches the next batch. Fetch is safe to call concurrently for parallel fetching.
// An error will be returned if data, possibly individual calls as multi-error, fails to be fetched.
// Any individual items that fail to be fetched will automatically be rescheduled for later fetching.
// An io.EOF error will be returned once the fetching is done.
func (rf *ReceiptsFetcher) Fetch(ctx context.Context) error {
	return rf.iterBatchCall.Fetch(ctx, rf.batchSize)
}

func (rf *ReceiptsFetcher) Complete() bool {
	return rf.iterBatchCall.Complete()
}

func (rf *ReceiptsFetcher) Result() (types.Receipts, error) {
	if !rf.iterBatchCall.Complete() {
		return nil, errors.New("no result available yet, fetching is not complete")
	}
	// We don't trust the RPC to provide consistent cached receipt info that we use for critical rollup derivation work.
	// Let's check everything quickly.
	logIndex := uint(0)
	for i, r := range rf.receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return nil, fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
		}
		if r.TransactionIndex != uint(i) {
			return nil, fmt.Errorf("receipt %d has unexpected tx index %d", i, r.TransactionIndex)
		}
		if r.BlockNumber.Uint64() != rf.block.Number {
			return nil, fmt.Errorf("receipt %d has unexpected block number %d, expected %d", i, r.BlockNumber, rf.block.Number)
		}
		if r.BlockHash != rf.block.Hash {
			return nil, fmt.Errorf("receipt %d has unexpected block hash %s, expected %s", i, r.BlockHash, rf.block.Hash)
		}
		for j, log := range r.Logs {
			if log.Index != logIndex {
				return nil, fmt.Errorf("log %d (%d of tx %d) has unexpected log index %d", logIndex, j, i, log.Index)
			}
			if log.TxIndex != uint(i) {
				return nil, fmt.Errorf("log %d has unexpected tx index %d", log.Index, log.TxIndex)
			}
			if log.BlockHash != rf.block.Hash {
				return nil, fmt.Errorf("log %d of block %s has unexpected block hash %s", log.Index, rf.block.Hash, log.BlockHash)
			}
			if log.BlockNumber != rf.block.Number {
				return nil, fmt.Errorf("log %d of block %d has unexpected block number %d", log.Index, rf.block.Number, log.BlockNumber)
			}
			if h := rf.txs[i].Hash(); log.TxHash != h {
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
	computed := types.DeriveSha(types.Receipts(rf.receipts), hasher)
	if rf.receiptHash != computed {
		return nil, fmt.Errorf("failed to fetch list of receipts: expected receipt root %s but computed %s from retrieved receipts", rf.receiptHash, computed)
	}
	return rf.receipts, nil
}

// NewReceiptsFetcher creates a receipt fetcher that can iteratively fetch the receipts matching the given t
func NewReceiptsFetcher(block eth.BlockID, receiptHash common.Hash, txs types.Transactions, getBatch batchCallContextFn, batchSize int) (eth.ReceiptsFetcher, error) {
	if len(txs) == 0 {
		if receiptHash != types.EmptyRootHash {
			return nil, fmt.Errorf("no transactions, but got non-empty receipt trie root: %s", receiptHash)
		}
		return eth.FetchedReceipts(nil), nil
	}
	if len(txs) < batchSize {
		batchSize = len(txs)
	}
	if batchSize < 1 {
		batchSize = 1
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

	return &ReceiptsFetcher{
		iterBatchCall: NewIterativeBatchCall(receiptRequests, getBatch),
		receipts:      receipts,
		receiptHash:   receiptHash,
		block:         block,
		txs:           txs,
		batchSize:     uint(batchSize),
	}, nil
}
