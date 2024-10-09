package fromda

import (
	"fmt"
	"io"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type state struct {
	// next entry index, including the contents of `out`
	nextEntryIndex entrydb.EntryIdx

	derivedFrom  types.BlockSeal
	derivedUntil uint64 // L2 block that we last derived until starting deriving from this L1 block
	derivedSince uint32 // amount of blocks derived from derivedFrom thus far

	derived types.BlockSeal // produced using L1 data up to and including that of derivedFrom

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

func (l *state) DerivedFrom() (id types.BlockSeal, ok bool) {
	return l.derivedFrom, l.need == 0
}

func (l *state) DerivedSince() (count uint32, ok bool) {
	return l.derivedSince, l.need == 0
}

func (l *state) DerivedUntil() (derivedUntil uint64, ok bool) {
	return l.derivedUntil, l.need == 0
}

func (l *state) Derived() (id types.BlockSeal, ok bool) {
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
		v, err := newSearchCheckpointFromEntry(entry)
		if err != nil {
			return err
		}
		l.derivedFrom = types.BlockSeal{
			Hash:      common.Hash{},
			Number:    v.blockNum,
			Timestamp: v.timestamp,
		}
		l.derivedSince = v.derivedSince
		l.need.Remove(FlagSearchCheckpoint)
		l.need.Add(FlagCanonicalHash)
	case TypeCanonicalHash:
		v, err := newCanonicalHashFromEntry(entry)
		if err != nil {
			return err
		}
		l.derivedFrom.Hash = v.hash
		l.need.Remove(FlagCanonicalHash)
	case TypeDerivedLink:
		v, err := newDerivedLinkFromEntry(entry)
		if err != nil {
			return err
		}
		l.need.Remove(FlagDerivedLink)
		l.need.Add(FlagDerivedCheck)
		l.derived = types.BlockSeal{
			Hash:      common.Hash{},
			Number:    v.number,
			Timestamp: v.timestamp,
		}
	case TypeDerivedCheck:
		v, err := newDerivedCheckFromEntry(entry)
		if err != nil {
			return err
		}
		l.need.Remove(FlagDerivedCheck)
		l.derived.Hash = v.hash
		// we derived a new block!
		l.derivedSince += 1
	case TypePadding:
		l.need.Remove(FlagPadding)
	default:
		return fmt.Errorf("unknown entry type: %s", entry.Type())
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
		l.appendEntry(newSearchCheckpoint(l.derivedFrom.Number, l.derivedFrom.Timestamp, l.derivedSince, l.derivedUntil))
		l.need.Add(FlagCanonicalHash) // always follow with a canonical hash
		l.need.Remove(FlagSearchCheckpoint)
		return nil
	}
	if l.need.Any(FlagCanonicalHash) {
		l.appendEntry(newCanonicalHash(l.derivedFrom.Hash))
		l.need.Remove(FlagCanonicalHash)
		return nil
	}
	if l.need.Any(FlagDerivedLink) {
		// Add padding if this link/check combination is going to overlap with the checkpoint
		switch l.nextEntryIndex % searchCheckpointFrequency {
		case searchCheckpointFrequency - 1:
			l.need.Add(FlagPadding)
			return nil
		}
		l.appendEntry(newDerivedLink(l.derived.Number, l.derived.Timestamp))
		l.need.Remove(FlagDerivedLink)
		l.need.Any(FlagDerivedCheck)
		return nil
	}
	if l.need.Any(FlagDerivedCheck) {
		l.appendEntry(newDerivedCheck(l.derived.Hash))
		l.need.Remove(FlagDerivedCheck)
		// we derived a new L2 block!
		l.derivedSince += 1
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

// AddDerived adds a L1<>L2 block derivation link.
// This may repeat the L1 block if there are multiple L2 blocks derived from it, or repeat the L2 block if the L1 block is empty.
func (l *state) AddDerived(derivedFrom eth.BlockRef, derived eth.BlockRef) error {
	// If we don't have any entries yet, allow any block to start things off
	if l.nextEntryIndex != 0 {
		// TODO insert starting point
	}

	if l.derived.ID() == derived.ID() && l.derivedFrom.ID() == derivedFrom.ID() {
		// Repeat of same information. No entries to be written.
		// But we can silently ignore and not return an error, as that brings the caller
		// in a consistent state, after which it can insert the actual new derived-from information.
		return nil
	}

	// Check derived relation: the L2 chain has to be sequential without gaps. An L2 block may repeat if the L1 block is empty.
	if l.derived.Number == derived.Number {
		// Same block height? Then it must be the same block.
		// I.e. we encountered an empty L1 block, and the same L2 block continues to be the last block that was derived from it.
		if l.derived.Hash != derived.Hash {
			// TODO
		}
	} else if l.derived.Number+1 == derived.Number {
		if l.derived.Hash != derived.ParentHash {
			return fmt.Errorf("derived block %s (parent %s) does not build on %s: %w",
				derived, derived.ParentHash, l.derived, entrydb.ErrConflict)
		}
	} else if l.derived.Number+1 < derived.Number {
		return fmt.Errorf("derived block %s (parent: %s) is too new, expected to build on top of %s: %w",
			derived, derived.ParentHash, l.derived, entrydb.ErrOutOfOrder)
	} else {
		return fmt.Errorf("derived block %s is older than current derived block %s: %w",
			derived, l.derived, entrydb.ErrOutOfOrder)
	}

	// Check derived-from relation: multiple L2 blocks may be derived from the same L1 block. But everything in sequence.
	if l.derivedFrom.Number == derivedFrom.Number {
		// Same block height? Then it must be the same block.
		if l.derivedFrom.Hash != derivedFrom.Hash {
			return fmt.Errorf("cannot add block %s as derived from %s, expected to be derived from %s at this block height: %w",
				derived, derivedFrom, l.derivedFrom, entrydb.ErrConflict)
		}
	} else if l.derivedFrom.Number+1 == derivedFrom.Number {
		// parent hash check
		if l.derivedFrom.Hash != derivedFrom.ParentHash {
			return fmt.Errorf("cannot add block %s as derived from %s (parent %s) derived on top of %s: %w",
				derived, derivedFrom, derivedFrom.ParentHash, l.derivedFrom, entrydb.ErrConflict)
		}
	} else if l.derivedFrom.Number+1 < derivedFrom.Number {
		// adding block that is derived from something too far into the future
		return fmt.Errorf("cannot add block %s as derived from %s, still deriving from %s: %s",
			derived, derivedFrom, l.derivedFrom, entrydb.ErrOutOfOrder)
	} else {
		// adding block that is derived from something too old
		return fmt.Errorf("cannot add block %s as derived from %s, deriving already at %s: %w",
			derived, derivedFrom, l.derivedFrom, entrydb.ErrOutOfOrder)
	}

	if l.derivedFrom.ID() != derivedFrom.ID() {
		// Sanity check our state
		if expected := l.derivedUntil + uint64(l.derivedSince); expected != l.derived.Number {
			panic(fmt.Errorf("expected to have derived up to %d (%d until current L1 block, and %d since then), but have %d",
				expected, l.derivedUntil, l.derivedSince, l.derived.Number))
		}
		l.need.Add(FlagSearchCheckpoint)
		l.derivedUntil += l.derived.Number

		l.derivedFrom = types.BlockSeal{
			Hash:      derivedFrom.Hash,
			Number:    derivedFrom.Number,
			Timestamp: derivedFrom.Time,
		}
	}

	if l.derived.ID() != derived.ID() {
		l.need.Add(FlagDerivedLink)
		l.derived = types.BlockSeal{
			Hash:      derived.Hash,
			Number:    derived.Number,
			Timestamp: derived.Time,
		}
	}

	return l.inferFull()
}
