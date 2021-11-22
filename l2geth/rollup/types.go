package rollup

import (
	"bytes"
	"fmt"

	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
)

// OVMContext represents the blocknumber and timestamp
// that exist during L2 execution
type OVMContext struct {
	blockNumber uint64
	timestamp   uint64
}

// Backend represents the type of transactions that are being synced.
// The different types have different security models.
type Backend uint

// String implements the Stringer interface
func (s Backend) String() string {
	switch s {
	case BackendL1:
		return "l1"
	case BackendL2:
		return "l2"
	default:
		return ""
	}
}

// NewBackend creates a Backend from a human readable string
func NewBackend(typ string) (Backend, error) {
	switch typ {
	case "l1":
		return BackendL1, nil
	case "l2":
		return BackendL2, nil
	default:
		return 0, fmt.Errorf("Unknown Backend: %s", typ)
	}
}

const (
	// BackendL1 Backend involves syncing transactions that have been batched to
	// Layer One. Once the transactions have been batched to L1, they cannot be
	// removed assuming that they are not reorganized out of the chain.
	BackendL1 Backend = iota
	// BackendL2 Backend involves syncing transactions from the sequencer,
	// meaning that the transactions may have not been batched to Layer One yet.
	// This gives higher latency access to the sequencer data but no guarantees
	// around the transactions as they have not been submitted via a batch to
	// L1.
	BackendL2
)

func isCtcTxEqual(a, b *types.Transaction) bool {
	if a.To() == nil && b.To() != nil {
		if !bytes.Equal(b.To().Bytes(), common.Address{}.Bytes()) {
			return false
		}
	}
	if a.To() != nil && b.To() == nil {
		if !bytes.Equal(a.To().Bytes(), common.Address{}.Bytes()) {
			return false
		}
		return false
	}
	if a.To() != nil && b.To() != nil {
		if !bytes.Equal(a.To().Bytes(), b.To().Bytes()) {
			return false
		}
	}
	if !bytes.Equal(a.Data(), b.Data()) {
		return false
	}
	if a.L1MessageSender() == nil && b.L1MessageSender() != nil {
		return false
	}
	if a.L1MessageSender() != nil && b.L1MessageSender() == nil {
		return false
	}
	if a.L1MessageSender() != nil && b.L1MessageSender() != nil {
		if !bytes.Equal(a.L1MessageSender().Bytes(), b.L1MessageSender().Bytes()) {
			return false
		}
	}
	if a.Gas() != b.Gas() {
		return false
	}
	return true
}
