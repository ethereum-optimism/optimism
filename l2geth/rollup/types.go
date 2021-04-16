package rollup

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// OVMContext represents the blocknumber and timestamp
// that exist during L2 execution
type OVMContext struct {
	blockNumber uint64
	timestamp   uint64
}

// SyncType represents the type of transactions that are being synced.
// The different types have different security models.
type SyncType uint

func (s SyncType) String() string {
	switch s {
	case SyncTypeBatched:
		return "batched"
	case SyncTypeSequenced:
		return "sequenced"
	default:
		return ""
	}
}

func NewSyncType(typ string) (SyncType, error) {
	switch typ {
	case "batched":
		return SyncTypeBatched, nil
	case "sequenced":
		return SyncTypeSequenced, nil
	default:
		return 0, fmt.Errorf("Unknown SyncType: %s", typ)
	}
}

const (
	// Batched SyncType involves syncing transactions that have been batched to
	// Layer One. Once the transactions have been batched to L1, they cannot be
	// removed assuming that they are not reorganized out of the chain.
	SyncTypeBatched SyncType = iota
	// Sequenced SyncType involves syncing transactions from the sequencer,
	// meaning that the transactions may have not been batched to Layer One yet.
	// This gives higher latency access to the sequencer data but no gurantees
	// around the transactions as they have not been submitted via a batch to
	// L1.
	SyncTypeSequenced
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
