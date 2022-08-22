package l1

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum/go-ethereum/core/types"
)

// ReceiptsFetcher fetches the receipts of the transactions using RPC batching in iterative calls,
// and then verifies if the receipts are complete and correct, and then returns results.
type ReceiptsFetcher struct {
	mu            sync.RWMutex // mu locks iterBatchCall, read=fetch (parallel with other reads), reset=write
	iterBatchCall *IterativeBatchCall
	receipts      []*types.Receipt
	receiptHash   common.Hash
	block         eth.BlockID
	txHashes      []common.Hash
	batchSize     uint
}

// Fetch fetches the next batch. Fetch is safe to call concurrently for parallel fetching.
// An error will be returned if data, possibly individual calls as multi-error, fails to be fetched.
// Any individual items that fail to be fetched will automatically be rescheduled for later fetching.
// An io.EOF error will be returned once the fetching is done.
func (rf *ReceiptsFetcher) Fetch(ctx context.Context) error {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.iterBatchCall.Fetch(ctx, rf.batchSize)
}

func (rf *ReceiptsFetcher) Complete() bool {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.iterBatchCall.Complete()
}

func (rf *ReceiptsFetcher) Reset() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.iterBatchCall = NewIterativeBatchCall(rf.iterBatchCall.requests, rf.iterBatchCall.getBatch)
}

func (rf *ReceiptsFetcher) Result() (types.Receipts, error) {
	rf.mu.RLock()
	if !rf.iterBatchCall.Complete() {
		rf.mu.RUnlock()
		return nil, errors.New("no result available yet, receipt fetching is not complete")
	}
	err := checkReceipts(rf.receipts, rf.block, rf.receiptHash, rf.txHashes)
	rf.mu.RUnlock()
	if err != nil {
		rf.Reset() // if we got invalid results then restart the call, we may get valid results after a reorg.
		return nil, fmt.Errorf("results are invalid, receipt fetching failed: %w", err)
	}
	return rf.receipts, nil
}

func checkReceipts(receipts []*types.Receipt, block eth.BlockID, receiptHash common.Hash, txHashes []common.Hash) error {
	if len(receipts) != len(txHashes) {
		return fmt.Errorf("got %d receipts but expected %d", len(receipts), len(txHashes))
	}
	// We don't trust the RPC to provide consistent cached receipt info that we use for critical rollup derivation work.
	// Let's check everything quickly.
	logIndex := uint(0)
	for i, r := range receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
		}
		if r.TransactionIndex != uint(i) {
			return fmt.Errorf("receipt %d has unexpected tx index %d", i, r.TransactionIndex)
		}
		if r.BlockNumber.Uint64() != block.Number {
			return fmt.Errorf("receipt %d has unexpected block number %d, expected %d", i, r.BlockNumber, block.Number)
		}
		if r.BlockHash != block.Hash {
			return fmt.Errorf("receipt %d has unexpected block hash %s, expected %s", i, r.BlockHash, block.Hash)
		}
		for j, log := range r.Logs {
			if log.Index != logIndex {
				return fmt.Errorf("log %d (%d of tx %d) has unexpected log index %d", logIndex, j, i, log.Index)
			}
			if log.TxIndex != uint(i) {
				return fmt.Errorf("log %d has unexpected tx index %d", log.Index, log.TxIndex)
			}
			if log.BlockHash != block.Hash {
				return fmt.Errorf("log %d of block %s has unexpected block hash %s", log.Index, block.Hash, log.BlockHash)
			}
			if log.BlockNumber != block.Number {
				return fmt.Errorf("log %d of block %d has unexpected block number %d", log.Index, block.Number, log.BlockNumber)
			}
			if log.TxHash != txHashes[i] {
				return fmt.Errorf("log %d of tx %s has unexpected tx hash %s", log.Index, txHashes[i], log.TxHash)
			}
			if log.Removed {
				return fmt.Errorf("canonical log (%d) must never be removed due to reorg", log.Index)
			}
			logIndex++
		}
	}

	// Sanity-check: external L1-RPC sources are notorious for not returning all receipts,
	// or returning them out-of-order. Verify the receipts against the expected receipt-hash.
	hasher := trie.NewStackTrie(nil)
	computed := types.DeriveSha(types.Receipts(receipts), hasher)
	if receiptHash != computed {
		return fmt.Errorf("failed to fetch list of receipts: expected receipt root %s but computed %s from retrieved receipts", receiptHash, computed)
	}
	return nil
}

// NewReceiptsFetcher creates a receipt fetcher that can iteratively fetch the receipts matching the given txs.
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
	txHashes := make([]common.Hash, len(txs))
	for i := 0; i < len(txs); i++ {
		receipts[i] = new(types.Receipt)
		txHashes[i] = txs[i].Hash()
		receiptRequests[i] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txHashes[i]},
			Result: &receipts[i], // receipt may become nil, double pointer is intentional
		}
	}

	return &ReceiptsFetcher{
		iterBatchCall: NewIterativeBatchCall(receiptRequests, getBatch),
		receipts:      receipts,
		receiptHash:   receiptHash,
		block:         block,
		txHashes:      txHashes,
		batchSize:     uint(batchSize),
	}, nil
}
