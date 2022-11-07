package sources

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

func makeReceiptsFn(block eth.BlockID, receiptHash common.Hash) func(txHashes []common.Hash, receipts []*types.Receipt) (types.Receipts, error) {
	return func(txHashes []common.Hash, receipts []*types.Receipt) (types.Receipts, error) {
		if len(receipts) != len(txHashes) {
			return nil, fmt.Errorf("got %d receipts but expected %d", len(receipts), len(txHashes))
		}
		if len(txHashes) == 0 {
			if receiptHash != types.EmptyRootHash {
				return nil, fmt.Errorf("no transactions, but got non-empty receipt trie root: %s", receiptHash)
			}
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
			if r.BlockNumber == nil {
				return nil, fmt.Errorf("receipt %d has unexpected nil block number, expected %d", i, block.Number)
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
				if log.TxHash != txHashes[i] {
					return nil, fmt.Errorf("log %d of tx %s has unexpected tx hash %s", log.Index, txHashes[i], log.TxHash)
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
}

func makeReceiptRequest(txHash common.Hash) (*types.Receipt, rpc.BatchElem) {
	out := new(types.Receipt)
	return out, rpc.BatchElem{
		Method: "eth_getTransactionReceipt",
		Args:   []interface{}{txHash},
		Result: &out, // receipt may become nil, double pointer is intentional
	}
}

// NewReceiptsFetcher creates a receipt fetcher that can iteratively fetch the receipts matching the given txs.
func NewReceiptsFetcher(block eth.BlockID, receiptHash common.Hash, txHashes []common.Hash, getBatch BatchCallContextFn, batchSize int) eth.ReceiptsFetcher {
	return NewIterativeBatchCall[common.Hash, *types.Receipt, types.Receipts](
		txHashes,
		makeReceiptRequest,
		makeReceiptsFn(block, receiptHash),
		getBatch,
		batchSize,
	)
}
