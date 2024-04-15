package txmgr

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

var (
	// Returned by CriticalError when there is an incompatible tx type already in the mempool.
	// geth defines this error as txpool.ErrAlreadyReserved in v1.13.14 so we can remove this
	// declaration once op-geth is updated to this version.
	ErrAlreadyReserved = errors.New("address already reserved")

	// Returned by CriticalError when the system is unable to get the tx into the mempool in the
	// alloted time
	ErrMempoolDeadlineExpired = errors.New("failed to get tx into the mempool")
)

// SendState tracks information about the publication state of a given txn. In
// this context, a txn may correspond to multiple different txn hashes due to
// varying gas prices, though we treat them all as the same logical txn. This
// struct is primarily used to determine whether or not the txmgr should abort a
// given txn.
type SendState struct {
	minedTxs map[common.Hash]struct{}
	mu       sync.RWMutex
	now      func() time.Time

	// Config
	nonceTooLowCount    uint64
	txInMempoolDeadline time.Time // deadline to abort at if no transactions are in the mempool

	// Counts of the different types of errors
	successFullPublishCount   uint64 // nil error => tx made it to the mempool
	safeAbortNonceTooLowCount uint64 // nonce too low error

	// Whether any attempt to send the tx resulted in ErrAlreadyReserved
	alreadyReserved bool

	// Miscellaneous tracking
	bumpCount int // number of times we have bumped the gas price
}

// NewSendStateWithNow creates a new send state with the provided clock.
func NewSendStateWithNow(safeAbortNonceTooLowCount uint64, unableToSendTimeout time.Duration, now func() time.Time) *SendState {
	if safeAbortNonceTooLowCount == 0 {
		panic("txmgr: safeAbortNonceTooLowCount cannot be zero")
	}

	return &SendState{
		minedTxs:                  make(map[common.Hash]struct{}),
		safeAbortNonceTooLowCount: safeAbortNonceTooLowCount,
		txInMempoolDeadline:       now().Add(unableToSendTimeout),
		now:                       now,
	}
}

// NewSendState creates a new send state
func NewSendState(safeAbortNonceTooLowCount uint64, unableToSendTimeout time.Duration) *SendState {
	return NewSendStateWithNow(safeAbortNonceTooLowCount, unableToSendTimeout, time.Now)
}

// ProcessSendError should be invoked with the error returned for each
// publication. It is safe to call this method with nil or arbitrary errors.
func (s *SendState) ProcessSendError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record the type of error
	switch {
	case err == nil:
		s.successFullPublishCount++
	case errStringMatch(err, core.ErrNonceTooLow):
		s.nonceTooLowCount++
	case errStringMatch(err, ErrAlreadyReserved):
		s.alreadyReserved = true
	}
}

// TxMined records that the txn with txnHash has been mined and is await
// confirmation. It is safe to call this function multiple times.
func (s *SendState) TxMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.minedTxs[txHash] = struct{}{}
}

// TxNotMined records that the txn with txnHash has not been mined or has been
// reorg'd out. It is safe to call this function multiple times.
func (s *SendState) TxNotMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, wasMined := s.minedTxs[txHash]
	delete(s.minedTxs, txHash)

	// If the txn got reorged and left us with no mined txns, reset the nonce
	// too low count, otherwise we might abort too soon when processing the next
	// error. If the nonce too low errors persist, we want to ensure we wait out
	// the full safe abort count to ensure we have a sufficient number of
	// observations.
	if len(s.minedTxs) == 0 && wasMined {
		s.nonceTooLowCount = 0
	}
}

// CriticalError returns a non-nil error if the txmgr should give up on trying a given txn with the
// target nonce.  This occurs when the set of errors recorded indicates that no further progress
// can be made on this transaction, or if there is an incompatible tx type currently in the
// mempool.
func (s *SendState) CriticalError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch {
	case len(s.minedTxs) > 0:
		// Never abort if our latest sample reports having at least one mined txn.
		return nil
	case s.nonceTooLowCount >= s.safeAbortNonceTooLowCount:
		// we have exceeded the nonce too low count
		return core.ErrNonceTooLow
	case s.successFullPublishCount == 0 && s.now().After(s.txInMempoolDeadline):
		// unable to get the tx into the mempool in the alloted time
		return ErrMempoolDeadlineExpired
	case s.alreadyReserved:
		// incompatible tx type in mempool
		return ErrAlreadyReserved
	}
	return nil
}

// IsWaitingForConfirmation returns true if we have at least one confirmation on
// one of our txs.
func (s *SendState) IsWaitingForConfirmation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.minedTxs) > 0
}
