package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

// FrameWithL1Inclusion tags a frame with the L1 Block that is was derived from
type FrameWithL1Inclusion struct {
	frame            Frame
	l1InclusionBlock eth.L1BlockRef
}

// FrameQueue maintains a queue of frames from L1. It fetches transactions from L1
// and does all filtering/parsing when `ProcessOrigin` is called.
type FrameQueue struct {
	// Internal buffer of frames.
	frames []FrameWithL1Inclusion

	// Supporting data / objects
	fetcher            L1TransactionFetcher // Fetcher for transactions
	l1Signer           types.Signer         // signer to recover the sender address
	batchInboxAddress  common.Address       // the `to` address of batch submissions
	batchSenderAddress common.Address       // the address of authorized batch submitters
	log                log.Logger
}

// NewFrameQueue creates a new frame queue.
func NewFrameQueue(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher) *FrameQueue {
	return &FrameQueue{
		fetcher:            fetcher,
		l1Signer:           cfg.L1Signer(),
		batchInboxAddress:  cfg.BatchInboxAddress,
		batchSenderAddress: cfg.BatchSenderAddress,
		log:                log,
	}
}

// NextFrame returns either the next frame or io.EOF if there are no more frames.
// No other errors will be returned.
func (f *FrameQueue) NextFrame() (FrameWithL1Inclusion, error) {
	if len(f.frames) == 0 {
		return FrameWithL1Inclusion{}, io.EOF
	}
	ret := f.frames[0]
	f.frames = f.frames[1:]
	return ret, nil
}

// ProcessOrigin fetches all transactions from a specific L2 block and then loads them into it's internal frame
// buffer. This function should only be called once per origin. This function will only err on a fetcher error.
func (f *FrameQueue) ProcessOrigin(ctx context.Context, origin eth.L1BlockRef) error {
	if _, txs, err := f.fetcher.InfoAndTxsByHash(ctx, origin.Hash); errors.Is(err, ethereum.NotFound) {
		return NewResetError(fmt.Errorf("failed to find transactions to process a L1 Block: %w", err))
	} else if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find transactions to process a L1 Block: %w", err))
	} else {
		f.parseFrames(f.filterTransactions(txs), origin)
		return nil
	}
}

// filterTransactions returns the valid batch submission transactions (transactions that
// are sent to the batch inbox address from the batch sender address).
func (f *FrameQueue) filterTransactions(txs types.Transactions) types.Transactions {
	var out []*types.Transaction
	for i, tx := range txs {
		if to := tx.To(); to != nil && *to == f.batchInboxAddress {
			seqDataSubmitter, err := f.l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				f.log.Warn("tx in inbox with invalid signature", "index", i, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != f.batchSenderAddress {
				f.log.Warn("tx in inbox with unauthorized submitter", "index", i, "err", err)
				continue // not an authorized batch submitter, ignore
			}
			out = append(out, tx)
		}
	}
	return out
}

// parseFrames parses the set of frames in each supplied transaction. It then stores
// each frame as a FrameWithL1Inclusion in queue.
func (f *FrameQueue) parseFrames(txs types.Transactions, origin eth.L1BlockRef) {
	for i, tx := range txs {
		if frames, err := ParseFrames(tx.Data()); err != nil {
			f.log.Warn("failed to parse frame on tx %v: %w", i, err)
		} else {
			for _, frame := range frames {
				f.frames = append(f.frames, FrameWithL1Inclusion{frame: frame, l1InclusionBlock: origin})
			}
		}
	}
}

// Clear removes stored frames from the frame queue.
func (f *FrameQueue) Clear() {
	f.frames = f.frames[:]
}
