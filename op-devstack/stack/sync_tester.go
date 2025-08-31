package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SyncTesterID identifies a syncTester by name and chainID, is type-safe, and can be value-copied and used as map key.
type SyncTesterID idWithChain

var _ IDWithChain = (*SyncTesterID)(nil)

const SyncTesterKind Kind = "SyncTester"

func NewSyncTesterID(key string, chainID eth.ChainID) SyncTesterID {
	return SyncTesterID{
		key:     key,
		chainID: chainID,
	}
}

func (id SyncTesterID) String() string {
	return idWithChain(id).string(SyncTesterKind)
}

func (id SyncTesterID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id SyncTesterID) Kind() Kind {
	return SyncTesterKind
}

func (id SyncTesterID) Key() string {
	return id.key
}

func (id SyncTesterID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id SyncTesterID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(SyncTesterKind)
}

func (id *SyncTesterID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(SyncTesterKind, data)
}

func SortSyncTesterIDs(ids []SyncTesterID) []SyncTesterID {
	return copyAndSort(ids, func(a, b SyncTesterID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortSyncTesters(elems []SyncTester) []SyncTester {
	return copyAndSort(elems, func(a, b SyncTester) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ SyncTesterMatcher = SyncTesterID{}

func (id SyncTesterID) Match(elems []SyncTester) []SyncTester {
	return findByID(id, elems)
}

type SyncTester interface {
	Common
	ID() SyncTesterID
	API() apis.SyncTester
}
