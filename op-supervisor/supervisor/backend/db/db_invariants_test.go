package db

import (
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
