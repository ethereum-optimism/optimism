package db

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/stretchr/testify/require"
)

type statInvariant func(stat os.FileInfo, m *stubMetrics) error
type entryInvariant func(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error

// checkDBInvariants reads the database log directly and asserts a set of invariants on the data.
func checkDBInvariants(t *testing.T, dbPath string, m *stubMetrics) {
	stat, err := os.Stat(dbPath)
	require.NoError(t, err)

	statInvariants := []statInvariant{
		invariantFileSizeMultipleOfEntrySize,
		invariantFileSizeMatchesEntryCountMetric,
	}
	for _, invariant := range statInvariants {
		require.NoError(t, invariant(stat, m))
	}

	// Read all entries as binary blobs
	file, err := os.OpenFile(dbPath, os.O_RDONLY, 0o644)
	require.NoError(t, err)
	entries := make([]entrydb.Entry, stat.Size()/entrydb.EntrySize)
	for i := range entries {
		n, err := io.ReadFull(file, entries[i][:])
		require.NoErrorf(t, err, "failed to read entry %v", i)
		require.EqualValuesf(t, entrydb.EntrySize, n, "read wrong length for entry %v", i)
	}

	entryInvariants := []entryInvariant{
		invariantSearchCheckpointOnlyAtFrequency,
		invariantSearchCheckpointAtEverySearchCheckpointFrequency,
		invariantCanonicalHashAfterEverySearchCheckpoint,
		invariantSearchCheckpointBeforeEveryCanonicalHash,
		invariantIncrementLogIdxIfNotImmediatelyAfterCanonicalHash,
		invariantExecLinkAfterInitEventWithFlagSet,
		invariantExecLinkOnlyAfterInitiatingEventWithFlagSet,
		invariantExecCheckAfterExecLink,
		invariantExecCheckOnlyAfterExecLink,
		invariantValidLastEntry,
	}
	for i, entry := range entries {
		for _, invariant := range entryInvariants {
			err := invariant(i, entry, entries, m)
			if err != nil {
				require.NoErrorf(t, err, "Invariant breached: \n%v", fmtEntries(entries))
			}
		}
	}
}

func fmtEntries(entries []entrydb.Entry) string {
	out := ""
	for i, entry := range entries {
		out += fmt.Sprintf("%v: %x\n", i, entry)
	}
	return out
}

func invariantFileSizeMultipleOfEntrySize(stat os.FileInfo, _ *stubMetrics) error {
	size := stat.Size()
	if size%entrydb.EntrySize != 0 {
		return fmt.Errorf("expected file size to be a multiple of entry size (%v) but was %v", entrydb.EntrySize, size)
	}
	return nil
}

func invariantFileSizeMatchesEntryCountMetric(stat os.FileInfo, m *stubMetrics) error {
	size := stat.Size()
	if m.entryCount*entrydb.EntrySize != size {
		return fmt.Errorf("expected file size to be entryCount (%v) * entrySize (%v) = %v but was %v", m.entryCount, entrydb.EntrySize, m.entryCount*entrydb.EntrySize, size)
	}
	return nil
}

func invariantSearchCheckpointOnlyAtFrequency(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeSearchCheckpoint {
		return nil
	}
	if entryIdx%searchCheckpointFrequency != 0 {
		return fmt.Errorf("should only have search checkpoints every %v entries but found at entry %v", searchCheckpointFrequency, entryIdx)
	}
	return nil
}

func invariantSearchCheckpointAtEverySearchCheckpointFrequency(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entryIdx%searchCheckpointFrequency == 0 && entry[0] != typeSearchCheckpoint {
		return fmt.Errorf("should have search checkpoints every %v entries but entry %v was %x", searchCheckpointFrequency, entryIdx, entry)
	}
	return nil
}

func invariantCanonicalHashAfterEverySearchCheckpoint(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeSearchCheckpoint {
		return nil
	}
	if entryIdx+1 >= len(entries) {
		return fmt.Errorf("expected canonical hash after search checkpoint at entry %v but no further entries found", entryIdx)
	}
	nextEntry := entries[entryIdx+1]
	if nextEntry[0] != typeCanonicalHash {
		return fmt.Errorf("expected canonical hash after search checkpoint at entry %v but got %x", entryIdx, nextEntry)
	}
	return nil
}

// invariantSearchCheckpointBeforeEveryCanonicalHash ensures we don't have extra canonical-hash entries
func invariantSearchCheckpointBeforeEveryCanonicalHash(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeCanonicalHash {
		return nil
	}
	if entryIdx == 0 {
		return fmt.Errorf("expected search checkpoint before canonical hash at entry %v but no previous entries present", entryIdx)
	}
	prevEntry := entries[entryIdx-1]
	if prevEntry[0] != typeSearchCheckpoint {
		return fmt.Errorf("expected search checkpoint before canonical hash at entry %v but got %x", entryIdx, prevEntry)
	}
	return nil
}

func invariantIncrementLogIdxIfNotImmediatelyAfterCanonicalHash(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeInitiatingEvent {
		return nil
	}
	if entryIdx == 0 {
		return fmt.Errorf("found initiating event at index %v before any search checkpoint", entryIdx)
	}
	blockDiff := entry[1]
	flags := entry[2]
	incrementsLogIdx := flags&eventFlagIncrementLogIdx != 0
	prevEntry := entries[entryIdx-1]
	prevEntryIsCanonicalHash := prevEntry[0] == typeCanonicalHash
	if incrementsLogIdx && prevEntryIsCanonicalHash {
		return fmt.Errorf("initiating event at index %v increments logIdx despite being immediately after canonical hash (prev entry %x)", entryIdx, prevEntry)
	}
	if incrementsLogIdx && blockDiff > 0 {
		return fmt.Errorf("initiating event at index %v increments logIdx despite starting a new block", entryIdx)
	}
	if !incrementsLogIdx && !prevEntryIsCanonicalHash && blockDiff == 0 {
		return fmt.Errorf("initiating event at index %v does not increment logIdx when block unchanged and not after canonical hash (prev entry %x)", entryIdx, prevEntry)
	}
	return nil
}

func invariantExecLinkAfterInitEventWithFlagSet(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeInitiatingEvent {
		return nil
	}
	hasExecMessage := entry[2]&eventFlagHasExecutingMessage != 0
	if !hasExecMessage {
		return nil
	}
	linkIdx := entryIdx + 1
	if linkIdx%searchCheckpointFrequency == 0 {
		linkIdx += 2 // Skip over the search checkpoint and canonical hash events
	}
	if len(entries) <= linkIdx {
		return fmt.Errorf("expected executing link after initiating event with exec msg flag set at entry %v but there were no more events", entryIdx)
	}
	if entries[linkIdx][0] != typeExecutingLink {
		return fmt.Errorf("expected executing link at idx %v after initiating event with exec msg flag set at entry %v but got type %v", linkIdx, entryIdx, entries[linkIdx][0])
	}
	return nil
}

func invariantExecLinkOnlyAfterInitiatingEventWithFlagSet(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeExecutingLink {
		return nil
	}
	if entryIdx == 0 {
		return errors.New("found executing link as first entry")
	}
	initIdx := entryIdx - 1
	if initIdx%searchCheckpointFrequency == 1 {
		initIdx -= 2 // Skip the canonical hash and search checkpoint entries
	}
	if initIdx < 0 {
		return fmt.Errorf("found executing link without a preceding initiating event at entry %v", entryIdx)
	}
	initEntry := entries[initIdx]
	if initEntry[0] != typeInitiatingEvent {
		return fmt.Errorf("expected initiating event at entry %v prior to executing link at %v but got %x", initIdx, entryIdx, initEntry[0])
	}
	flags := initEntry[2]
	if flags&eventFlagHasExecutingMessage == 0 {
		return fmt.Errorf("initiating event at %v prior to executing link at %v does not have flag set to indicate needing a executing event: %x", initIdx, entryIdx, initEntry)
	}
	return nil
}

func invariantExecCheckAfterExecLink(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeExecutingLink {
		return nil
	}
	checkIdx := entryIdx + 1
	if checkIdx%searchCheckpointFrequency == 0 {
		checkIdx += 2 // Skip the search checkpoint and canonical hash entries
	}
	if checkIdx >= len(entries) {
		return fmt.Errorf("expected executing link at %v to be followed by executing check at %v but ran out of entries", entryIdx, checkIdx)
	}
	checkEntry := entries[checkIdx]
	if checkEntry[0] != typeExecutingCheck {
		return fmt.Errorf("expected executing link at %v to be followed by executing check at %v but got type %v", entryIdx, checkIdx, checkEntry[0])
	}
	return nil
}

func invariantExecCheckOnlyAfterExecLink(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entry[0] != typeExecutingCheck {
		return nil
	}
	if entryIdx == 0 {
		return errors.New("found executing check as first entry")
	}
	linkIdx := entryIdx - 1
	if linkIdx%searchCheckpointFrequency == 1 {
		linkIdx -= 2 // Skip the canonical hash and search checkpoint entries
	}
	if linkIdx < 0 {
		return fmt.Errorf("found executing link without a preceding initiating event at entry %v", entryIdx)
	}
	linkEntry := entries[linkIdx]
	if linkEntry[0] != typeExecutingLink {
		return fmt.Errorf("expected executing link at entry %v prior to executing check at %v but got %x", linkIdx, entryIdx, linkEntry[0])
	}
	return nil
}

// invariantValidLastEntry checks that the last entry is either a executing check or initiating event with no exec message
func invariantValidLastEntry(entryIdx int, entry entrydb.Entry, entries []entrydb.Entry, m *stubMetrics) error {
	if entryIdx+1 < len(entries) {
		return nil
	}
	if entry[0] == typeExecutingCheck {
		return nil
	}
	if entry[0] != typeInitiatingEvent {
		return fmt.Errorf("invalid final event type: %v", entry[0])
	}
	evt, err := newInitiatingEventFromEntry(entry)
	if err != nil {
		return fmt.Errorf("final event was invalid: %w", err)
	}
	if evt.hasExecMsg {
		return errors.New("ends with init event that should have exec msg but no exec msg follows")
	}
	return nil
}
