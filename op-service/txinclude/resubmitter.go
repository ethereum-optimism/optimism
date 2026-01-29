package txinclude

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
)

type Resubmitter struct {
	inner     Sender
	blockTime time.Duration
	cfg       *resubmitterConfig
}

var _ Sender = (*Resubmitter)(nil)

type resubmitterConfig struct {
	observer ResubmitterObserver
}

type ResubmitterOption func(*resubmitterConfig)

func WithObserver(observer ResubmitterObserver) ResubmitterOption {
	return func(cfg *resubmitterConfig) {
		cfg.observer = observer
	}
}

func NewResubmitter(inner Sender, blockTime time.Duration, opts ...ResubmitterOption) *Resubmitter {
	cfg := &resubmitterConfig{
		observer: NoOpResubmitterObserver{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Resubmitter{
		cfg:       cfg,
		inner:     inner,
		blockTime: blockTime,
	}
}

var fatalErrs = []error{
	// Nonces
	core.ErrNonceTooLow,
	core.ErrNonceTooHigh,
	core.ErrNonceMax,
	legacypool.ErrOutOfOrderTxFromDelegated,

	// Fees
	txpool.ErrReplaceUnderpriced,
	txpool.ErrUnderpriced,

	// Transaction limits.
	txpool.ErrOversizedData,
	core.ErrMaxInitCodeSizeExceeded,
	legacypool.ErrAuthorityReserved,
	legacypool.ErrFutureReplacePending,

	// Validity.
	txpool.ErrInvalidSender,
	txpool.ErrGasLimit, // This one could be transient, but in practice it's usually fatal.
	txpool.ErrNegativeValue,
	core.ErrInsufficientFunds,
	core.ErrInsufficientFundsForTransfer,
	core.ErrInsufficientBalanceWitness,
	core.ErrGasUintOverflow,
	core.ErrIntrinsicGas,
	core.ErrFloorDataGas,
	core.ErrTipAboveFeeCap,
	core.ErrTipVeryHigh,
	core.ErrFeeCapVeryHigh,
	core.ErrFeeCapTooLow,
	core.ErrSenderNoEOA,
	core.ErrTxTypeNotSupported,
	// Blobs.
	core.ErrBlobFeeCapTooLow,
	core.ErrMissingBlobHashes,
	core.ErrBlobTxCreate,
	// 7702.
	core.ErrEmptyAuthList,
	core.ErrSetCodeTxCreate,
	// Interop.
	core.ErrTxFilteredOut,
}

var recognizedErrs = append([]error{
	txpool.ErrAlreadyKnown,
	legacypool.ErrTxPoolOverflow,
	// Account limits.
	txpool.ErrAlreadyReserved,
	txpool.ErrAccountLimitExceeded,
	txpool.ErrInflightTxLimitReached,
}, fatalErrs...)

func tryToRecognizeError(err error) error {
	if err == nil {
		return nil
	}
	for _, recognizedErr := range recognizedErrs {
		// TODO(13408): we should not need to use strings.Contains.
		if strings.Contains(err.Error(), recognizedErr.Error()) {
			return recognizedErr
		}
	}
	return err
}

// SendTransaction implements Sender. It will continue resubmitting unless an error is hit
// that the resubmitter considers unfixable with resubmissions alone (e.g., requiring modifications to tx)
// See fatalErrs for the list of these errors.
func (r *Resubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	for {
		err := tryToRecognizeError(r.inner.SendTransaction(ctx, tx))
		r.cfg.observer.SubmissionError(err)

		for _, fatalErr := range fatalErrs {
			if errors.Is(err, fatalErr) {
				return err
			}
		}

		var resubmitDelay time.Duration
		if err == nil || errors.Is(err, txpool.ErrAlreadyKnown) {
			// Resubmit after three blocks after successful inclusion in the mempool.
			// The transaction could be evicted for some reason, so we'll continue to resubmit
			// to make sure it stays in the mempool.
			resubmitDelay = 3 * r.blockTime
		} else {
			// Resubmit after one block on benign and unknown errors.
			// Our goal is to get tx in the mempool and make sure it stays there.
			// It's not there right now, so we should resubmit faster than in the successful case.
			resubmitDelay = r.blockTime
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(resubmitDelay):
		}
	}
}
