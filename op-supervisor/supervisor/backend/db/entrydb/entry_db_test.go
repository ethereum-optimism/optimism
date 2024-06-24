package entrydb

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestReadWrite(t *testing.T) {
	t.Run("BasicReadWrite", func(t *testing.T) {
		db := createEntryDB(t)
		require.EqualValues(t, 0, db.Size())
		require.NoError(t, db.Append(createEntry(1)))
		require.EqualValues(t, 1, db.Size())
		require.NoError(t, db.Append(createEntry(2)))
		require.EqualValues(t, 2, db.Size())
		require.NoError(t, db.Append(createEntry(3)))
		require.EqualValues(t, 3, db.Size())
		require.NoError(t, db.Append(createEntry(4)))
		require.EqualValues(t, 4, db.Size())

		requireRead(t, db, 0, createEntry(1))
		requireRead(t, db, 1, createEntry(2))
		requireRead(t, db, 2, createEntry(3))
		requireRead(t, db, 3, createEntry(4))

		// Check we can read out of order
		requireRead(t, db, 1, createEntry(2))
	})

	t.Run("ReadPastEndOfFileReturnsEOF", func(t *testing.T) {
		db := createEntryDB(t)
		_, err := db.Read(0)
		require.ErrorIs(t, err, io.EOF)
	})
}

func TestTruncate(t *testing.T) {
	t.Run("Partial", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(createEntry(1)))
		require.NoError(t, db.Append(createEntry(2)))
		require.NoError(t, db.Append(createEntry(3)))
		require.NoError(t, db.Append(createEntry(4)))
		require.NoError(t, db.Append(createEntry(5)))
		require.EqualValues(t, 5, db.Size())

		require.NoError(t, db.Truncate(3))
		require.EqualValues(t, 4, db.Size()) // 0, 1, 2 and 3 are preserved
		requireRead(t, db, 0, createEntry(1))
		requireRead(t, db, 1, createEntry(2))
		requireRead(t, db, 2, createEntry(3))

		// 4 and 5 have been removed
		_, err := db.Read(4)
		require.ErrorIs(t, err, io.EOF)
		_, err = db.Read(5)
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("Complete", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(createEntry(1)))
		require.NoError(t, db.Append(createEntry(2)))
		require.NoError(t, db.Append(createEntry(3)))
		require.EqualValues(t, 3, db.Size())

		require.NoError(t, db.Truncate(-1))
		require.EqualValues(t, 0, db.Size()) // All items are removed
		_, err := db.Read(0)
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("AppendAfterTruncate", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(createEntry(1)))
		require.NoError(t, db.Append(createEntry(2)))
		require.NoError(t, db.Append(createEntry(3)))
		require.EqualValues(t, 3, db.Size())

		require.NoError(t, db.Truncate(1))
		require.EqualValues(t, 2, db.Size())
		newEntry := createEntry(4)
		require.NoError(t, db.Append(newEntry))
		entry, err := db.Read(2)
		require.NoError(t, err)
		require.Equal(t, newEntry, entry)
	})
}

func TestTruncateTrailingPartialEntries(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	file := filepath.Join(t.TempDir(), "entries.db")
	entry1 := createEntry(1)
	entry2 := createEntry(2)
	invalidData := make([]byte, len(entry1)+len(entry2)+4)
	copy(invalidData, entry1[:])
	copy(invalidData[EntrySize:], entry2[:])
	invalidData[len(invalidData)-1] = 3 // Some invalid trailing data
	require.NoError(t, os.WriteFile(file, invalidData, 0o644))
	db, err := NewEntryDB(logger, file)
	require.NoError(t, err)
	defer db.Close()

	// Should automatically truncate the file to remove trailing partial entries
	require.EqualValues(t, 2, db.Size())
	stat, err := os.Stat(file)
	require.NoError(t, err)
	require.EqualValues(t, 2*EntrySize, stat.Size())
}

func requireRead(t *testing.T, db *EntryDB, idx int64, expected Entry) {
	actual, err := db.Read(idx)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func createEntry(i byte) Entry {
	return Entry(bytes.Repeat([]byte{i}, EntrySize))
}

func createEntryDB(t *testing.T) *EntryDB {
	logger := testlog.Logger(t, log.LvlInfo)
	db, err := NewEntryDB(logger, filepath.Join(t.TempDir(), "entries.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}
