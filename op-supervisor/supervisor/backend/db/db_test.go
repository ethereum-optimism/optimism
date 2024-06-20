package db

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func createHash(i int) common.Hash {
	data := bytes.Repeat([]byte{byte(i)}, common.HashLength)
	return common.BytesToHash(data)
}

func TestErrorOpeningDatabase(t *testing.T) {
	dir := t.TempDir()
	_, err := NewFromFile(testlog.Logger(t, log.LvlInfo), &stubMetrics{}, filepath.Join(dir, "missing-dir", "file.db"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func runDBTest(t *testing.T, setup func(t *testing.T, db *DB, m *stubMetrics), assert func(t *testing.T, db *DB, m *stubMetrics)) {
	createDb := func(t *testing.T, dir string) (*DB, *stubMetrics) {
		logger := testlog.Logger(t, log.LvlInfo)
		path := filepath.Join(dir, "test.db")
		m := &stubMetrics{}
		db, err := NewFromFile(logger, m, path)
		require.NoError(t, err, "Failed to create database")
		t.Cleanup(func() {
			err := db.Close()
			if err != nil {
				require.ErrorIs(t, err, fs.ErrClosed)
			}
		})
		return db, m
	}

	t.Run("New", func(t *testing.T) {
		db, m := createDb(t, t.TempDir())
		setup(t, db, m)
		assert(t, db, m)
	})

	t.Run("Existing", func(t *testing.T) {
		dir := t.TempDir()
		db, m := createDb(t, dir)
		setup(t, db, m)
		// Close and recreate the database
		require.NoError(t, db.Close())

		db2, m := createDb(t, dir)
		assert(t, db2, m)
	})
}

func TestEmptyDbDoesNotFindEntry(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {},
		func(t *testing.T, db *DB, m *stubMetrics) {
			result, err := db.Contains(0, 0, createHash(1))
			require.NoError(t, err)
			require.False(t, result)

			// Should not contain the empty hash
			result, err = db.Contains(0, 0, common.Hash{})
			require.NoError(t, err)
			require.False(t, result)
		})
}

//func TestContainsRecordedLog(t *testing.T) {
//	t.Skip("TODO")
//	runDBTest(t,
//		func(t *testing.T, db *DB) {
//			err := db.Add(50, []common.Hash{createHash(0), createHash(2), createHash(1)})
//			require.NoError(t, err)
//		},
//		func(t *testing.T, db *DB) {
//			actual, err := db.Contains(50, 0, createHash(0))
//			require.NoError(t, err)
//			require.True(t, actual)
//
//			actual, err = db.Contains(50, 1, createHash(2))
//			require.NoError(t, err)
//			require.True(t, actual)
//
//			actual, err = db.Contains(50, 2, createHash(1))
//			require.NoError(t, err)
//			require.True(t, actual)
//
//			actual, err = db.Contains(49, 0, createHash(0))
//			require.NoError(t, err)
//			require.False(t, actual)
//
//			actual, err = db.Contains(51, 0, createHash(0))
//			require.NoError(t, err)
//			require.False(t, actual)
//
//			actual, err = db.Contains(50, 3, createHash(3))
//			require.NoError(t, err)
//			require.False(t, actual)
//
//			// Existing log hash, wrong log index
//			actual, err = db.Contains(50, 1, createHash(0))
//			require.NoError(t, err)
//			require.False(t, actual)
//		})
//}

func TestAddLog(t *testing.T) {
	t.Run("BlockZero", func(t *testing.T) {
		// There are no logs in the genesis block so recording an entry for block 0 should be rejected.
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 0}, 5000, 0)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("FirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				result, err := db.Contains(15, 0, createHash(1))
				require.NoError(t, err)
				require.True(t, result)
			})
	})

	t.Run("MultipleEntriesFromSameBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1)
				require.NoError(t, err)
				err = db.AddLog(createHash(3), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.EqualValues(t, 5, m.entryCount, "should not output new searchCheckpoint for every log")
				requireContains(t, db, 15, 0, createHash(1))
				requireContains(t, db, 15, 1, createHash(2))
				requireContains(t, db, 15, 2, createHash(3))
			})
	})

	t.Run("MultipleEntriesFromMultipleBlocks", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1)
				require.NoError(t, err)
				err = db.AddLog(createHash(3), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 0)
				require.NoError(t, err)
				err = db.AddLog(createHash(4), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 1)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.EqualValues(t, 6, m.entryCount, "should not output new searchCheckpoint for every block")
				requireContains(t, db, 15, 0, createHash(1))
				requireContains(t, db, 15, 1, createHash(2))
				requireContains(t, db, 16, 0, createHash(3))
				requireContains(t, db, 16, 1, createHash(4))
			})
	})

	t.Run("ErrorWhenBeforeCurrentBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 3)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4998, 3)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 3)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 2)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenAtCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 3)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 3)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("MultipleSearchCheckpoints", func(t *testing.T) {
		block1 := eth.BlockID{Hash: createHash(11), Number: 11}
		block2 := eth.BlockID{Hash: createHash(12), Number: 12}
		block3 := eth.BlockID{Hash: createHash(15), Number: 15}
		block4 := eth.BlockID{Hash: createHash(16), Number: 16}
		// First checkpoint is at entry idx 0
		// Block 1 logs don't reach the second checkpoint
		block1LogCount := searchCheckpointFrequency - 10
		// Block 2 logs extend to just after the third checkpoint
		block2LogCount := searchCheckpointFrequency + 20
		// Block 3 logs extend to immediately before the fourth checkpoint
		block3LogCount := searchCheckpointFrequency - 16
		block4LogCount := 2
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := 0; i < block1LogCount; i++ {
					err := db.AddLog(createHash(i), block1, 3000, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 1", i)
				}
				for i := 0; i < block2LogCount; i++ {
					err := db.AddLog(createHash(i), block2, 3002, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 2", i)
				}
				for i := 0; i < block3LogCount; i++ {
					err := db.AddLog(createHash(i), block3, 3004, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 3", i)
				}
				// Verify that we're right before the fourth checkpoint will be written.
				// entryCount is the number of entries, so given 0 based indexing is the index of the next entry
				// the first checkpoint is at entry 0, the second at entry searchCheckpointFrequency etc
				// so the fourth is at entry 3*searchCheckpointFrequency
				require.EqualValues(t, 3*searchCheckpointFrequency, m.entryCount)
				for i := 0; i < block4LogCount; i++ {
					err := db.AddLog(createHash(i), block4, 3006, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 4", i)
				}
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				// Check that we wrote additional search checkpoints
				expectedCheckpointCount := 4
				expectedEntryCount := block1LogCount + block2LogCount + block3LogCount + block4LogCount + (2 * expectedCheckpointCount)
				require.EqualValues(t, expectedEntryCount, m.entryCount)
				// Check we can find all the logs.
				for i := 0; i < block1LogCount; i++ {
					requireContains(t, db, block1.Number, uint32(i), createHash(i))
				}
				// Block 2 logs extend to just after the third checkpoint
				for i := 0; i < block2LogCount; i++ {
					requireContains(t, db, block2.Number, uint32(i), createHash(i))
				}
				// Block 3 logs extend to immediately before the fourth checkpoint
				for i := 0; i < block3LogCount; i++ {
					requireContains(t, db, block3.Number, uint32(i), createHash(i))
				}
				// Block 4 logs start immediately after the fourth checkpoint
				for i := 0; i < block4LogCount; i++ {
					requireContains(t, db, block4.Number, uint32(i), createHash(i))
				}
			})
	})
}

func requireContains(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, logHash)
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Truef(t, result, "Did not find log %v in block %v with hash %v", logIdx, blockNum, logHash)
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
	require.NotZero(t, m.entriesReadForSearch, "Must read at least some entries to find the log")
}

func TestShouldRollBackInMemoryChangesOnWriteFailure(t *testing.T) {

}

func TestShouldRecoverWhenSearchCheckpointWrittenButNotCanonicalHash(t *testing.T) {

}

func TestShouldRecoverWhenPartialEntryWritten(t *testing.T) {

}

func TestShouldRecoverWhenInitiatingEventWrittenButNotExecutingLink(t *testing.T) {

}

//func TestRewind(t *testing.T) {
//	t.Run("WhenEmpty", func(t *testing.T) {
//		runDBTest(t, func(t *testing.T, db *DB) {},
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Rewind(100))
//				require.NoError(t, db.Rewind(0))
//			})
//	})
//
//	t.Run("AfterLastEntry", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(51, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(74, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(75))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(50, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(50, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//			})
//	})
//
//	t.Run("BeforeFirstEntry", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(25))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(50, 0, createHash(1))
//				require.NoError(t, err)
//				require.False(t, contains)
//
//				contains, err = db.Contains(50, 1, createHash(2))
//				require.NoError(t, err)
//				require.False(t, contains)
//			})
//	})
//
//	t.Run("AtFirstEntry", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(51, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(50))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(50, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(50, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(51, 0, createHash(1))
//				require.NoError(t, err)
//				require.False(t, contains)
//
//				contains, err = db.Contains(51, 1, createHash(2))
//				require.NoError(t, err)
//				require.False(t, contains)
//			})
//	})
//
//	t.Run("BetweenEntries", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(55))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(50, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(50, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(60, 0, createHash(1))
//				require.NoError(t, err)
//				require.False(t, contains)
//
//				contains, err = db.Contains(60, 1, createHash(2))
//				require.NoError(t, err)
//				require.False(t, contains)
//			})
//	})
//
//	t.Run("AtExistingEntry", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(59, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(61, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(60))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(59, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(59, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(60, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(60, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(61, 0, createHash(1))
//				require.NoError(t, err)
//				require.False(t, contains)
//
//				contains, err = db.Contains(61, 1, createHash(2))
//				require.NoError(t, err)
//				require.False(t, contains)
//			})
//	})
//
//	t.Run("AtLastEntry", func(t *testing.T) {
//		runDBTest(t,
//			func(t *testing.T, db *DB) {
//				require.NoError(t, db.Add(50, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Add(70, []common.Hash{createHash(1), createHash(2)}))
//				require.NoError(t, db.Rewind(70))
//			},
//			func(t *testing.T, db *DB) {
//				contains, err := db.Contains(50, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(50, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(60, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(60, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(70, 0, createHash(1))
//				require.NoError(t, err)
//				require.True(t, contains)
//
//				contains, err = db.Contains(70, 1, createHash(2))
//				require.NoError(t, err)
//				require.True(t, contains)
//			})
//	})
//
//	t.Run("ReaddDeletedBlocks", func(t *testing.T) {
//		runDBTest(t, func(t *testing.T, db *DB) {
//			require.NoError(t, db.Add(59, []common.Hash{createHash(1), createHash(2)}))
//			require.NoError(t, db.Add(60, []common.Hash{createHash(1), createHash(2)}))
//			require.NoError(t, db.Add(61, []common.Hash{createHash(1), createHash(2)}))
//			require.NoError(t, db.Rewind(60))
//		},
//			func(t *testing.T, db *DB) {
//				err := db.Add(59, []common.Hash{createHash(1), createHash(2)})
//				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block before rewound head")
//				err = db.Add(60, []common.Hash{createHash(1), createHash(2)})
//				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block that was rewound to")
//				err = db.Add(61, []common.Hash{createHash(1), createHash(2)})
//				require.NoError(t, err, "Can re-add deleted block")
//			})
//	})
//}

type stubMetrics struct {
	entryCount           int64
	entriesReadForSearch int64
}

func (s *stubMetrics) RecordEntryCount(count int64) {
	s.entryCount = count
}

func (s *stubMetrics) RecordSearchEntriesRead(count int64) {
	s.entriesReadForSearch = count
}

var _ Metrics = (*stubMetrics)(nil)
