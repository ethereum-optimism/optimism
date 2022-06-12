package derive

import (
	"bytes"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core/types"
)

func BatchesFromEVMTransactions(config *rollup.Config, txLists []types.Transactions) ([]*BatchData, []error) {
	var out []*BatchData
	var errs []error
	l1Signer := config.L1Signer()
	for i, txs := range txLists {
		for j, tx := range txs {
			if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
				seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
				if err != nil {
					errs = append(errs, fmt.Errorf("invalid signature: tx list: %d, tx: %d, err: %w", i, j, err))
					continue // bad signature, ignore
				}
				// some random L1 user might have sent a transaction to our batch inbox, ignore them
				if seqDataSubmitter != config.BatchSenderAddress {
					errs = append(errs, fmt.Errorf("unauthorized batch submitter: tx list: %d, tx: %d", i, j))
					continue // not an authorized batch submitter, ignore
				}
				batches, err := DecodeBatches(config, bytes.NewReader(tx.Data()))
				if err != nil {
					errs = append(errs, fmt.Errorf("invalid batch: tx list: %d, tx: %d, err: %w", i, j, err))
					continue
				}
				out = append(out, batches...)
			}
		}
	}
	return out, errs
}

func FilterBatches(config *rollup.Config, epoch rollup.Epoch, minL2Time uint64, maxL2Time uint64, batches []*BatchData) (out []*BatchData) {
	uniqueTime := make(map[uint64]struct{})
	for _, batch := range batches {
		if !ValidBatch(batch, config, epoch, minL2Time, maxL2Time) {
			continue
		}
		// Check if we have already seen a batch for this L2 block
		if _, ok := uniqueTime[batch.Timestamp]; ok {
			// block already exists, batch is duplicate (first batch persists, others are ignored)
			continue
		}
		uniqueTime[batch.Timestamp] = struct{}{}
		out = append(out, batch)
	}
	return
}

func ValidBatch(batch *BatchData, config *rollup.Config, epoch rollup.Epoch, minL2Time uint64, maxL2Time uint64) bool {
	if batch.Epoch != epoch {
		// Batch was tagged for past or future epoch,
		// i.e. it was included too late or depends on the given L1 block to be processed first.
		return false
	}
	if (batch.Timestamp-config.Genesis.L2Time)%config.BlockTime != 0 {
		return false // bad timestamp, not a multiple of the block time
	}
	if batch.Timestamp < minL2Time {
		return false // old batch
	}
	// limit timestamp upper bound to avoid huge amount of empty blocks
	if batch.Timestamp >= maxL2Time {
		return false // too far in future
	}
	for _, txBytes := range batch.Transactions {
		if len(txBytes) == 0 {
			return false // transaction data must not be empty
		}
		if txBytes[0] == types.DepositTxType {
			return false // sequencers may not embed any deposits into batch data
		}
	}
	return true
}

// FillMissingBatches turns a collection of batches to the input batches for a series of blocks
func FillMissingBatches(batches []*BatchData, epoch, blockTime, minL2Time, nextL1Time uint64) []*BatchData {
	m := make(map[uint64]*BatchData)
	// The number of L2 blocks per sequencing window is variable, we do not immediately fill to maxL2Time:
	// - ensure at least 1 block
	// - fill up to the next L1 block timestamp, if higher, to keep up with L1 time
	// - fill up to the last valid batch, to keep up with L2 time
	newHeadL2Timestamp := minL2Time
	if nextL1Time > newHeadL2Timestamp+blockTime {
		newHeadL2Timestamp = nextL1Time - blockTime
	}
	for _, b := range batches {
		m[b.BatchV1.Timestamp] = b
		if b.Timestamp > newHeadL2Timestamp {
			newHeadL2Timestamp = b.Timestamp
		}
	}
	var out []*BatchData
	for t := minL2Time; t <= newHeadL2Timestamp; t += blockTime {
		b, ok := m[t]
		if ok {
			out = append(out, b)
		} else {
			out = append(out, &BatchData{
				BatchV1{
					Epoch:     rollup.Epoch(epoch),
					Timestamp: t,
				},
			})
		}
	}
	return out
}
