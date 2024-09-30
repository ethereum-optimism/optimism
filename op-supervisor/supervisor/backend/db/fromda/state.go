package fromda

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"io"
	"slices"
)

type state struct {
	// next entry index, including the contents of `out`
	nextEntryIndex entrydb.EntryIdx

	derivedFrom eth.BlockID
	derived     eth.BlockID

	need EntryTypeFlag

	// buffer of entries not yet in the DB.
	// This is generated as objects are applied.
	// E.g. you can build things on top of the state,
	// before flushing the entries to a DB.
	// However, no entries can be read from the DB while objects are being applied.
	out []Entry
}

var _ entrydb.IndexState[EntryType, Key] = (*state)(nil)

func (l *state) Key() (k Key, ok bool) {
	return Key{DerivedFrom: l.derivedFrom.Number, Derived: l.derived.Number}, l.need == 0
}

func (l *state) Incomplete() bool {
	return l.need != 0
}

func (l *state) Out() []Entry {
	return slices.Clone(l.out)
}

func (l *state) ClearOut() {
	l.out = l.out[:0]
}

func (l *state) NextIndex() entrydb.EntryIdx {
	return l.nextEntryIndex
}

func (l *state) DerivedFrom() (id eth.BlockID, ok bool) {
	return l.derivedFrom, l.need == 0
}

func (l *state) Derived() (id eth.BlockID, ok bool) {
	return l.derived, l.need == 0
}

// ApplyEntry applies an entry on top of the current state.
func (l *state) ApplyEntry(entry Entry) error {
	// Wrap processEntry to add common useful error message info
	err := l.processEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to process type %s entry at idx %d (%x): %w", entry.Type().String(), l.nextEntryIndex, entry[:], err)
	}
	return nil
}

func (l *state) processEntry(entry Entry) error {
	if len(l.out) != 0 {
		panic("can only apply without appending if the state is still empty")
	}
	switch entry.Type() {
	case TypeSearchCheckpoint:
		current, err := newSearchCheckpointFromEntry(entry)
		if err != nil {
			return err
		}
		// TODO
	}
	return nil
}

// appendEntry add the entry to the output-buffer,
// and registers it as last processed entry type, and increments the next entry-index.
func (l *state) appendEntry(obj EntryObj) {
	entry := obj.encode()
	l.out = append(l.out, entry)
	l.nextEntryIndex += 1
}

// infer advances the logContext in cases where complex entries contain multiple implied entries
// eg. a SearchCheckpoint implies a CannonicalHash will follow
// this also handles inserting the searchCheckpoint at the set frequency, and padding entries
func (l *state) infer() error {
	// We force-insert a checkpoint whenever we hit the known fixed interval.
	if l.nextEntryIndex%searchCheckpointFrequency == 0 {
		l.need.Add(FlagSearchCheckpoint)
	}
	if l.need.Any(FlagSearchCheckpoint) {
		l.appendEntry(newSearchCheckpoint(l.blockNum, l.logsSince, l.timestamp))
		l.need.Add(FlagCanonicalHash) // always follow with a canonical hash
		l.need.Remove(FlagSearchCheckpoint)
		return nil
	}
	if l.need.Any(FlagCanonicalHash) {
		l.appendEntry(newCanonicalHash(l.blockHash))
		l.need.Remove(FlagCanonicalHash)
		return nil
	}

	return io.EOF
}

// inferFull advances the logContext until it cannot infer any more entries.
func (l *state) inferFull() error {
	for i := 0; i < 10; i++ {
		err := l.infer()
		if err == nil {
			continue
		}
		if err == io.EOF { // wrapped io.EOF does not count.
			return nil
		} else {
			return err
		}
	}
	panic("hit sanity limit")
}

// whenever a L2 block is derived from a L1 bloc
func (l *state) AddDerived(derivedFrom eth.BlockID, crossVerified eth.BlockRef) error {
	// TODO check if derived from current block

	// TODO check parent hash matches last known cross verified L2 block
	return nil
}

func (l *state) SealDerivedFrom(derivedFrom eth.BlockRef) error {
	// TODO check parent-hash of seal
	// TODO add checkpoint
	return nil
}
