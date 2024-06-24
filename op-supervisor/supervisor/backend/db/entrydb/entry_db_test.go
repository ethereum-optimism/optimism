package entrydb

import (
	"bytes"
	"errors"
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

	t.Run("WriteMultiple", func(t *testing.T) {
		db := createEntryDB(t)
		require.NoError(t, db.Append(createEntry(1), createEntry(2), createEntry(3)))
		require.EqualValues(t, 3, db.Size())
		requireRead(t, db, 0, createEntry(1))
		requireRead(t, db, 1, createEntry(2))
		requireRead(t, db, 2, createEntry(3))
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

func TestWriteErrors(t *testing.T) {
	expectedErr := errors.New("some error")

	t.Run("TruncatePartiallyWrittenData", func(t *testing.T) {
		db, stubData := createEntryDBWithStubData()
		stubData.writeErr = expectedErr
		stubData.writeErrAfterBytes = 3
		err := db.Append(createEntry(1), createEntry(2))
		require.ErrorIs(t, err, expectedErr)

		require.EqualValues(t, 0, db.Size(), "should not consider entries written")
		require.Len(t, stubData.data, 0, "should truncate written bytes")
	})

	t.Run("FailBeforeDataWritten", func(t *testing.T) {
		db, stubData := createEntryDBWithStubData()
		stubData.writeErr = expectedErr
		stubData.writeErrAfterBytes = 0
		err := db.Append(createEntry(1), createEntry(2))
		require.ErrorIs(t, err, expectedErr)

		require.EqualValues(t, 0, db.Size(), "should not consider entries written")
		require.Len(t, stubData.data, 0, "no data written")
	})

	t.Run("PartialWriteAndTruncateFails", func(t *testing.T) {
		db, stubData := createEntryDBWithStubData()
		stubData.writeErr = expectedErr
		stubData.writeErrAfterBytes = EntrySize + 2
		stubData.truncateErr = errors.New("boom")
		err := db.Append(createEntry(1), createEntry(2))
		require.ErrorIs(t, err, expectedErr)

		require.EqualValues(t, 0, db.Size(), "should not consider entries written")
		require.Len(t, stubData.data, stubData.writeErrAfterBytes, "rollback failed")

		_, err = db.Read(0)
		require.ErrorIs(t, err, io.EOF, "should not have first entry")
		_, err = db.Read(1)
		require.ErrorIs(t, err, io.EOF, "should not have second entry")

		// Should retry truncate on next write
		stubData.writeErr = nil
		stubData.truncateErr = nil
		err = db.Append(createEntry(3))
		require.NoError(t, err)
		actual, err := db.Read(0)
		require.NoError(t, err)
		require.Equal(t, createEntry(3), actual)
	})
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

func createEntryDBWithStubData() (*EntryDB, *stubDataAccess) {
	stubData := &stubDataAccess{}
	db := &EntryDB{data: stubData, size: 0}
	return db, stubData
}

type stubDataAccess struct {
	data               []byte
	writeErr           error
	writeErrAfterBytes int
	truncateErr        error
}

func (s *stubDataAccess) ReadAt(p []byte, off int64) (n int, err error) {
	return bytes.NewReader(s.data).ReadAt(p, off)
}

func (s *stubDataAccess) Write(p []byte) (n int, err error) {
	if s.writeErr != nil {
		s.data = append(s.data, p[:s.writeErrAfterBytes]...)
		return s.writeErrAfterBytes, s.writeErr
	}
	s.data = append(s.data, p...)
	return len(p), nil
}

func (s *stubDataAccess) Close() error {
	return nil
}

func (s *stubDataAccess) Truncate(size int64) error {
	if s.truncateErr != nil {
		return s.truncateErr
	}
	s.data = s.data[:size]
	return nil
}

var _ dataAccess = (*stubDataAccess)(nil)
