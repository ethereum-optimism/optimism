package rawdb

import (
	"testing"
)

func TestReadWriteHeadIndex(t *testing.T) {
	indices := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
	}

	db := NewMemoryDatabase()
	for _, height := range indices {
		WriteHeadIndex(db, height)
		got := ReadHeadIndex(db)
		if height != *got {
			t.Fatal("Header height mismatch")
		}
	}
}

func TestReadWriteHeadQueueIndex(t *testing.T) {
	indices := []uint64{
		1,
		1 << 4,
		1 << 5,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, height := range indices {
		WriteHeadQueueIndex(db, height)
		got := ReadHeadQueueIndex(db)
		if height != *got {
			t.Fatal("Header height mismatch")
		}
	}
}
