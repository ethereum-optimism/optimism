package txmgr

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// SendState tracks information about the publication state of a given txn. In
// this context, a txn may correspond to multiple different txn hashes due to
// varying gas prices, though we treat them all as the same logical txn. This
// struct is primarily used to determine whether or not the txmgr should abort a
// given txn and retry with a higher nonce.
type SendState struct {
	minedTxs         map[common.Hash]struct{}
	nonceTooLowCount uint64
	mu               sync.RWMutex

	safeAbortNonceTooLowCount uint64
}

// NewSendState parameterizes a new SendState from the passed
// safeAbortNonceTooLowCount.
func NewSendState(safeAbortNonceTooLowCount uint64) *SendState {
	if safeAbortNonceTooLowCount == 0 {
		panic("txmgr: safeAbortNonceTooLowCount cannot be zero")
	}

	return &SendState{
		minedTxs:                  make(map[common.Hash]struct{}),
		nonceTooLowCount:          0,
		safeAbortNonceTooLowCount: safeAbortNonceTooLowCount,
	}
}

// ProcessSendError should be invoked with the error returned for each
// publication. It is safe to call this method with nil or arbitrary errors.
// Currently it only acts on errors containing the ErrNonceTooLow message.
func (s *SendState) ProcessSendError(err error) {
	// Nothing to do.
	if err == nil {
		return
	}

	// Only concerned with ErrNonceTooLow.
	if !strings.Contains(err.Error(), core.ErrNonceTooLow.Error()) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Record this nonce too low observation.
	s.nonceTooLowCount++
}

// TxMined records that the txn with txnHash has been mined and is await
// confirmation. It is safe to call this function multiple times.
func (s *SendState) TxMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.minedTxs[txHash] = struct{}{}
}

// TxMined records that the txn with txnHash has not been mined or has been
// reorg'd out. It is safe to call this function multiple times.
func (s *SendState) TxNotMined(txHash common.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, wasMined := s.minedTxs[txHash]
	delete(s.minedTxs, txHash)

	// If the txn got reorged and left us with no mined txns, reset the nonce
	// too low count, otherwise we might abort too soon when processing the next
	// error. If the nonce too low errors persist, we want to ensure we wait out
	// the full safe abort count to enesure we have a sufficient number of
	// observations.
	if len(s.minedTxs) == 0 && wasMined {
		s.nonceTooLowCount = 0
	}
}

// ShouldAbortImmediately returns true if the txmgr should give up on trying a
// given txn with the target nonce. For now, this only happens if we see an
// extended period of getting ErrNonceTooLow without having a txn mined.
func (s *SendState) ShouldAbortImmediately() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Never abort if our latest sample reports having at least one mined txn.
	if len(s.minedTxs) > 0 {
		return false
	}

	// Only abort if we've observed enough ErrNonceTooLow to meet our safe abort
	// threshold.
	return s.nonceTooLowCount >= s.safeAbortNonceTooLowCount
}

// IsWaitingForConfirmation returns true if we have at least one confirmation on
// one of our txs.
func (s *SendState) IsWaitingForConfirmation() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.minedTxs) > 0
}
