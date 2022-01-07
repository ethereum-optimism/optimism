package sequencer

import (
	"errors"
	"fmt"

	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
)

var (
	// ErrBlockWithInvalidContext signals an attempt to generate a
	// BatchContext that specifies a total of zero txs.
	ErrBlockWithInvalidContext = errors.New("attempted to generate batch " +
		"context with 0 queued and 0 sequenced txs")
)

// BatchElement reflects the contents of an atomic update to the L2 state.
// Currently, each BatchElement is constructed from a single block containing
// exactly one tx.
type BatchElement struct {
	// Timestamp is the L1 timestamp of the batch.
	Timestamp uint64

	// BlockNumber is the L1 BlockNumber of the batch.
	BlockNumber uint64

	// Tx is the optional transaction that was applied in this batch.
	//
	// NOTE: This field will only be populated for sequencer txs.
	Tx *CachedTx
}

// IsSequencerTx returns true if this batch contains a tx that needs to be
// posted to the L1 CTC contract.
func (b *BatchElement) IsSequencerTx() bool {
	return b.Tx != nil
}

// BatchElementFromBlock constructs a BatchElement from a single L2 block. This
// method expects that there is exactly ONE tx per block. The returned
// BatchElement will reflect whether or not the lone tx is a sequencer tx or a
// queued tx.
func BatchElementFromBlock(block *l2types.Block) BatchElement {
	txs := block.Transactions()
	if len(txs) != 1 {
		panic(fmt.Sprintf("attempting to create batch element from block %d, "+
			"found %d txs instead of 1", block.Number(), len(txs)))
	}

	tx := txs[0]

	// Extract L2 metadata.
	l1BlockNumber := tx.L1BlockNumber().Uint64()
	isSequencerTx := tx.QueueOrigin() == l2types.QueueOriginSequencer

	// Only include sequencer txs in the returned BatchElement.
	var cachedTx *CachedTx
	if isSequencerTx {
		cachedTx = NewCachedTx(tx)
	}

	return BatchElement{
		Timestamp:   block.Time(),
		BlockNumber: l1BlockNumber,
		Tx:          cachedTx,
	}
}

type groupedBlock struct {
	sequenced []BatchElement
	queued    []BatchElement
}

// GenSequencerBatchParams generates a valid AppendSequencerBatchParams from a
// list of BatchElements. The BatchElements are assumed to be ordered in
// ascending order by L2 block height.
func GenSequencerBatchParams(
	shouldStartAtElement uint64,
	blockOffset uint64,
	batch []BatchElement,
) (*AppendSequencerBatchParams, error) {

	var (
		contexts               []BatchContext
		groupedBlocks          []groupedBlock
		txs                    []*CachedTx
		lastBlockIsSequencerTx bool
		lastTimestamp          uint64
		lastBlockNumber        uint64
	)

	// Iterate over the batch elements, grouping the elements according to
	// the following critera:
	//  - All sequencer txs in the same group must have the same timestamp
	//     and block number.
	//  - If sequencer txs exist in a group, they must come before all
	//     queued txs.
	//  - A group should never split consecutive queued txs.
	//
	// Assuming the block and timestamp criteria for sequencer txs are
	// respected within each group, the following are examples of groupings:
	//  - [s]         // sequencer can exist by itself
	//  - [q]         // ququed tx can exist by itself
	//  - [s] [s]     // differing sequencer tx timestamp/blocknumber
	//  - [s q] [s]   // sequencer tx must precede queued tx in group
	//  - [q] [q s]   // INVALID: consecutive queued txs are split
	//  - [q q] [s]   // correct split for preceding case
	//  - [s q] [s q] // alternating sequencer tx interleaved with queued
	for _, el := range batch {
		// To enforce the above groupings, the following condition is
		// used to determine when to create a new batch:
		//  - On the first pass, or
		//  - Whenever a sequecer tx is observed, and:
		//    - The preceding tx was a queued tx, or
		//    - The preceding sequencer tx has a different
		//       block number/timestamp.
		// Note that a sequencer tx is required to create a new group,
		// so a queued tx may ONLY exist as the first element in a group
		// if it is the very first element.
		needsNewGroupOnSequencerTx := !lastBlockIsSequencerTx ||
			el.Timestamp != lastTimestamp ||
			el.BlockNumber != lastBlockNumber
		if len(groupedBlocks) == 0 ||
			(el.IsSequencerTx() && needsNewGroupOnSequencerTx) {

			groupedBlocks = append(groupedBlocks, groupedBlock{})
		}

		// Append the tx to either the sequenced or queued txs,
		// depending on its type.
		cur := len(groupedBlocks) - 1
		if el.IsSequencerTx() {
			groupedBlocks[cur].sequenced =
				append(groupedBlocks[cur].sequenced, el)

			// Gather all sequencer txs, as these will be encoded in
			// the calldata of the batch tx submitted to the L1 CTC
			// contract.
			txs = append(txs, el.Tx)
		} else {
			groupedBlocks[cur].queued =
				append(groupedBlocks[cur].queued, el)
		}

		lastBlockIsSequencerTx = el.IsSequencerTx()
		lastTimestamp = el.Timestamp
		lastBlockNumber = el.BlockNumber
	}

	// For each group, construct the resulting BatchContext.
	for _, block := range groupedBlocks {
		numSequencedTxs := uint64(len(block.sequenced))
		numSubsequentQueueTxs := uint64(len(block.queued))

		// Ensure at least one tx was included in this group.
		if numSequencedTxs == 0 && numSubsequentQueueTxs == 0 {
			return nil, ErrBlockWithInvalidContext
		}

		// Compute the timestamp and block number from for the batch
		// using either the earliest sequenced tx or the earliest queued
		// tx. If a batch has a sequencer tx it is given preference,
		// since it is guaranteed to be the earliest item in the group.
		// Otherwise, we fallback to the earliest queued tx since it was
		// the very first item.
		var (
			timestamp   uint64
			blockNumber uint64
		)
		if numSequencedTxs > 0 {
			timestamp = block.sequenced[0].Timestamp
			blockNumber = block.sequenced[0].BlockNumber
		} else {
			timestamp = block.queued[0].Timestamp
			blockNumber = block.queued[0].BlockNumber
		}

		contexts = append(contexts, BatchContext{
			NumSequencedTxs:       numSequencedTxs,
			NumSubsequentQueueTxs: numSubsequentQueueTxs,
			Timestamp:             timestamp,
			BlockNumber:           blockNumber,
		})
	}

	return &AppendSequencerBatchParams{
		ShouldStartAtElement:  shouldStartAtElement - blockOffset,
		TotalElementsToAppend: uint64(len(batch)),
		Contexts:              contexts,
		Txs:                   txs,
	}, nil
}
