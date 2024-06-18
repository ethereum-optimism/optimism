package db

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func createHash(i int) common.Hash {
	data := bytes.Repeat([]byte{byte(i)}, common.HashLength)
	return common.BytesToHash(data)
}

func TestErrorOpeningDatabase(t *testing.T) {
	dir := t.TempDir()
	_, err := NewFromFile(filepath.Join(dir, "missing-dir", "file.db"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func runDBTest(t *testing.T, setup func(t *testing.T, db *DB), assert func(t *testing.T, db *DB)) {
	createDb := func(t *testing.T, dir string) *DB {
		path := filepath.Join(dir, "test.db")
		db, err := NewFromFile(path)
		require.NoError(t, err, "Failed to create database")
		t.Cleanup(func() {
			err := db.Close()
			if err != nil {
				require.ErrorIs(t, err, fs.ErrClosed)
			}
		})
		return db
	}

	t.Run("New", func(t *testing.T) {
		db := createDb(t, t.TempDir())
		setup(t, db)
		assert(t, db)
	})

	t.Run("Existing", func(t *testing.T) {
		dir := t.TempDir()
		db := createDb(t, dir)
		setup(t, db)
		// Close and recreate the database
		require.NoError(t, db.Close())

		db2 := createDb(t, dir)
		assert(t, db2)
	})
}

func TestEmptyDbDoesNotFindEntry(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB) {},
		func(t *testing.T, db *DB) {
			result, err := db.Contains(0, 0, createHash(1))
			require.NoError(t, err)
			require.False(t, result)

			// Should not contain the empty hash
			result, err = db.Contains(0, 0, common.Hash{})
			require.NoError(t, err)
			require.False(t, result)
		})
}

func TestContainsRecordedLog(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB) {
			err := db.Add(50, []common.Hash{createHash(0), createHash(2), createHash(1)})
			require.NoError(t, err)
		},
		func(t *testing.T, db *DB) {
			actual, err := db.Contains(50, 0, createHash(0))
			require.NoError(t, err)
			require.True(t, actual)

			actual, err = db.Contains(50, 1, createHash(2))
			require.NoError(t, err)
			require.True(t, actual)

			actual, err = db.Contains(50, 2, createHash(1))
			require.NoError(t, err)
			require.True(t, actual)

			actual, err = db.Contains(49, 0, createHash(0))
			require.NoError(t, err)
			require.False(t, actual)

			actual, err = db.Contains(51, 0, createHash(0))
			require.NoError(t, err)
			require.False(t, actual)

			actual, err = db.Contains(50, 3, createHash(3))
			require.NoError(t, err)
			require.False(t, actual)

			// Existing log hash, wrong log index
			actual, err = db.Contains(50, 1, createHash(0))
			require.NoError(t, err)
			require.False(t, actual)
		})
}

func TestErrorWhenAddingBlockOutOfOrder(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB) {
			err := db.Add(5, []common.Hash{createHash(1)})
			require.NoError(t, err)
		},
		func(t *testing.T, db *DB) {
			// Can't add block before head
			err := db.Add(4, []common.Hash{createHash(2)})
			require.ErrorIs(t, err, ErrBlockOutOfOrder)

			// Can't replace head block
			err = db.Add(5, []common.Hash{createHash(1)})
			require.ErrorIs(t, err, ErrBlockOutOfOrder)
		})
}

func TestAddBlockZero(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB) {},
		func(t *testing.T, db *DB) {
			// There are never any logs in the genesis block, so forbid adding a block 0
			err := db.Add(0, []common.Hash{createHash(2)})
			require.ErrorIs(t, err, ErrBlockOutOfOrder)
		})
}

func TestRewind(t *testing.T) {
	t.Run("WhenEmpty", func(t *testing.T) {
		runDBTest(t, func(t *testing.T, db *DB) {},
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Rewind(100))
				require.NoError(t, db.Rewind(0))
			})
	})

	t.Run("AfterLastEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(51, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(74, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(75))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(50, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(50, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)
			})
	})

	t.Run("BeforeFirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(25))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(50, 0, createHash(1))
				require.NoError(t, err)
				require.False(t, contains)

				contains, err = db.Contains(50, 1, createHash(2))
				require.NoError(t, err)
				require.False(t, contains)
			})
	})

	t.Run("AtFirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(51, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(50))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(50, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(50, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(51, 0, createHash(1))
				require.NoError(t, err)
				require.False(t, contains)

				contains, err = db.Contains(51, 1, createHash(2))
				require.NoError(t, err)
				require.False(t, contains)
			})
	})

	t.Run("BetweenEntries", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(55))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(50, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(50, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(60, 0, createHash(1))
				require.NoError(t, err)
				require.False(t, contains)

				contains, err = db.Contains(60, 1, createHash(2))
				require.NoError(t, err)
				require.False(t, contains)
			})
	})

	t.Run("AtExistingEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(59, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(61, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(60))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(59, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(59, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(60, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(60, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(61, 0, createHash(1))
				require.NoError(t, err)
				require.False(t, contains)

				contains, err = db.Contains(61, 1, createHash(2))
				require.NoError(t, err)
				require.False(t, contains)
			})
	})

	t.Run("AtLastEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB) {
				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Add(70, []common.Hash{createHash(1), createHash(2)}))
				require.NoError(t, db.Rewind(70))
			},
			func(t *testing.T, db *DB) {
				contains, err := db.Contains(50, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(50, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(60, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(60, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(70, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, contains)

				contains, err = db.Contains(70, 1, createHash(2))
				require.NoError(t, err)
				require.True(t, contains)
			})
	})

	t.Run("ReaddDeletedBlocks", func(t *testing.T) {
		runDBTest(t, func(t *testing.T, db *DB) {
			require.NoError(t, db.Add(59, []common.Hash{createHash(1), createHash(2)}))
			require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
			require.NoError(t, db.Add(61, []common.Hash{createHash(1), createHash(2)}))
			require.NoError(t, db.Rewind(60))
		},
			func(t *testing.T, db *DB) {
				err := db.Add(59, []common.Hash{createHash(1), createHash(2)})
				require.ErrorIs(t, err, ErrBlockOutOfOrder, "Cannot add block before rewound head")
				err = db.Add(60, []common.Hash{createHash(1), createHash(2)})
				require.ErrorIs(t, err, ErrBlockOutOfOrder, "Cannot add block that was rewound to")
				err = db.Add(61, []common.Hash{createHash(1), createHash(2)})
				require.NoError(t, err, "Can re-add deleted block")
			})
	})
}
