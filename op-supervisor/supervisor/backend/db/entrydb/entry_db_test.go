package entrydb

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWrite(t *testing.T) {
	t.Run("BasicReadWrite", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(createEntry(1)))
		require.NoError(t, db.Append(createEntry(2)))
		require.NoError(t, db.Append(createEntry(3)))
		require.NoError(t, db.Append(createEntry(4)))
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

	t.Run("WriteMultiple", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(
			createEntry(1),
			createEntry(2),
			createEntry(3),
		))
		requireRead(t, db, 0, createEntry(1))
		requireRead(t, db, 1, createEntry(2))
		requireRead(t, db, 2, createEntry(3))
	})
}

func TestTruncate(t *testing.T) {
	db := createEntryDB(t)
	require.NoError(t, db.Append(createEntry(1)))
	require.NoError(t, db.Append(createEntry(2)))
	require.NoError(t, db.Append(createEntry(3)))
	require.NoError(t, db.Append(createEntry(4)))
	require.NoError(t, db.Append(createEntry(5)))

	require.NoError(t, db.Truncate(3))
	requireRead(t, db, 0, createEntry(1))
	requireRead(t, db, 1, createEntry(2))
	requireRead(t, db, 2, createEntry(3))

	// 4 and 5 have been removed
	_, err := db.Read(4)
	require.ErrorIs(t, err, io.EOF)
	_, err = db.Read(5)
	require.ErrorIs(t, err, io.EOF)
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
	db, err := NewEntryDB(filepath.Join(t.TempDir(), "entries.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}
