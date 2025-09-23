package txinclude

import (
	"slices"
	"sync"
)

// nonceManager tracks nonces for an account and handles gaps in the nonce sequence.
// When transactions fail to be included, their nonces are will be used preferentially
// for future transactions.
type nonceManager struct {
	mu        sync.Mutex
	nextNonce uint64
	gaps      []uint64 // sorted list of nonce gaps
}

// newNonceManager creates a new nonce manager starting at the given nonce.
func newNonceManager(startNonce uint64) *nonceManager {
	return &nonceManager{
		nextNonce: startNonce,
		gaps:      make([]uint64, 0),
	}
}

func (nm *nonceManager) Next() uint64 {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	if len(nm.gaps) > 0 {
		nonce := nm.gaps[0]
		nm.gaps = nm.gaps[1:]
		return nonce
	}
	nonce := nm.nextNonce
	nm.nextNonce++
	return nonce
}

// InsertGap inserts a nonce gap. It is a no-op if nonce is already a gap.
func (nm *nonceManager) InsertGap(nonce uint64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	i, exists := slices.BinarySearch(nm.gaps, nonce)
	if exists {
		return
	}
	nm.gaps = slices.Insert(nm.gaps, i, nonce)
}
