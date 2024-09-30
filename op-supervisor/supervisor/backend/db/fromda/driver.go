package fromda

import (
	"errors"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

type driver struct {
}

func (d driver) Less(a, b Key) bool {
	return a.DerivedFrom < b.DerivedFrom || (a.DerivedFrom == b.DerivedFrom && a.Derived < b.Derived)
}

func (d driver) Copy(src, dst *state) {
	*dst = *src    // shallow copy is enough
	dst.ClearOut() // don't retain output (there shouldn't be any)
}

func (d driver) NewState(nextIndex entrydb.EntryIdx) *state {
	return &state{
		nextEntryIndex: nextIndex,
		derivedFrom:    eth.BlockID{},
		derived:        eth.BlockID{},
		need:           FlagSearchCheckpoint | FlagCanonicalHash,
		out:            nil,
	}
}

func (d driver) KeyFromCheckpoint(e Entry) (Key, error) {
	if e.Type() != TypeSearchCheckpoint {
		return Key{}, errors.New("expected search checkpoint")
	}
	p, err := newSearchCheckpointFromEntry(e)
	if err != nil {
		return Key{}, err
	}
	return Key{DerivedFrom: p.blockNum, Derived: p.lastDerived}, nil
}

func (d driver) ValidEnd(e Entry) bool {
	return e.Type() == TypeCanonicalHash
}

func (d driver) SearchCheckpointFrequency() uint64 {
	return searchCheckpointFrequency
}

var _ entrydb.IndexDriver[EntryType, Key, *state] = (*driver)(nil)
