package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

type ReceiptsProvider interface {
	// FetchReceipts returns a block info and all of the receipts associated with transactions in the block.
	// It verifies the receipt hash in the block header against the receipt hash of the fetched receipts
	// to ensure that the execution engine did not fail to return any receipts.
	FetchReceipts(ctx context.Context, block eth.BlockID, txHashes []common.Hash) (types.Receipts, error)
}

// validateReceipts validates that the receipt contents are valid.
// Warning: contractAddress is not verified, since it is a more expensive operation for data we do not use.
// See go-ethereum/crypto.CreateAddress to verify contract deployment address data based on sender and tx nonce.
func validateReceipts(block eth.BlockID, receiptHash common.Hash, txHashes []common.Hash, receipts []*types.Receipt) error {
	if len(receipts) != len(txHashes) {
		return fmt.Errorf("got %d receipts but expected %d", len(receipts), len(txHashes))
	}
	if len(txHashes) == 0 {
		if receiptHash != types.EmptyRootHash {
			return fmt.Errorf("no transactions, but got non-empty receipt trie root: %s", receiptHash)
		}
	}
	// We don't trust the RPC to provide consistent cached receipt info that we use for critical rollup derivation work.
	// Let's check everything quickly.
	logIndex := uint(0)
	cumulativeGas := uint64(0)
	for i, r := range receipts {
		if r == nil { // on reorgs or other cases the receipts may disappear before they can be retrieved.
			return fmt.Errorf("receipt of tx %d returns nil on retrieval", i)
		}
		if r.TransactionIndex != uint(i) {
			return fmt.Errorf("receipt %d has unexpected tx index %d", i, r.TransactionIndex)
		}
		if r.BlockNumber == nil {
			return fmt.Errorf("receipt %d has unexpected nil block number, expected %d", i, block.Number)
		}
		if r.BlockNumber.Uint64() != block.Number {
			return fmt.Errorf("receipt %d has unexpected block number %d, expected %d", i, r.BlockNumber, block.Number)
		}
		if r.BlockHash != block.Hash {
			return fmt.Errorf("receipt %d has unexpected block hash %s, expected %s", i, r.BlockHash, block.Hash)
		}
		if expected := r.CumulativeGasUsed - cumulativeGas; r.GasUsed != expected {
			return fmt.Errorf("receipt %d has invalid gas used metadata: %d, expected %d", i, r.GasUsed, expected)
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
		cumulativeGas = r.CumulativeGasUsed
		// Note: 3 non-consensus L1 receipt fields are ignored:
		// PostState - not part of L1 ethereum anymore since EIP 658 (part of Byzantium)
		// ContractAddress - we do not care about contract deployments
		// And Optimism L1 fee meta-data in the receipt is ignored as well
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
