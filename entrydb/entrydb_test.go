package entrydb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

var testEntryWidth = 32
var testCheckpointFreq = 100
var testEntryType1 = byte(0)
var testEntryType2 = byte(1)

func EntryDBForTesting(t *testing.T) EntryDB {
	typeMap := make(map[byte]Entry)
	typeMap[testEntryType1] = &MockEntry{}
	typeMap[testEntryType2] = &MockEntry2{}

	dbdir := t.TempDir()
	dbFile, err := os.Create(path.Join(dbdir, "test.db"))
	require.NoError(t, err)
	return NewEntryDB(
		typeMap,
		testEntryWidth,
		testCheckpointFreq,
		dbFile,
	)
}

func TestEntryDBRead(t *testing.T) {
	t.Run("single item in db", func(t *testing.T) {
		db := EntryDBForTesting(t)
		entry := &MockEntry{
			ID:   1,
			Name: "Alice",
			Age:  30 + i,
		}
		err := db.Put(entry)
		require.NoError(t, err)

		// get the entry
		ret, err := db.Get(entry.SequenceValue(), &MockEntry{})
		require.NoError(t, err)
		require.Equal(t, entry, ret)
	})
	t.Run("many items in db", func(t *testing.T) {
		db := EntryDBForTesting(t)

		entries := make([]*MockEntry, 100)
		for i := 0; i < 100; i++ {
			entries[i] = &MockEntry{
				ID:   uint64(i),
				Name: "Alice",
				Age:  30,
			}
		}

		// put all the entries
		for _, entry := range entries {
			err := db.Put(entry)
			require.NoError(t, err)
		}

		// get all the entries
		for _, entry := range entries {
			fmt.Println("looking for", entry.SequenceValue())
			ret, err := db.Get(entry.SequenceValue(), &MockEntry{})
			require.NoError(t, err)
			require.Equal(t, entry, ret)
		}
	})
}
func TestEntryDBWrite(t *testing.T) {
	t.Run("write single entry", func(t *testing.T) {
		db := EntryDBForTesting(t)
		entry := &MockEntry{
			ID:   1,
			Name: "Alice",
			Age:  30,
		}
		err := db.Put(entry)
		require.NoError(t, err)
		header, data, err := entry.Encode()
		require.NoError(t, err)
		// there's at least the header in the database
		expectedLen := 1
		// if there's data, there's more frames
		if len(data) > 0 {
			if len(data)%testEntryWidth == 0 {
				expectedLen += len(data) / testEntryWidth
			} else {
				// account for the last frame that's not full
				expectedLen += len(data)/testEntryWidth + 1
			}
		}
		// check the length of the database
		require.Equal(t, expectedLen, db.(*entryDB).length)
		// check the last written sequence value
		expectedSeq := header.ID
		require.Equal(t, expectedSeq, db.(*entryDB).lastWritten)
	})
	t.Run("write incrementing entries", func(t *testing.T) {
		db := EntryDBForTesting(t)
		num := 1
		expectedLen := 0
		var header EntryHeader
		for i := 0; i < num; i++ {
			entry := &MockEntry{
				ID:   uint64(i),
				Name: "Alice",
				Age:  30,
			}
			err := db.Put(entry)
			require.NoError(t, err)
			// there's at least the header in the database
			expectedLen += 1
			h, data, err := entry.Encode()
			header = h
			require.NoError(t, err)
			// if there's data, there's more frames
			if len(data) > 0 {
				if len(data)%testEntryWidth == 0 {
					expectedLen += len(data) / testEntryWidth
				} else {
					// account for the last frame that's not full
					expectedLen += len(data)/testEntryWidth + 1
				}
			}
		}
		// we checkpointed every testCheckpointFreq writes
		expectedLen += expectedLen / testCheckpointFreq
		// check the length of the database
		require.Equal(t, expectedLen, db.(*entryDB).length)
		// check the last written sequence value
		expectedSeq := header.ID
		require.Equal(t, expectedSeq, db.(*entryDB).lastWritten)
	})
	t.Run("write many entries", func(t *testing.T) {
		db := EntryDBForTesting(t)
		entry := &MockEntry{
			ID:   1,
			Name: "Alice",
			Age:  30,
		}
		num := 100
		for i := 0; i < num; i++ {
			err := db.Put(entry)
			require.NoError(t, err)
		}
		header, data, err := entry.Encode()
		require.NoError(t, err)
		// there's at least the header in the database
		expectedLen := 1
		// if there's data, there's more frames
		if len(data) > 0 {
			if len(data)%testEntryWidth == 0 {
				expectedLen += len(data) / testEntryWidth
			} else {
				// account for the last frame that's not full
				expectedLen += len(data)/testEntryWidth + 1
			}
		}
		// we wrote the data num times
		expectedLen *= num
		// and we checkpointed every testCheckpointFreq writes
		expectedLen += expectedLen / testCheckpointFreq
		// check the length of the database
		require.Equal(t, expectedLen, db.(*entryDB).length)
		// check the last written sequence value
		expectedSeq := header.ID
		require.Equal(t, expectedSeq, db.(*entryDB).lastWritten)
	})
	t.Run("write mixed entries", func(t *testing.T) {
		db := EntryDBForTesting(t)
		entryA := &MockEntry{
			ID:   1,
			Name: "Alice",
			Age:  30,
		}
		entryB := &MockEntry2{
			ID: 1,
		}
		num := 50
		for i := 0; i < num; i++ {
			err := db.Put(entryA)
			require.NoError(t, err)
			err = db.Put(entryB)
			require.NoError(t, err)
		}
		header, data, err := entryA.Encode()
		require.NoError(t, err)
		// there's at least the header
		expectedLen := 1
		// if there's data, there's more frames
		if len(data) > 0 {
			if len(data)%testEntryWidth == 0 {
				expectedLen += len(data) / testEntryWidth
			} else {
				// account for the last frame that's not full
				expectedLen += len(data)/testEntryWidth + 1
			}
		}
		// we wrote the data num times
		expectedLen *= num
		// we also wrote header-only data num times
		expectedLen += num
		// and we checkpointed every testCheckpointFreq writes
		expectedLen += expectedLen / testCheckpointFreq
		// check the length of the database
		require.Equal(t, expectedLen, db.(*entryDB).length)
		// check the last written sequence value
		expectedSeq := header.ID
		require.Equal(t, expectedSeq, db.(*entryDB).lastWritten)
	})
}

type MockEntry struct {
	ID   uint64
	Name string
	Age  uint64
}

func (e *MockEntry) SequenceValue() SequenceValue {
	ret := SequenceValue{}
	ret[0] = byte(e.ID)
	return ret
}

func (e *MockEntry) Type() byte {
	return testEntryType1
}

func (e *MockEntry) Encode() (EntryHeader, []byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(e)
	if err != nil {
		return EntryHeader{}, nil, err
	}
	header := EntryHeader{
		Type:       testEntryType1,
		FrameCount: byte(buf.Len() / testEntryWidth),
		ID:         e.SequenceValue(),
	}
	return header, buf.Bytes(), nil
}

func (e *MockEntry) Decode(_ EntryHeader, data []byte) (Entry, error) {
	var entry MockEntry
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

type MockEntry2 struct {
	ID byte
}

func (e *MockEntry2) Type() byte {
	return testEntryType2
}

func (e *MockEntry2) SequenceValue() SequenceValue {
	ret := SequenceValue{}
	ret[0] = e.ID
	return ret
}

func (e *MockEntry2) Encode() (EntryHeader, []byte, error) {
	header := EntryHeader{
		Type:       testEntryType1,
		FrameCount: 0,
		ID:         e.SequenceValue(),
	}
	return header, nil, nil
}

func (e *MockEntry2) Decode(header EntryHeader, _ []byte) (Entry, error) {
	return &MockEntry2{
		ID: byte(header.ID[0]),
	}, nil
}
