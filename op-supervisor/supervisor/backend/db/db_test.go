package db

import (
	"bytes"
	"io"
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

func createTruncatedHash(i int) TruncatedHash {
	return TruncateHash(createHash(i))
}

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
	createDb := func(t *testing.T, dir string) (*DB, *stubMetrics, string) {
		logger := testlog.Logger(t, log.LvlTrace)
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
		return db, m, path
	}

	t.Run("New", func(t *testing.T) {
		db, m, _ := createDb(t, t.TempDir())
		setup(t, db, m)
		assert(t, db, m)
	})

	t.Run("Existing", func(t *testing.T) {
		dir := t.TempDir()
		db, m, path := createDb(t, dir)
		setup(t, db, m)
		// Close and recreate the database
		require.NoError(t, db.Close())
		checkDBInvariants(t, path, m)

		db2, m, path := createDb(t, dir)
		assert(t, db2, m)
		checkDBInvariants(t, path, m)
	})
}

func TestEmptyDbDoesNotFindEntry(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {},
		func(t *testing.T, db *DB, m *stubMetrics) {
			requireNotContains(t, db, 0, 0, createHash(1))
			requireNotContains(t, db, 0, 0, common.Hash{})
		})
}

func TestAddLog(t *testing.T) {
	t.Run("BlockZero", func(t *testing.T) {
		// There are no logs in the genesis block so recording an entry for block 0 should be rejected.
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 0}, 5000, 0)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("FirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 15, 0, createHash(1))
			})
	})

	t.Run("MultipleEntriesFromSameBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2)
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 0)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(4), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 1)
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4998, 0)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentBlockButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(13), Number: 13}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4998, 0)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 0)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentLogEventButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 1)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenAtCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 1)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenAtCurrentLogEventButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 2)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenSkippingLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 2)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenFirstLogIsNotLogIdxZero", func(t *testing.T) {
		runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 5)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenFirstLogOfNewBlockIsNotLogIdxZero", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4996, 0))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 1)
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
					err := db.AddLog(createTruncatedHash(i), block1, 3000, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 1", i)
				}
				for i := 0; i < block2LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block2, 3002, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 2", i)
				}
				for i := 0; i < block3LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block3, 3004, uint32(i))
					require.NoErrorf(t, err, "failed to add log %v of block 3", i)
				}
				// Verify that we're right before the fourth checkpoint will be written.
				// entryCount is the number of entries, so given 0 based indexing is the index of the next entry
				// the first checkpoint is at entry 0, the second at entry searchCheckpointFrequency etc
				// so the fourth is at entry 3*searchCheckpointFrequency
				require.EqualValues(t, 3*searchCheckpointFrequency, m.entryCount)
				for i := 0; i < block4LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block4, 3006, uint32(i))
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

func TestContains(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
			require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 2))
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 0))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 1))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			// Should find added logs
			requireContains(t, db, 50, 0, createHash(1))
			requireContains(t, db, 50, 1, createHash(3))
			requireContains(t, db, 50, 2, createHash(2))
			requireContains(t, db, 52, 0, createHash(1))
			requireContains(t, db, 52, 1, createHash(3))

			// Should not find log when block number too low
			requireNotContains(t, db, 49, 0, createHash(1))

			// Should not find log when block number too high
			requireNotContains(t, db, 51, 0, createHash(1))

			// Should not find log when requested log after end of database
			requireNotContains(t, db, 52, 2, createHash(3))
			requireNotContains(t, db, 53, 0, createHash(3))

			// Should not find log when log index too high
			requireNotContains(t, db, 50, 3, createHash(2))

			// Should not find log when hash doesn't match log at block number and index
			requireNotContains(t, db, 50, 0, createHash(5))
		})
}

func TestGetBlockInfo(t *testing.T) {
	t.Run("ReturnsEOFWhenEmpty", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				_, _, err := db.ClosestBlockInfo(10)
				require.ErrorIs(t, err, io.EOF)
			})
	})

	t.Run("ReturnsEOFWhenRequestedBlockBeforeFirstSearchCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(11), Number: 11}, 500, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				_, _, err := db.ClosestBlockInfo(10)
				require.ErrorIs(t, err, io.EOF)
			})
	})

	t.Run("ReturnFirstBlockInfo", func(t *testing.T) {
		block := eth.BlockID{Hash: createHash(11), Number: 11}
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), block, 500, 0)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireClosestBlockInfo(t, db, 11, block.Number, block.Hash)
				requireClosestBlockInfo(t, db, 12, block.Number, block.Hash)
				requireClosestBlockInfo(t, db, 200, block.Number, block.Hash)
			})
	})

	t.Run("ReturnClosestCheckpointBlockInfo", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := 1; i < searchCheckpointFrequency+3; i++ {
					block := eth.BlockID{Hash: createHash(i), Number: uint64(i)}
					err := db.AddLog(createTruncatedHash(i), block, uint64(i)*2, 0)
					require.NoError(t, err)
				}
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				// Expect block from the first checkpoint
				requireClosestBlockInfo(t, db, 1, 1, createHash(1))
				requireClosestBlockInfo(t, db, 10, 1, createHash(1))
				requireClosestBlockInfo(t, db, searchCheckpointFrequency-3, 1, createHash(1))

				// Expect block from the second checkpoint
				// 2 entries used for initial checkpoint but we start at block 1
				secondCheckpointBlockNum := searchCheckpointFrequency - 1
				requireClosestBlockInfo(t, db, uint64(secondCheckpointBlockNum), uint64(secondCheckpointBlockNum), createHash(secondCheckpointBlockNum))
				requireClosestBlockInfo(t, db, uint64(secondCheckpointBlockNum)+1, uint64(secondCheckpointBlockNum), createHash(secondCheckpointBlockNum))
				requireClosestBlockInfo(t, db, uint64(secondCheckpointBlockNum)+2, uint64(secondCheckpointBlockNum), createHash(secondCheckpointBlockNum))
			})
	})
}

func requireClosestBlockInfo(t *testing.T, db *DB, searchFor uint64, expectedBlockNum uint64, expectedHash common.Hash) {
	blockNum, hash, err := db.ClosestBlockInfo(searchFor)
	require.NoError(t, err)
	require.Equal(t, expectedBlockNum, blockNum)
	require.Equal(t, TruncateHash(expectedHash), hash)
}

func requireContains(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, TruncateHash(logHash))
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Truef(t, result, "Did not find log %v in block %v with hash %v", logIdx, blockNum, logHash)
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
	require.NotZero(t, m.entriesReadForSearch, "Must read at least some entries to find the log")
}

func requireNotContains(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, TruncateHash(logHash))
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Falsef(t, result, "Found unexpected log %v in block %v with hash %v", logIdx, blockNum, logHash)
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
}

func TestShouldRollBackInMemoryChangesOnWriteFailure(t *testing.T) {
	t.Skip("TODO(optimism#10857)")
}

func TestShouldRecoverWhenSearchCheckpointWrittenButNotCanonicalHash(t *testing.T) {
	t.Skip("TODO(optimism#10857)")
}

func TestShouldRecoverWhenPartialEntryWritten(t *testing.T) {
	t.Skip("TODO(optimism#10857)")
}

func TestShouldRecoverWhenInitiatingEventWrittenButNotExecutingLink(t *testing.T) {
	t.Skip("TODO(optimism#10857)")
}

func TestRewind(t *testing.T) {
	t.Run("WhenEmpty", func(t *testing.T) {
		runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.Rewind(100))
				require.NoError(t, db.Rewind(0))
			})
	})

	t.Run("AfterLastBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(4), eth.BlockID{Hash: createHash(74), Number: 74}, 700, 0))
				require.NoError(t, db.Rewind(75))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 50, 0, createHash(1))
				requireContains(t, db, 50, 1, createHash(2))
				requireContains(t, db, 51, 0, createHash(3))
				requireContains(t, db, 74, 0, createHash(4))
			})
	})

	t.Run("BeforeFirstBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
				require.NoError(t, db.Rewind(25))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireNotContains(t, db, 50, 0, createHash(1))
				requireNotContains(t, db, 50, 0, createHash(1))
				require.Zero(t, m.entryCount)
			})
	})

	t.Run("AtFirstBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 1))
				require.NoError(t, db.Rewind(50))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 50, 0, createHash(1))
				requireContains(t, db, 50, 1, createHash(2))
				requireNotContains(t, db, 51, 0, createHash(1))
				requireNotContains(t, db, 51, 1, createHash(2))
			})
	})

	t.Run("AtSecondCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := uint32(0); m.entryCount < searchCheckpointFrequency; i++ {
					require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, i))
				}
				require.EqualValues(t, searchCheckpointFrequency, m.entryCount)
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0))
				require.EqualValues(t, searchCheckpointFrequency+3, m.entryCount, "Should have inserted new checkpoint and extra log")
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 1))
				require.NoError(t, db.Rewind(50))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.EqualValues(t, searchCheckpointFrequency, m.entryCount, "Should have deleted second checkpoint")
				requireContains(t, db, 50, 0, createHash(1))
				requireContains(t, db, 50, 1, createHash(1))
				requireNotContains(t, db, 51, 0, createHash(1))
				requireNotContains(t, db, 51, 1, createHash(2))
			})
	})

	t.Run("BetweenLogEntries", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1))
				require.NoError(t, db.Rewind(55))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 50, 0, createHash(1))
				requireContains(t, db, 50, 1, createHash(2))
				requireNotContains(t, db, 60, 0, createHash(1))
				requireNotContains(t, db, 60, 1, createHash(2))
			})
	})

	t.Run("AtExistingLogEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 1))
				require.NoError(t, db.Rewind(60))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 59, 0, createHash(1))
				requireContains(t, db, 59, 1, createHash(2))
				requireContains(t, db, 60, 0, createHash(1))
				requireContains(t, db, 60, 1, createHash(2))
				requireNotContains(t, db, 61, 0, createHash(1))
				requireNotContains(t, db, 61, 1, createHash(2))
			})
	})

	t.Run("AtLastEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(70), Number: 70}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(70), Number: 70}, 502, 1))
				require.NoError(t, db.Rewind(70))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 50, 0, createHash(1))
				requireContains(t, db, 50, 1, createHash(2))
				requireContains(t, db, 60, 0, createHash(1))
				requireContains(t, db, 60, 1, createHash(2))
				requireContains(t, db, 70, 0, createHash(1))
				requireContains(t, db, 70, 1, createHash(2))
			})
	})

	t.Run("ReaddDeletedBlocks", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 0))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 1))
				require.NoError(t, db.Rewind(60))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1)
				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block before rewound head")
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1)
				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block that was rewound to")
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 61}, 502, 0)
				require.NoError(t, err, "Can re-add deleted block")
			})
	})
}

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
