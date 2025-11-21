package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SyncTesterID identifies a syncTester by name and chainID, is type-safe, and can be value-copied and used as map key.
type SyncTesterID ComponentID

const SyncTesterKind Kind = "SyncTester"

func NewSyncTesterID(key string) ComponentID {
	return ComponentID{
		Kind: SyncTesterKind,
		Key:  key,
	}
}

func SortSyncTesters(elems []SyncTester) []SyncTester {
	return copyAndSort(elems, func(a, b SyncTester) bool {
		return isLess(ComponentID(a.ID()), ComponentID(b.ID()))
	})
}

type SyncTester interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
	API() apis.SyncTester

	APIWithSession(sessionID string) apis.SyncTester
}
