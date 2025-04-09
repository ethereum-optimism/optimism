package fromda

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type EntryStore interface {
	Size() int64
	LastEntryIdx() entrydb.EntryIdx
	Read(idx entrydb.EntryIdx) (Entry, error)
	Append(entries ...Entry) error
	Truncate(idx entrydb.EntryIdx) error
	Close() error
}

// DB implements an append only database for log data and cross-chain dependencies.
// Each entry is fixed size, and denotes an increment in L1 (source) and/or L2 (derived) block.
//
// Data is an append-only log, that can be binary searched for any necessary derivation-link data.
//
// The DB only rewinds when the L1 (source) entries are invalidated.
// If the L2 (derived) entries are no longer valid (due to cross-chain dependency invalidation),
// then we register the new L2 entries with a higher revision number
// (matching the block-number of the block where it first invalidated and replaced).
//
// The key-space of the DB is thus as following, in order:
// - source block number -> incremental, there may be adjacent repeat entries (for multiple L2 blocks derived from same L1 block)
// - revision number -> incremental, repeats until the chain invalidates something
// - derived block number -> NOT incremental, but incremental within the scope of a single revision.
//
// This key-space allows for fast binary-search over the source blocks to find any derived blocks,
// but also the reverse: if the revision is known, the derived blocks can be searched, to find the relevant source data.
type DB struct {
	log    log.Logger
	m      Metrics
	store  EntryStore
	rwLock sync.RWMutex
}

func NewFromFile(logger log.Logger, m Metrics, path string) (*DB, error) {
	store, err := entrydb.NewEntryDB[EntryType, Entry, EntryBinary](logger, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	return NewFromEntryStore(logger, m, store)
}

func NewFromEntryStore(logger log.Logger, m Metrics, store EntryStore) (*DB, error) {
	db := &DB{
		log:   logger,
		m:     m,
		store: store,
	}
	db.m.RecordDBDerivedEntryCount(db.store.Size())
	return db, nil
}

// First returns the first known values, alike to Latest.
func (db *DB) First() (pair types.DerivedBlockSealPair, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	lastIndex := db.store.LastEntryIdx()
	if lastIndex < 0 {
		return types.DerivedBlockSealPair{}, types.ErrFuture
	}
	last, err := db.readAt(0)
	if err != nil {
		return types.DerivedBlockSealPair{}, fmt.Errorf("failed to read first derivation data: %w", err)
	}
	return last.sealOrErr()
}

// PreviousDerived returns the previous derived block.
// Warning: only safe to use on cross-DB.
// This will prioritize the last time the input L2 block number was seen, and consistency-checks it against the hash.
func (db *DB) PreviousDerived(derived eth.BlockID, revision types.Revision) (prevDerived types.BlockSeal, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	// last is always the latest view, and thus canonical.
	_, lastCanonical, err := db.derivedNumToLastSource(derived.Number, revision)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("failed to find last derived %d: %w", derived.Number, err)
	}
	// get the first time this L2 block was seen.
	selfIndex, self, err := db.derivedNumToFirstSource(derived.Number, revision)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("failed to find first derived %d: %w", derived.Number, err)
	}
	// The first entry might not match, since it may have been invalidated with a later L1 scope.
	// But the last entry should always match.
	if lastCanonical.derived.ID() != derived {
		return types.BlockSeal{}, fmt.Errorf("found %s, but expected %s: %w", self.derived, derived, types.ErrConflict)
	}
	if selfIndex == 0 { // genesis block has a zeroed block as parent block
		return types.BlockSeal{}, nil
	}
	prev, err := db.readAt(selfIndex - 1)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("cannot find previous derived before %s: %w", derived, err)
	}
	return prev.derived, nil
}

// Last returns the last known values:
// source: the L1 block that the L2 block is safe for (not necessarily the first, multiple L2 blocks may be derived from the same L1 block).
// derived: the L2 block that was derived (not necessarily the first, the L1 block may have been empty and repeated the last safe L2 block).
// If the last entry is invalidated, this returns a types.ErrAwaitReplacementBlock error.
func (db *DB) Last() (pair types.DerivedBlockSealPair, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	link, err := db.latest()
	if err != nil {
		return types.DerivedBlockSealPair{}, err
	}
	return link.sealOrErr()
}

func (db *DB) LastRevision() (revision types.Revision, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	link, err := db.latest()
	if err != nil {
		return types.Revision(0), err
	}
	return link.revision, nil
}

// latest is like Latest, but without lock, for internal use.
func (db *DB) latest() (link LinkEntry, err error) {
	lastIndex := db.store.LastEntryIdx()
	if lastIndex < 0 {
		return LinkEntry{}, types.ErrFuture
	}
	last, err := db.readAt(lastIndex)
	if err != nil {
		return LinkEntry{}, fmt.Errorf("failed to read last derivation data: %w", err)
	}
	return last, nil
}

func (db *DB) Invalidated() (pair types.DerivedBlockSealPair, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	link, err := db.latest()
	if err != nil {
		return types.DerivedBlockSealPair{}, err
	}
	if !link.invalidated {
		return types.DerivedBlockSealPair{}, fmt.Errorf("last entry %s is not invalidated: %w", link, types.ErrConflict)
	}
	return types.DerivedBlockSealPair{
		Source:  link.source,
		Derived: link.derived,
	}, nil
}

// SourceToLastDerived returns the last L2 block derived from the given L1 block.
// This may return types.ErrAwaitReplacementBlock if the entry was invalidated and needs replacement.
func (db *DB) SourceToLastDerived(source eth.BlockID) (derived types.BlockSeal, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	_, link, err := db.sourceNumToLastDerived(source.Number)
	if err != nil {
		return types.BlockSeal{}, err
	}
	if link.source.ID() != source {
		return types.BlockSeal{}, fmt.Errorf("searched for last derived-from %s but found %s: %w",
			source, link.source, types.ErrConflict)
	}
	if link.invalidated {
		return types.BlockSeal{}, types.ErrAwaitReplacementBlock
	}
	return link.derived, nil
}

// NextDerived finds the next L2 block after derived, and what it was derived from.
// This may return types.ErrAwaitReplacementBlock if the entry was invalidated and needs replacement.
// This will prioritize the last time the input L2 block number was seen, and consistency-checks it against the hash.
// Older occurrences of the same number with different hash cannot be iterated from, and are non-canonical.
func (db *DB) NextDerived(derived eth.BlockID, revision types.Revision) (pair types.DerivedBlockSealPair, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	// Get the last time this L2 block was seen. This is attached to the latest L1 view, and thus canonical.
	selfIndex, self, err := db.derivedNumToLastSource(derived.Number, revision)
	if err != nil {
		return types.DerivedBlockSealPair{}, fmt.Errorf("failed to find derived %d: %w", derived.Number, err)
	}
	if self.derived.ID() != derived {
		return types.DerivedBlockSealPair{}, fmt.Errorf("found %s, but expected %s: %w", self.derived, derived, types.ErrConflict)
	}
	next, err := db.readAt(selfIndex + 1)
	if err != nil {
		return types.DerivedBlockSealPair{}, fmt.Errorf("cannot find next derived after %s: %w", derived, err)
	}
	return next.sealOrErr()
}

// Candidate returns the candidate block for cross-safe promotion after the given
// (maxSource, afterDerived) pair (the cross-safe block).
// It may return types.ErrOutOfScope with a pair value, to request the source to increase.
func (db *DB) Candidate(maxSource eth.BlockID, afterDerived eth.BlockID, revision types.Revision) (pair types.DerivedBlockRefPair, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	_, lastOffered, err := db.sourceNumToLastDerived(maxSource.Number)
	if err != nil {
		return types.DerivedBlockRefPair{}, fmt.Errorf("failed to get last derived block from %s: %w", maxSource, err)
	}
	if lastOffered.source.ID() != maxSource {
		return types.DerivedBlockRefPair{}, fmt.Errorf("expected source %s, but got %s: %w", maxSource, afterDerived, types.ErrConflict)
	}
	maxSourceSeal := lastOffered.source

	// attach the parent (or zero-block) to the cross-safe source
	var sourceRef eth.BlockRef
	parentSource, err := db.PreviousSource(maxSource)
	if errors.Is(err, types.ErrPreviousToFirst) {
		// if we are working with the first item in the database, PreviousSource will return ErrPreviousToFirst
		// in which case we can attach a zero parent to the block, as the parent block is unknown
		// ForceWithParent will not panic if the parent is not as expected (like a zero-block)
		sourceRef = maxSourceSeal.ForceWithParent(eth.BlockID{})
	} else if err != nil {
		return types.DerivedBlockRefPair{}, fmt.Errorf("failed to find parent-block of derived-from %s: %w", maxSourceSeal, err)
	} else {
		// if we have a parent, we can attach it to the cross-safe source
		// MustWithParent will panic if the parent is not the previous block
		sourceRef = maxSourceSeal.MustWithParent(parentSource.ID())
	}

	if lastOffered.derived.Number > afterDerived.Number {
		// keep source, just find the next canonical entry.
		// That entry can be attached to an older source however.
		_, link, err := db.derivedNumToLastSource(afterDerived.Number+1, revision)
		if errors.Is(err, types.ErrNotExact) {
			_, link, err = db.derivedNumToLastSource(afterDerived.Number+1, types.Revision(afterDerived.Number+1))
			if err != nil {
				return types.DerivedBlockRefPair{}, fmt.Errorf("cannot find derived block %d: %w", afterDerived.Number+1, types.ErrDataCorruption)
			}
		}
		derivedRef, err := link.derived.WithParent(afterDerived)
		if err != nil {
			return types.DerivedBlockRefPair{}, err
		}
		return types.DerivedBlockRefPair{
			Source:  sourceRef,
			Derived: derivedRef,
		}, nil
	} else {
		// need scope-bump before we can access next derived values
		return types.DerivedBlockRefPair{
			Source:  sourceRef,
			Derived: eth.BlockRef{},
		}, types.ErrOutOfScope
	}
}

// DerivedToRevision retrieves the revision of the latest occurrence of the given block.
// This may be open-ended (read: match not-yet cross-safe blocks) if the block is not known yet.
// WARNING: this is only safe to use on the cross-safe DB.
func (db *DB) DerivedToRevision(derived eth.BlockID) (types.Revision, error) {
	_, link, err := db.derivedNumToLastSource(derived.Number, types.RevisionAny)
	if err != nil {
		// This often happens in the cross-safe DB,
		// when looking for the revision of a local-safe entry that is not yet in the cross-safe DB.
		return types.Revision(0), fmt.Errorf("failed to get link entry: %w", err)
	}
	if id := link.derived.ID(); id != derived {
		return types.Revision(0), fmt.Errorf("cannot determine revision, db entry %s does not match query %s: %w", id, derived, types.ErrConflict)
	}
	return link.revision, nil
}

// SourceToRevision lookups a specific source entry, and returns the corresponding revision.
// This ignores invalidation status of the given pair.
// It is used in those cases also, to determine the revision to use for a cross-safe DB,
// based on the invalidated entry in the local-safe DB.
func (db *DB) SourceToRevision(source eth.BlockID) (types.Revision, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()

	if db.store.Size() == 0 {
		return FirstRevision, nil
	}
	_, link, err := db.sourceNumToLastDerived(source.Number)
	if err != nil {
		return types.Revision(0), err
	}
	if link.source.ID() != source {
		return types.Revision(0), fmt.Errorf("expected %s, got %s: %w", source, link.source, types.ErrConflict)
	}
	return link.revision, nil
}

// ContainsDerived checks if the given block is canonical for the given chain.
// This returns an ErrFuture if the block is not known yet.
// An ErrConflict if there is a different block.
// Or an ErrAwaitReplacementBlock if it was invalidated.
func (db *DB) ContainsDerived(derived eth.BlockID, revision types.Revision) error {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()

	// Take the last entry: this will be the latest canonical view,
	// if the block was previously invalidated.
	_, link, err := db.derivedNumToLastSource(derived.Number, revision)
	if err != nil {
		return err
	}
	if link.derived.ID() != derived {
		return fmt.Errorf("searched if derived %s but found %s: %w",
			derived, link.derived, types.ErrConflict)
	}
	if link.invalidated {
		return fmt.Errorf("derived %s, but invalidated it: %w", derived, types.ErrAwaitReplacementBlock)
	}
	return nil
}

// DerivedToFirstSource determines where a L2 block was first derived from.
//   - A L2 block may repeat if the following L1 blocks are empty and don't produce additional L2 blocks
//   - A L2 block may reoccur later (with a gap) attached to a newer L1 block,
//     if the prior information was invalidated with new L1 information.
func (db *DB) DerivedToFirstSource(derived eth.BlockID, revision types.Revision) (types.BlockSeal, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	_, link, err := db.derivedNumToFirstSource(derived.Number, revision)
	if err != nil {
		return types.BlockSeal{}, err
	}
	if link.derived.ID() != derived {
		return types.BlockSeal{}, fmt.Errorf("searched for first derived %s but found %s: %w",
			derived, link.derived, types.ErrConflict)
	}
	return link.source, nil
}

func (db *DB) PreviousSource(source eth.BlockID) (types.BlockSeal, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	return db.previousSource(source)
}

func (db *DB) previousSource(source eth.BlockID) (types.BlockSeal, error) {
	// get the last time this L1 block was seen.
	selfIndex, self, err := db.sourceNumToFirstDerived(source.Number)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("failed to find derived %d: %w", source.Number, err)
	}
	if self.source.ID() != source {
		return types.BlockSeal{}, fmt.Errorf("found %s, but expected %s: %w", self.source, source, types.ErrConflict)
	}
	if selfIndex == 0 {
		// genesis block has a zeroed block as parent block
		if self.source.Number == 0 {
			return types.BlockSeal{}, nil
		} else {
			return types.BlockSeal{},
				fmt.Errorf("cannot find previous derived before start of database: %s (%w)", source, types.ErrPreviousToFirst)
		}
	}
	prev, err := db.readAt(selfIndex - 1)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("cannot find previous derived before %s: %w", source, err)
	}
	return prev.source, nil
}

// NextSource finds the next source after the given source
func (db *DB) NextSource(source eth.BlockID) (types.BlockSeal, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	// Source-entries are unique, doesn't matter if we use the first derived entry or last derived entry.
	selfIndex, self, err := db.sourceNumToLastDerived(source.Number)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("failed to find derived-from %d: %w", source.Number, err)
	}
	if self.source.ID() != source {
		return types.BlockSeal{}, fmt.Errorf("found %s, but expected %s: %w", self.source, source, types.ErrConflict)
	}
	next, err := db.readAt(selfIndex + 1)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("cannot find next derived-from after %s: %w", source, err)
	}
	return next.source, nil
}

// Next returns the next Derived Block Pair after the given pair.
// This may return types.ErrAwaitReplacementBlock if the entry was invalidated and needs replacement.
func (db *DB) Next(pair types.DerivedIDPair) (types.DerivedBlockSealPair, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	selfIndex, selfLink, err := db.lookup(pair.Source.Number, pair.Derived.Number)
	if err != nil {
		return types.DerivedBlockSealPair{}, err
	}
	if selfLink.source.ID() != pair.Source {
		return types.DerivedBlockSealPair{}, fmt.Errorf("DB has derived-from %s but expected %s: %w", selfLink.source, pair.Source, types.ErrConflict)
	}
	if selfLink.derived.ID() != pair.Derived {
		return types.DerivedBlockSealPair{}, fmt.Errorf("DB has derived %s but expected %s: %w", selfLink.derived, pair.Derived, types.ErrConflict)
	}
	next, err := db.readAt(selfIndex + 1)
	if err != nil {
		return types.DerivedBlockSealPair{}, err
	}
	return next.sealOrErr()
}

func (db *DB) derivedNumToFirstSource(derivedNum uint64, revision types.Revision) (entrydb.EntryIdx, LinkEntry, error) {
	// Forward: prioritize the first entry.
	return db.find(false, false, func(link LinkEntry) int {
		res := -revision.Cmp(link.revision.Number())
		if res == 0 {
			return cmp.Compare(link.derived.Number, derivedNum)
		}
		return res
	})
}

func (db *DB) derivedNumToLastSource(derivedNum uint64, revision types.Revision) (entrydb.EntryIdx, LinkEntry, error) {
	// Reverse: prioritize the last entry.
	return db.find(true, false, func(link LinkEntry) int {
		res := revision.Cmp(link.revision.Number())
		if res == 0 {
			return cmp.Compare(derivedNum, link.derived.Number)
		}
		return res
	})
}

func (db *DB) sourceNumToFirstDerived(sourceNum uint64) (entrydb.EntryIdx, LinkEntry, error) {
	// Forward: prioritize the first entry.
	return db.find(false, false, func(link LinkEntry) int {
		return cmp.Compare(link.source.Number, sourceNum)
	})
}

func (db *DB) sourceNumToLastDerived(sourceNum uint64) (entrydb.EntryIdx, LinkEntry, error) {
	// Reverse: prioritize the last entry.
	return db.find(true, false, func(link LinkEntry) int {
		return cmp.Compare(sourceNum, link.source.Number)
	})
}

// lookup returns the *only* entry for which source and derived block numbers match.
// For any given source block a derived block should only show up once.
// There may however be other derived blocks.
func (db *DB) lookup(source, derived uint64) (entrydb.EntryIdx, LinkEntry, error) {
	// Lookup direction does not matter, as we do an exact lookup.
	return db.find(true, false, func(link LinkEntry) int {
		// Source (L1) is the primary key ("major key"):
		// the source number is always strictly incremental in the DB,
		// and can thus safely be used for global search
		res := cmp.Compare(source, link.source.Number)
		if res == 0 {
			// Derived (L2) blocks can re-occur in the DB, after invalidation of local-safe blocks.
			// Within the same source block it is strictly incremental however.
			return cmp.Compare(derived, link.derived.Number)
		}
		return res
	})
}

// lookupOrAfter looks for the (source, derived) pair.
// However, the pair may not be present, the DB may only have derived later blocks from the given source.
// If so, return the first derived entry from that source.
// This is useful when rolling back after invalidating: the local-safe DB may have derived newer data from later sources
// than what the cross-safe DB is deriving.
func (db *DB) lookupOrAfter(source, derived uint64) (entrydb.EntryIdx, LinkEntry, error) {
	// We search left to right, so that if we miss the `derived` target,
	// we end up on the first item that is after it.
	return db.find(false, true, func(link LinkEntry) int {
		// Source (L1) is the primary key ("major key"):
		// the source number is always strictly incremental in the DB,
		// and can thus safely be used for global search
		res := cmp.Compare(link.source.Number, source)
		if res == 0 {
			// Derived (L2) blocks can re-occur in the DB, after invalidation of local-safe blocks.
			// Within the same source block it is strictly incremental however.
			return cmp.Compare(link.derived.Number, derived)
		}
		return res
	})
}

// find finds the first entry for which cmpFn(link) returns 0.
// The cmpFn entries to the left should return -1, entries to the right 1.
// If reverse, the cmpFn should be flipped too, and the last entry for which cmpFn(link) is 0 will be found.
func (db *DB) find(reverse bool, acceptClosest bool, cmpFn func(link LinkEntry) int) (entrydb.EntryIdx, LinkEntry, error) {
	n := db.store.Size()
	if n == 0 {
		return -1, LinkEntry{}, types.ErrFuture
	}
	var searchErr error
	// binary-search for the smallest index i for which cmp(i) >= 0
	// i.e. find the earliest entry that is bigger or equal than the needle.
	result := sort.Search(int(n), func(i int) bool {
		at := entrydb.EntryIdx(i)
		if reverse {
			at = entrydb.EntryIdx(n) - 1 - at
		}
		entry, err := db.readAt(at)
		if err != nil {
			searchErr = err
			return false
		}
		return cmpFn(entry) >= 0
	})
	if searchErr != nil {
		return -1, LinkEntry{}, fmt.Errorf("failed to search: %w", searchErr)
	}
	// If we did not find anything, then we got the length of the input.
	if result == int(n) {
		if reverse {
			// If searching in reverse, then the last entry is the start.
			// I.e. the needle must be before the db start.
			return -1, LinkEntry{}, fmt.Errorf("no entry found: %w", types.ErrSkipped)
		} else {
			// If searing regularly, then the last entry is the end.
			// I.e. the needle must be after the db end.
			return -1, LinkEntry{}, fmt.Errorf("no entry found: %w", types.ErrFuture)
		}
	}
	// If the very first entry matched, then we might be missing prior data.
	firstTry := result == 0
	// Transform back the index, if we were searching in reverse
	if reverse {
		result = int(n) - 1 - result
	}
	// Whatever we found as first entry to be bigger or equal, must be checked for equality.
	// We don't want it if it's bigger, we were searching for the equal-case.
	link, err := db.readAt(entrydb.EntryIdx(result))
	if err != nil {
		return -1, LinkEntry{}, fmt.Errorf("failed to read final result entry %d: %w", result, err)
	}
	if cmpFn(link) != 0 {
		if firstTry { // if the first found entry already is bigger, then we are missing the real data.
			if reverse {
				return -1, LinkEntry{}, fmt.Errorf("query is past last entry %s: %w", link, types.ErrFuture)
			} else {
				return -1, LinkEntry{}, fmt.Errorf("query is before first entry %s: %w", link, types.ErrSkipped)
			}
		} else if !acceptClosest {
			return -1, LinkEntry{}, fmt.Errorf("traversed data, no exact match found, but hit %s: %w", link, types.ErrNotExact)
		}
	}
	return entrydb.EntryIdx(result), link, nil
}

func (db *DB) readAt(i entrydb.EntryIdx) (LinkEntry, error) {
	entry, err := db.store.Read(i)
	if err != nil {
		if err == io.EOF {
			return LinkEntry{}, types.ErrFuture
		}
		return LinkEntry{}, err
	}
	var out LinkEntry
	err = out.decode(entry)
	return out, err
}

func (db *DB) Close() error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	return db.store.Close()
}
