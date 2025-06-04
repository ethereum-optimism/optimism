package fromda

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var errInvalidateMismatch = fmt.Errorf("cannot invalidate mismatching block")

func (db *DB) IsEmpty() bool {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	return db.store.Size() == 0
}

func (db *DB) AddDerived(source eth.BlockRef, derived eth.BlockRef, revision types.Revision) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	return db.addLink(source, derived, common.Hash{}, revision)
}

// ReplaceInvalidatedBlock replaces the current Invalidated block with the given replacement.
// The to-be invalidated hash must be provided for consistency checks.
func (db *DB) ReplaceInvalidatedBlock(inv reads.Invalidator, replacementDerived eth.BlockRef, invalidated common.Hash) (
	out types.DerivedBlockRefPair, err error,
) {
	release, err := inv.TryInvalidate(reads.InvalidationRules{
		reads.DerivedInvalidation{Timestamp: replacementDerived.Time},
		// source block stays the same, so nothing to invalidate there.
	})
	if err != nil {
		return types.DerivedBlockRefPair{}, err
	}
	defer release()
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	db.log.Warn("Replacing invalidated block", "replacement", replacementDerived, "invalidated", invalidated)

	// We take the last occurrence. This is where it started to be considered invalid,
	// and where we thus stopped building additional entries for it.
	lastIndex := db.store.LastEntryIdx()
	if lastIndex < 0 {
		return types.DerivedBlockRefPair{}, types.ErrFuture
	}
	last, err := db.readAt(lastIndex)
	if err != nil {
		return types.DerivedBlockRefPair{}, fmt.Errorf("failed to read last derivation data: %w", err)
	}
	if !last.invalidated {
		return types.DerivedBlockRefPair{}, fmt.Errorf("cannot replace block %d, that was not invalidated, with block %s: %w", last.derived, replacementDerived, types.ErrConflict)
	}
	if last.derived.Hash != invalidated {
		return types.DerivedBlockRefPair{}, fmt.Errorf("cannot replace invalidated %s, DB contains %s: %w", invalidated, last.derived, types.ErrConflict)
	}
	// Find the parent-block of derived-from.
	// We need this to build a block-ref, so the DB can be consistency-checked when the next entry is added.
	// There is always one, since the first entry in the DB should never be an invalidated one.
	prevSource, err := db.previousSource(last.source.ID())
	if err != nil {
		return types.DerivedBlockRefPair{}, err
	}
	// Remove the invalidated placeholder and everything after
	err = db.store.Truncate(lastIndex - 1)
	if err != nil {
		return types.DerivedBlockRefPair{}, err
	}
	replacement := types.DerivedBlockRefPair{
		Source:  last.source.ForceWithParent(prevSource.ID()),
		Derived: replacementDerived,
	}
	// Insert the replacement
	if err := db.addLink(replacement.Source, replacement.Derived, invalidated, last.revision); err != nil {
		return types.DerivedBlockRefPair{}, fmt.Errorf("failed to add %s as replacement at %s: %w", replacement.Derived, replacement.Source, err)
	}
	return replacement, nil
}

// RewindAndInvalidate rolls back the database to just before the invalidated block,
// and then marks the block as invalidated, so that no new data can be added to the DB
// until a Rewind or ReplaceInvalidatedBlock.
func (db *DB) RewindAndInvalidate(inv reads.Invalidator, invalidated types.DerivedBlockRefPair) error {
	release, err := inv.TryInvalidate(reads.InvalidationRules{
		reads.SourceInvalidation{Number: invalidated.Source.Number},
		reads.DerivedInvalidation{Timestamp: invalidated.Derived.Time},
	})
	if err != nil {
		return err
	}
	defer release()
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	t := types.DerivedBlockSealPair{
		Source:  types.BlockSealFromRef(invalidated.Source),
		Derived: types.BlockSealFromRef(invalidated.Derived),
	}
	i, link, err := db.lookupOrAfter(t.Source.Number, t.Derived.Number)
	if err != nil {
		return err
	}
	if link.invalidated {
		return fmt.Errorf("cannot invalidate already invalidated block %s with %s: %w", link, invalidated, types.ErrAwaitReplacementBlock)
	}
	// We must have an exact match for the source, this is where and when we decide to invalidate
	if link.source.Hash != t.Source.Hash {
		return fmt.Errorf("found derived-from %s, but expected %s: %w",
			link.source, t.Source, types.ErrConflict)
	}
	// If we optimistically derived some block already for a previous source,
	// and only invalidated because of a later view of newer source data,
	// then we may have derived later blocks and will not exactly match.
	if link.derived.Hash != t.Derived.Hash {
		if link.derived.Number <= t.Derived.Number {
			return fmt.Errorf("found derived %s, but expected %s: %w",
				link.derived, t.Derived, types.ErrConflict)
		} else {
			db.log.Warn("Invalidating block that was previously optimistically assumed as canonical with previous source",
				"invalidated_source", invalidated.Source, "invalidated_derived", invalidated.Derived,
				"link_source", link.source, "link_derived", link.derived)
		}
	}
	// we rewind to exclude the entry we found
	target := i - 1
	if err := db.store.Truncate(target); err != nil {
		return fmt.Errorf("failed to rewind upon block invalidation of %s: %w", t, err)
	}
	db.m.RecordDBDerivedEntryCount(int64(target) + 1)

	// Starting with the placeholder invalidated entry, we are building a new canonical chain.
	// The block-number of the invalidated derived entry is used as revision number,
	// so that every DB copy that invalidates here uses the same number.
	revision := types.Revision(invalidated.Derived.Number)
	if err := db.addLink(invalidated.Source, invalidated.Derived, invalidated.Derived.Hash, revision); err != nil {
		return fmt.Errorf("failed to add invalidation entry %s: %w", invalidated, err)
	}
	return nil
}

// Rewind rolls back the database to the target, including the target if the including flag is set.
// it locks the DB and calls rewindLocked.
func (db *DB) Rewind(inv reads.Invalidator, target types.DerivedBlockSealPair, including bool) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	return db.rewindLocked(inv, target, including)
}

// RewindToScope rewinds the DB to the last entry with
// a source value matching the given scope (inclusive, scope is retained in DB).
// Note that this drop L1 blocks that resulted in a previously invalidated local-safe block.
// This returns ErrFuture if the block is newer than the last known block.
// This returns ErrConflict if a different block at the given height is known.
// TODO: rename this "RewindToSource" to match the idea of Source
func (db *DB) RewindToScope(inv reads.Invalidator, scope eth.BlockID) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	_, link, err := db.sourceNumToLastDerived(scope.Number)
	if err != nil {
		return fmt.Errorf("failed to find last derived %d: %w", scope.Number, err)
	}
	if link.source.ID() != scope {
		return fmt.Errorf("found derived-from %s but expected %s: %w", link.source, scope, types.ErrConflict)
	}
	return db.rewindLocked(inv, types.DerivedBlockSealPair{
		Source:  link.source,
		Derived: link.derived,
	}, false)
}

// RewindToFirstDerived rewinds to the first time
// when v was derived (inclusive, v is retained in DB).
func (db *DB) RewindToFirstDerived(inv reads.Invalidator, v eth.BlockID, revision types.Revision) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	_, link, err := db.derivedNumToFirstSource(v.Number, revision)
	if err != nil {
		return fmt.Errorf("failed to find when %d was first derived: %w", v.Number, err)
	}
	if link.derived.ID() != v {
		return fmt.Errorf("found derived %s but expected %s: %w", link.derived, v, types.ErrConflict)
	}
	return db.rewindLocked(inv, types.DerivedBlockSealPair{
		Source:  link.source,
		Derived: link.derived,
	}, false)
}

// rewindLocked performs the truncate operation to a specified block seal pair.
// data beyond the specified block seal pair is truncated from the database.
// If including is true, the block seal pair itself is removed as well.
// Note: This function must be called with the rwLock held.
// Callers are responsible for locking and unlocking the Database.
func (db *DB) rewindLocked(inv reads.Invalidator, t types.DerivedBlockSealPair, including bool) error {
	release, err := inv.TryInvalidate(reads.InvalidationRules{
		reads.SourceInvalidation{Number: t.Source.Number}, // TODO including bool
		reads.DerivedInvalidation{Timestamp: t.Derived.Timestamp},
	})
	if err != nil {
		return err
	}
	defer release()
	i, link, err := db.lookup(t.Source.Number, t.Derived.Number)
	if err != nil {
		return err
	}
	if link.source.Hash != t.Source.Hash {
		return fmt.Errorf("found derived-from %s, but expected %s: %w",
			link.source, t.Source, types.ErrConflict)
	}
	if link.derived.Hash != t.Derived.Hash {
		return fmt.Errorf("found derived %s, but expected %s: %w",
			link.derived, t.Derived, types.ErrConflict)
	}
	// adjust the target index to include the block seal pair itself if requested
	target := i
	if including {
		target = i - 1
	}
	if err := db.store.Truncate(target); err != nil {
		return fmt.Errorf("failed to rewind upon block invalidation of %s: %w", t, err)
	}
	db.m.RecordDBDerivedEntryCount(int64(target) + 1)
	return nil
}

// addLink adds a L1/L2 derivation link, with strong consistency checks.
// if the link invalidates a prior L2 block, that was valid in a prior L1,
// the invalidated hash needs to match it, even if a new derived block replaces it.
// If types.RevisionAny is provided, the last registered revision is repeated.
func (db *DB) addLink(source eth.BlockRef, derived eth.BlockRef, invalidated common.Hash, revision types.Revision) error {
	// - we are in regular operation if (invalidated = 0)
	// - we are invalidating if (invalidated != 0 && derived == invalidated)
	// - we are replacing an invalidated entry if (invalidated != 0 && derived != invalidated)
	link := LinkEntry{
		source: types.BlockSeal{
			Hash:      source.Hash,
			Number:    source.Number,
			Timestamp: source.Time,
		},
		derived: types.BlockSeal{
			Hash:      derived.Hash,
			Number:    derived.Number,
			Timestamp: derived.Time,
		},
		invalidated: (invalidated != common.Hash{}) && derived.Hash == invalidated,
		revision:    revision,
	}
	// If we don't have any entries yet, allow any block to start things off
	if db.store.Size() == 0 {
		if link.invalidated {
			return fmt.Errorf("first DB entry %s cannot be an invalidated entry: %w", link, types.ErrConflict)
		}
		if revision.Any() {
			link.revision = FirstRevision
		}
		e := link.encode()
		if err := db.store.Append(e); err != nil {
			return err
		}
		db.log.Debug("First entry in DB", "entry", link)
		db.m.RecordDBDerivedEntryCount(db.store.Size())
		return nil
	}

	last, err := db.latest()
	if err != nil {
		return err
	}
	if last.invalidated {
		return fmt.Errorf("cannot build %s on top of invalidated entry %s: %w", link, last, types.ErrAwaitReplacementBlock)
	}
	lastSource := last.source
	lastDerived := last.derived

	if revision.Any() {
		link.revision = last.revision
	} else {
		if invalidated == (common.Hash{}) {
			// Cross-safe db can bump the revision number without invalidating anything at the same time.
			// But can only do so when inserting the expected entry
			if last.revision != revision {
				if derived.Number != revision.Number() {
					return fmt.Errorf("cannot insert entry %s with revision %s, after %s with revision %s: %w",
						derived, revision, last.derived, last.revision, types.ErrDataCorruption)
				}
				db.log.Warn("New revision", "revision", revision, "source", source, "derived", derived)
			}
		} else if derived.Hash == invalidated {
			if last.revision == revision {
				return fmt.Errorf("invalidated link %s should change revision: %w", link, types.ErrConflict)
			}
			if derived.Number != revision.Number() {
				return fmt.Errorf("expecting invalidated/replaced entry %s to match revision %s: %w", derived, revision, types.ErrDataCorruption)
			}
			db.log.Debug("Invalidating entry", "revision", revision, "source", source, "derived", derived)
		}
	}

	if (lastSource.Number+1 == source.Number) && (lastDerived.Number+1 == derived.Number) {
		return fmt.Errorf("cannot add source:%s derived:%s on top of last entry source:%s derived:%s, must increment source or derived, not both: %w",
			source, derived, last.source, last.derived, types.ErrOutOfOrder)
	}

	if lastDerived.ID() == derived.ID() && lastSource.ID() == source.ID() {
		// it shouldn't be possible, but the ID component of a block ref doesn't include the timestamp
		// so if the timestamp doesn't match, still return no error to the caller, but at least log a warning
		if lastDerived.Timestamp != derived.Time {
			db.log.Warn("Derived block already exists with different timestamp", "derived", derived, "lastDerived", lastDerived)
		}
		if lastSource.Timestamp != source.Time {
			db.log.Warn("Derived-from block already exists with different timestamp", "source", source, "lastSource", lastSource)
		}
		// Repeat of same information. No entries to be written.
		// But we can silently ignore and not return an error, as that brings the caller
		// in a consistent state, after which it can insert the actual new derived-from information.
		db.log.Debug("Database link already written", "derived", derived, "lastDerived", lastDerived, "revision", last.revision)
		return nil
	}

	// Check derived relation: the L2 chain has to be sequential without gaps. An L2 block may repeat if the L1 block is empty.
	if lastDerived.Number == derived.Number {
		// Same block height? Then it must be the same block.
		// I.e. we encountered an empty L1 block, and the same L2 block continues to be the last block that was derived from it.
		if invalidated != (common.Hash{}) {
			if lastDerived.Hash != invalidated {
				return fmt.Errorf("inserting block %s that invalidates %s at height %d, but expected %s: %w", derived.Hash, invalidated, lastDerived.Number, lastDerived.Hash, types.ErrConflict)
			}
		} else {
			if lastDerived.Hash != derived.Hash {
				return fmt.Errorf("derived block %s conflicts with known derived block %s at same height: %w",
					derived, lastDerived, types.ErrConflict)
			}
		}
	} else if lastDerived.Number+1 == derived.Number {
		if lastDerived.Hash != derived.ParentHash {
			return fmt.Errorf("derived block %s (parent %s) does not build on %s: %w",
				derived, derived.ParentHash, lastDerived, types.ErrConflict)
		}
	} else if lastDerived.Number+1 < derived.Number {
		return fmt.Errorf("cannot add block (%s derived from %s), last block (%s derived from %s) is too far behind: (%w)",
			derived, source,
			lastDerived, lastSource,
			types.ErrFuture)
	} else {
		if invalidated != (common.Hash{}) {
			// Invalidated blocks or replacement blocks may be reverting back to an older derived block.
			// If it is older, let's sanity-check it's a known block that is being invalidated or replaced.
			if _, v, err := db.derivedNumToLastSource(derived.Number, last.revision); err != nil {
				return fmt.Errorf("failed to check if older invalidated derived block was known: %w", err)
			} else if v.derived.Hash != invalidated {
				return fmt.Errorf("expected %s but invalidating %s: %w", link.derived, invalidated, errInvalidateMismatch)
			}
		} else {
			return fmt.Errorf("derived block %s is older than current derived block %s: %w",
				derived, lastDerived, types.ErrOutOfOrder)
		}
	}

	// Check derived-from relation: multiple L2 blocks may be derived from the same L1 block. But everything in sequence.
	if lastSource.Number == source.Number {
		// Same block height? Then it must be the same block.
		if lastSource.Hash != source.Hash {
			return fmt.Errorf("cannot add block %s as derived from %s, expected to be derived from %s at this block height: %w",
				derived, source, lastSource, types.ErrConflict)
		}
	} else if lastSource.Number+1 == source.Number {
		// parent hash check
		if lastSource.Hash != source.ParentHash {
			return fmt.Errorf("cannot add block %s as derived from %s (parent %s) derived on top of %s: %w",
				derived, source, source.ParentHash, lastSource, types.ErrConflict)
		}
	} else if lastSource.Number+1 < source.Number {
		// adding block that is derived from something too far into the future
		return fmt.Errorf("cannot add block (%s derived from %s), last block (%s derived from %s) is too far behind: (%w)",
			derived, source,
			lastDerived, lastSource,
			types.ErrFuture)
	} else {
		if lastDerived.Hash == derived.Hash {
			// we might see L1 blocks repeat,
			// if the deriver has reset to the latest local-safe block,
			// since we don't reset it to any particular source block.
			// So check if it's canonical, and if it is, we can gracefully accept it, to allow forwards progress.
			_, got, err := db.lookup(source.Number, derived.Number)
			if err != nil {
				return fmt.Errorf("failed to check if block %s with old source %s was derived from canonical source chain: %w",
					derived, source, err)
			}
			if got.source.Hash != source.Hash {
				return fmt.Errorf("cannot add block %s that matches latest derived since it is derived from non-canonical source %s, expected %s: %w",
					derived, source, got.source, types.ErrConflict)
			}
			return fmt.Errorf("received latest block %s, derived from known old source %s, latest source is %s: %w",
				derived, source, lastSource, types.ErrIneffective)
		}
		// Adding a newer block that is derived from an older source, that cannot be right
		return fmt.Errorf("cannot add block %s as derived from %s, deriving already at %s: %w",
			derived, source, lastSource, types.ErrOutOfOrder)
	}

	e := link.encode()
	if err := db.store.Append(e); err != nil {
		return err
	}
	db.m.RecordDBDerivedEntryCount(db.store.Size())
	return nil
}
