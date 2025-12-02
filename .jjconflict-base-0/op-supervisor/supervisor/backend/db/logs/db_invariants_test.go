package logs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type statInvariant func(stat os.FileInfo, m *stubMetrics) error
type entryInvariant func(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error

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
	entries := make([]Entry, stat.Size()/EntrySize)
	for i := range entries {
		n, err := io.ReadFull(file, entries[i][:])
		require.NoErrorf(t, err, "failed to read entry %v", i)
		require.EqualValuesf(t, EntrySize, n, "read wrong length for entry %v", i)
	}

	entryInvariants := []entryInvariant{
		invariantSearchCheckpointAtEverySearchCheckpointFrequency,
		invariantCanonicalHashOrCheckpointAfterEverySearchCheckpoint,
		invariantSearchCheckpointBeforeEveryCanonicalHash,
		invariantExecChainIDAfterInitEventWithFlagSet,
		invariantExecChainIDOnlyAfterInitiatingEventWithFlagSet,
		invariantExecPositionAfterExecChainID,
		invariantExecPositionOnlyAfterExecChainID,
		invariantExecChecksumAfterExecPosition,
		invariantExecChecksumOnlyAfterExecPosition,
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

func fmtEntries(entries []Entry) string {
	out := ""
	for i, entry := range entries {
		out += fmt.Sprintf("%v: %x\n", i, entry)
	}
	return out
}

func invariantFileSizeMultipleOfEntrySize(stat os.FileInfo, _ *stubMetrics) error {
	size := stat.Size()
	if size%EntrySize != 0 {
		return fmt.Errorf("expected file size to be a multiple of entry size (%v) but was %v", EntrySize, size)
	}
	return nil
}

func invariantFileSizeMatchesEntryCountMetric(stat os.FileInfo, m *stubMetrics) error {
	size := stat.Size()
	if m.entryCount*EntrySize != size {
		return fmt.Errorf("expected file size to be entryCount (%v) * entrySize (%v) = %v but was %v", m.entryCount, EntrySize, m.entryCount*EntrySize, size)
	}
	return nil
}

func invariantSearchCheckpointAtEverySearchCheckpointFrequency(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entryIdx%searchCheckpointFrequency == 0 && entry.Type() != TypeSearchCheckpoint {
		return fmt.Errorf("should have search checkpoints every %v entries but entry %v was %x", searchCheckpointFrequency, entryIdx, entry)
	}
	return nil
}

func invariantCanonicalHashOrCheckpointAfterEverySearchCheckpoint(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeSearchCheckpoint {
		return nil
	}
	if entryIdx+1 >= len(entries) {
		return fmt.Errorf("expected canonical hash or checkpoint after search checkpoint at entry %v but no further entries found", entryIdx)
	}
	nextEntry := entries[entryIdx+1]
	if nextEntry.Type() != TypeCanonicalHash && nextEntry.Type() != TypeSearchCheckpoint {
		return fmt.Errorf("expected canonical hash or checkpoint after search checkpoint at entry %v but got %x", entryIdx, nextEntry)
	}
	return nil
}

// invariantSearchCheckpointBeforeEveryCanonicalHash ensures we don't have extra canonical-hash entries
func invariantSearchCheckpointBeforeEveryCanonicalHash(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeCanonicalHash {
		return nil
	}
	if entryIdx == 0 {
		return fmt.Errorf("expected search checkpoint before canonical hash at entry %v but no previous entries present", entryIdx)
	}
	prevEntry := entries[entryIdx-1]
	if prevEntry.Type() != TypeSearchCheckpoint {
		return fmt.Errorf("expected search checkpoint before canonical hash at entry %v but got %x", entryIdx, prevEntry)
	}
	return nil
}

func invariantExecChainIDAfterInitEventWithFlagSet(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeInitiatingEvent {
		return nil
	}
	hasExecMessage := entry[1]&eventFlagHasExecutingMessage != 0
	if !hasExecMessage {
		return nil
	}
	nextIdx := entryIdx + 1
	if nextIdx%searchCheckpointFrequency == 0 {
		nextIdx += 2 // Skip over the search checkpoint and canonical hash events
	}
	if len(entries) <= nextIdx {
		return fmt.Errorf("expected execChainID after initiating event with exec msg flag set at entry %v but there were no more events", entryIdx)
	}
	if entries[nextIdx].Type() != TypeExecChainID {
		return fmt.Errorf("expected execChainID at idx %v after initiating event with exec msg flag set at entry %v but got type %v", nextIdx, entryIdx, entries[nextIdx][0])
	}
	return nil
}

func invariantExecChainIDOnlyAfterInitiatingEventWithFlagSet(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeExecChainID {
		return nil
	}
	if entryIdx == 0 {
		return errors.New("found execChainID as first entry")
	}
	initIdx := entryIdx - 1
	if initIdx%searchCheckpointFrequency == 1 {
		initIdx -= 2 // Skip the canonical hash and search checkpoint entries
	}
	if initIdx < 0 {
		return fmt.Errorf("found execChainID without a preceding initiating event at entry %v", entryIdx)
	}
	initEntry := entries[initIdx]
	if initEntry.Type() != TypeInitiatingEvent {
		return fmt.Errorf("expected initiating event at entry %v prior to execChainID at %v but got %x", initIdx, entryIdx, initEntry[0])
	}
	flags := initEntry[1]
	if flags&eventFlagHasExecutingMessage == 0 {
		return fmt.Errorf("initiating event at %v prior to execChainID at %v does not have flag set to indicate needing a executing event: %x", initIdx, entryIdx, initEntry)
	}
	return nil
}

func invariantExecPositionAfterExecChainID(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeExecChainID {
		return nil
	}
	prevIdx := entryIdx + 1
	if prevIdx%searchCheckpointFrequency == 0 {
		prevIdx += 2 // Skip the search checkpoint and canonical hash entries
	}
	if prevIdx >= len(entries) {
		return fmt.Errorf("expected execChainID at %v to be followed by execPosition at %v but ran out of entries", entryIdx, prevIdx)
	}
	prevEntry := entries[prevIdx]
	if prevEntry.Type() != TypeExecPosition {
		return fmt.Errorf("expected execChainID at %v to be followed by execPosition at %v but got type %v", entryIdx, prevIdx, prevEntry[0])
	}
	return nil
}

func invariantExecPositionOnlyAfterExecChainID(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeExecPosition {
		return nil
	}
	if entryIdx == 0 {
		return errors.New("found execPosition as first entry")
	}
	prevIdx := entryIdx - 1
	if prevIdx%searchCheckpointFrequency == 1 {
		prevIdx -= 2 // Skip the canonical hash and search checkpoint entries
	}
	if prevIdx < 0 {
		return fmt.Errorf("found execPosition without a preceding execChainID at entry %v", entryIdx)
	}
	prevEntry := entries[prevIdx]
	if prevEntry.Type() != TypeExecChainID {
		return fmt.Errorf("expected execChainID at entry %v prior to execPosition at %v but got %x", prevIdx, entryIdx, prevEntry[0])
	}
	return nil
}

func invariantExecChecksumAfterExecPosition(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeExecPosition {
		return nil
	}
	nextIdx := entryIdx + 1
	if nextIdx%searchCheckpointFrequency == 0 {
		nextIdx += 2 // Skip the search checkpoint and canonical hash entries
	}
	if nextIdx >= len(entries) {
		return fmt.Errorf("expected execPosition at %v to be followed by execChecksum at %v but ran out of entries", entryIdx, nextIdx)
	}
	nextEntry := entries[nextIdx]
	if nextEntry.Type() != TypeExecChecksum {
		return fmt.Errorf("expected execPosition at %v to be followed by execChecksum at %v but got type %v", entryIdx, nextIdx, nextEntry[0])
	}
	return nil
}

func invariantExecChecksumOnlyAfterExecPosition(entryIdx int, entry Entry, entries []Entry, m *stubMetrics) error {
	if entry.Type() != TypeExecChecksum {
		return nil
	}
	if entryIdx == 0 {
		return errors.New("found execChecksum as first entry")
	}
	prevIdx := entryIdx - 1
	if prevIdx%searchCheckpointFrequency == 1 {
		prevIdx -= 2 // Skip the canonical hash and search checkpoint entries
	}
	if prevIdx < 0 {
		return fmt.Errorf("found execChecksum without a preceding execPosition at entry %v", entryIdx)
	}
	prevEntry := entries[prevIdx]
	if prevEntry.Type() != TypeExecPosition {
		return fmt.Errorf("expected execPosition at entry %v prior to execChecksum at %v but got %x", prevIdx, entryIdx, prevEntry[0])
	}
	return nil
}
