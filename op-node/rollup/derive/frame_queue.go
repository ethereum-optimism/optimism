package derive

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// This is a code PoC (non compiling / functional) to show a fully pull based
// integration of smaller stages.
//
// Some stages retain state (due to a requirement of a future stage). Others
// have no state (besides config + knowing the previous state) and are simply
// data transformations.
//
// Something not implemented is the tagging of the L1 Inclusion block, but that
// will be metadata passed along with the data
// This is also missing resets / NewFooBar functions, but given the simplicity of
// the stored state, they are incredibly easy to implement (or no-ops in some cases)
//
// One of the tricky things that complicates this code is the reliance on `io.EOF`.
// It is probably better to switch to a custom error and enforce the invariant that
// if the sentinel error (or any error for that matter) is returned, no data is returned
// and if data is returned, err == nil. This makes it easy to check for a non-nil error
// and immediately exit rather than checking for non nil + non io.EOF error, checking for
// data, and then checking for an io.EOF
//
// This code can replace most of what exists from the L1 traversal stage up to the channel bank
// stage. The channel bank stage can also be simplified to accept frames + output channels.

type FrameQueue struct {
	frames []Frame // TODO track L1 Inclusion block in frame . TODO: proper queue here
	// TODO: hook up struct / interface instead??
	getterFn func() ([]Frame, error)
}

// NextFrame gets the next frame from the FrameQueue
// It returns io.EOF when it is no longer able to make progress (i.e. the underlying
// frame supplier returns io.EOF and it has no buffered frames).
func (f *FrameQueue) NextFrame() (Frame, error) {
	if len(f.frames) == 0 {
		if new, err := f.getterFn(); err != nil && err != io.EOF {
			return Frame{}, err
		} else if len(new) > 0 {
			f.frames = append(f.frames, new...)
		} else if err == io.EOF {
			return Frame{}, io.EOF
		}

	}
	frame := f.frames[0]
	f.frames = f.frames[1:]
	return frame, nil
}

// FrameParser reads frame(s) from single transactions.
// It is a pure data transformation stage and returns io.EOF
// when the previous stage returns io.EOF.
type FrameParser struct {
	getterFn func() (*types.Transaction, error)
}

func (f *FrameParser) NextFrames() ([]Frame, error) {
	if tx, err := f.getterFn(); err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, err // TODO: wrap this error
	} else {
		return ParseFrames(tx.Data())
	}

}

type TransactionQueue struct {
	txns     types.Transactions
	getterFn func() (types.Transactions, error)
}

func (t *TransactionQueue) NextTransaction() (*types.Transaction, error) {
	if len(t.txns) == 0 {
		if txns, err := t.getterFn(); err != nil && err != io.EOF {
			return nil, err
		} else if len(txns) > 0 {
			t.txns = append(t.txns, txns...)
		} else if err == io.EOF {
			return nil, io.EOF
		}
	}
	tx := t.txns[0]
	t.txns = t.txns[1:]
	return tx, nil
}

// TransactionFilterer is a pure function from transactions + config to a subset of
// those transactions.
type TransactionFilterer struct {
	// Configuration to filter transactions
	l1Signer           types.Signer
	batchInboxAddress  common.Address
	batchSenderAddress common.Address

	getterFn func() (types.Transactions, error)

	log log.Logger
}

// NextTransactions returns the transactions to the batch inbox that are also properly authorized.
func (t *TransactionFilterer) NextTransactions() (types.Transactions, error) {
	txns, err := t.getterFn()
	// TODO: case where err == io.EOF && txns != nil
	if err != nil {
		return nil, err
	}
	// Filter transactions
	var ret types.Transactions
	for i, tx := range txns {
		if to := tx.To(); to != nil && *to == t.batchInboxAddress {
			seqDataSubmitter, err := t.l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				t.log.Warn("tx in inbox with invalid signature", "index", i, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != t.batchSenderAddress {
				t.log.Warn("tx in inbox with unauthorized submitter", "index", i, "err", err)
				continue // not an authorized batch submitter, ignore
			}
			ret = append(ret, tx)
		}
	}
	return ret, nil
}
