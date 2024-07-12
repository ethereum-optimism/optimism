package logs

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func createTruncatedHash(i int) types.TruncatedHash {
	return types.TruncateHash(createHash(i))
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 0}, 5000, 0, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("FirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 15, 0, createHash(1))
			})
	})

	t.Run("MultipleEntriesFromSameBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2, nil)
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 0, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(4), eth.BlockID{Hash: createHash(16), Number: 16}, 5002, 1, nil)
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4998, 0, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentBlockButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(13), Number: 13}, 5000, 0, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4998, 0, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 0, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenBeforeCurrentLogEventButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil)
				require.NoError(t, err)
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2, nil)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 1, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenAtCurrentLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 1, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenAtCurrentLogEventButAfterLastCheckpoint", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 2, nil))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 15}, 4998, 2, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenSkippingLogEvent", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, nil)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 2, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenFirstLogIsNotLogIdxZero", func(t *testing.T) {
		runDBTest(t, func(t *testing.T, db *DB, m *stubMetrics) {},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 5, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder)
			})
	})

	t.Run("ErrorWhenFirstLogOfNewBlockIsNotLogIdxZero", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(14), Number: 14}, 4996, 0, nil))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 4998, 1, nil)
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
					err := db.AddLog(createTruncatedHash(i), block1, 3000, uint32(i), nil)
					require.NoErrorf(t, err, "failed to add log %v of block 1", i)
				}
				for i := 0; i < block2LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block2, 3002, uint32(i), nil)
					require.NoErrorf(t, err, "failed to add log %v of block 2", i)
				}
				for i := 0; i < block3LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block3, 3004, uint32(i), nil)
					require.NoErrorf(t, err, "failed to add log %v of block 3", i)
				}
				// Verify that we're right before the fourth checkpoint will be written.
				// entryCount is the number of entries, so given 0 based indexing is the index of the next entry
				// the first checkpoint is at entry 0, the second at entry searchCheckpointFrequency etc
				// so the fourth is at entry 3*searchCheckpointFrequency
				require.EqualValues(t, 3*searchCheckpointFrequency, m.entryCount)
				for i := 0; i < block4LogCount; i++ {
					err := db.AddLog(createTruncatedHash(i), block4, 3006, uint32(i), nil)
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

func TestAddDependentLog(t *testing.T) {
	execMsg := types.ExecutingMessage{
		Chain:     3,
		BlockNum:  42894,
		LogIdx:    42,
		Timestamp: 8742482,
		Hash:      types.TruncateHash(createHash(8844)),
	}
	t.Run("FirstEntry", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, &execMsg)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 15, 0, createHash(1), execMsg)
			})
	})

	t.Run("CheckpointBetweenInitEventAndExecLink", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := uint32(0); m.entryCount < searchCheckpointFrequency-1; i++ {
					require.NoError(t, db.AddLog(createTruncatedHash(9), eth.BlockID{Hash: createHash(9), Number: 1}, 500, i, nil))
				}
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, &execMsg)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 15, 0, createHash(1), execMsg)
			})
	})

	t.Run("CheckpointBetweenInitEventAndExecLinkNotIncrementingBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {

				for i := uint32(0); m.entryCount < searchCheckpointFrequency-1; i++ {
					require.NoError(t, db.AddLog(createTruncatedHash(9), eth.BlockID{Hash: createHash(9), Number: 1}, 500, i, nil))
				}
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 1}, 5000, 253, &execMsg)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 1, 253, createHash(1), execMsg)
			})
	})

	t.Run("CheckpointBetweenExecLinkAndExecCheck", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := uint32(0); m.entryCount < searchCheckpointFrequency-2; i++ {
					require.NoError(t, db.AddLog(createTruncatedHash(9), eth.BlockID{Hash: createHash(9), Number: 1}, 500, i, nil))
				}
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 15}, 5000, 0, &execMsg)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 15, 0, createHash(1), execMsg)
			})
	})

	t.Run("CheckpointBetweenExecLinkAndExecCheckNotIncrementingBlock", func(t *testing.T) {
		runDBTest(t,
			func(t *testing.T, db *DB, m *stubMetrics) {
				for i := uint32(0); m.entryCount < searchCheckpointFrequency-2; i++ {
					require.NoError(t, db.AddLog(createTruncatedHash(9), eth.BlockID{Hash: createHash(9), Number: 1}, 500, i, nil))
				}
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(15), Number: 1}, 5000, 252, &execMsg)
				require.NoError(t, err)
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				requireContains(t, db, 1, 252, createHash(1), execMsg)
			})
	})
}

func TestContains(t *testing.T) {
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 2, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 0, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 1, nil))
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
			requireWrongHash(t, db, 50, 0, createHash(5), types.ExecutingMessage{})
		})
}

func TestExecutes(t *testing.T) {
	execMsg1 := types.ExecutingMessage{
		Chain:     33,
		BlockNum:  22,
		LogIdx:    99,
		Timestamp: 948294,
		Hash:      createTruncatedHash(332299),
	}
	execMsg2 := types.ExecutingMessage{
		Chain:     44,
		BlockNum:  55,
		LogIdx:    66,
		Timestamp: 77777,
		Hash:      createTruncatedHash(445566),
	}
	execMsg3 := types.ExecutingMessage{
		Chain:     77,
		BlockNum:  88,
		LogIdx:    89,
		Timestamp: 6578567,
		Hash:      createTruncatedHash(778889),
	}
	runDBTest(t,
		func(t *testing.T, db *DB, m *stubMetrics) {
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, &execMsg1))
			require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 2, nil))
			require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 0, &execMsg2))
			require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(52), Number: 52}, 500, 1, &execMsg3))
		},
		func(t *testing.T, db *DB, m *stubMetrics) {
			// Should find added logs
			requireExecutingMessage(t, db, 50, 0, types.ExecutingMessage{})
			requireExecutingMessage(t, db, 50, 1, execMsg1)
			requireExecutingMessage(t, db, 50, 2, types.ExecutingMessage{})
			requireExecutingMessage(t, db, 52, 0, execMsg2)
			requireExecutingMessage(t, db, 52, 1, execMsg3)

			// Should not find log when block number too low
			requireNotContains(t, db, 49, 0, createHash(1))

			// Should not find log when block number too high
			requireNotContains(t, db, 51, 0, createHash(1))

			// Should not find log when requested log after end of database
			requireNotContains(t, db, 52, 2, createHash(3))
			requireNotContains(t, db, 53, 0, createHash(3))

			// Should not find log when log index too high
			requireNotContains(t, db, 50, 3, createHash(2))
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
				err := db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(11), Number: 11}, 500, 0, nil)
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
				err := db.AddLog(createTruncatedHash(1), block, 500, 0, nil)
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
					err := db.AddLog(createTruncatedHash(i), block, uint64(i)*2, 0, nil)
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
	require.Equal(t, types.TruncateHash(expectedHash), hash)
}

func requireContains(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash, execMsg ...types.ExecutingMessage) {
	require.LessOrEqual(t, len(execMsg), 1, "cannot have multiple executing messages for a single log")
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, types.TruncateHash(logHash))
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Truef(t, result, "Did not find log %v in block %v with hash %v", logIdx, blockNum, logHash)
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
	require.NotZero(t, m.entriesReadForSearch, "Must read at least some entries to find the log")

	var expectedExecMsg types.ExecutingMessage
	if len(execMsg) == 1 {
		expectedExecMsg = execMsg[0]
	}
	requireExecutingMessage(t, db, blockNum, logIdx, expectedExecMsg)
}

func requireNotContains(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, types.TruncateHash(logHash))
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Falsef(t, result, "Found unexpected log %v in block %v with hash %v", logIdx, blockNum, logHash)
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")

	_, err = db.Executes(blockNum, logIdx)
	require.ErrorIs(t, err, ErrNotFound, "Found unexpected log when getting executing message")
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
}

func requireExecutingMessage(t *testing.T, db *DB, blockNum uint64, logIdx uint32, execMsg types.ExecutingMessage) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	actualExecMsg, err := db.Executes(blockNum, logIdx)
	require.NoError(t, err, "Error when searching for executing message")
	require.Equal(t, execMsg, actualExecMsg, "Should return matching executing message")
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
	require.NotZero(t, m.entriesReadForSearch, "Must read at least some entries to find the log")
}

func requireWrongHash(t *testing.T, db *DB, blockNum uint64, logIdx uint32, logHash common.Hash, execMsg types.ExecutingMessage) {
	m, ok := db.m.(*stubMetrics)
	require.True(t, ok, "Did not get the expected metrics type")
	result, err := db.Contains(blockNum, logIdx, types.TruncateHash(logHash))
	require.NoErrorf(t, err, "Error searching for log %v in block %v", logIdx, blockNum)
	require.Falsef(t, result, "Found unexpected log %v in block %v with hash %v", logIdx, blockNum, logHash)

	_, err = db.Executes(blockNum, logIdx)
	require.NoError(t, err, "Error when searching for executing message")
	require.LessOrEqual(t, m.entriesReadForSearch, int64(searchCheckpointFrequency), "Should not need to read more than between two checkpoints")
}

func TestRecoverOnCreate(t *testing.T) {
	createDb := func(t *testing.T, store *stubEntryStore) (*DB, *stubMetrics, error) {
		logger := testlog.Logger(t, log.LvlInfo)
		m := &stubMetrics{}
		db, err := NewFromEntryStore(logger, m, store)
		return db, m, err
	}

	validInitEvent, err := newInitiatingEvent(logContext{blockNum: 1, logIdx: 0}, 1, 0, createTruncatedHash(1), false)
	require.NoError(t, err)
	validEventSequence := []entrydb.Entry{
		newSearchCheckpoint(1, 0, 100).encode(),
		newCanonicalHash(createTruncatedHash(344)).encode(),
		validInitEvent.encode(),
	}
	var emptyEventSequence []entrydb.Entry

	for _, prefixEvents := range [][]entrydb.Entry{emptyEventSequence, validEventSequence} {
		prefixEvents := prefixEvents
		storeWithEvents := func(evts ...entrydb.Entry) *stubEntryStore {
			store := &stubEntryStore{}
			store.entries = append(store.entries, prefixEvents...)
			store.entries = append(store.entries, evts...)
			return store
		}
		t.Run(fmt.Sprintf("PrefixEvents-%v", len(prefixEvents)), func(t *testing.T) {
			t.Run("NoTruncateWhenLastEntryIsLogWithNoExecMessage", func(t *testing.T) {
				initEvent, err := newInitiatingEvent(logContext{blockNum: 3, logIdx: 0}, 3, 0, createTruncatedHash(1), false)
				require.NoError(t, err)
				store := storeWithEvents(
					newSearchCheckpoint(3, 0, 100).encode(),
					newCanonicalHash(createTruncatedHash(344)).encode(),
					initEvent.encode(),
				)
				db, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents)+3, m.entryCount)
				requireContains(t, db, 3, 0, createHash(1))
			})

			t.Run("NoTruncateWhenLastEntryIsExecutingCheck", func(t *testing.T) {
				initEvent, err := newInitiatingEvent(logContext{blockNum: 3, logIdx: 0}, 3, 0, createTruncatedHash(1), true)
				execMsg := types.ExecutingMessage{
					Chain:     4,
					BlockNum:  10,
					LogIdx:    4,
					Timestamp: 1288,
					Hash:      createTruncatedHash(4),
				}
				require.NoError(t, err)
				linkEvt, err := newExecutingLink(execMsg)
				require.NoError(t, err)
				store := storeWithEvents(
					newSearchCheckpoint(3, 0, 100).encode(),
					newCanonicalHash(createTruncatedHash(344)).encode(),
					initEvent.encode(),
					linkEvt.encode(),
					newExecutingCheck(execMsg.Hash).encode(),
				)
				db, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents)+5, m.entryCount)
				requireContains(t, db, 3, 0, createHash(1), execMsg)
			})

			t.Run("TruncateWhenLastEntrySearchCheckpoint", func(t *testing.T) {
				store := storeWithEvents(newSearchCheckpoint(3, 0, 100).encode())
				_, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents), m.entryCount)
			})

			t.Run("TruncateWhenLastEntryCanonicalHash", func(t *testing.T) {
				store := storeWithEvents(
					newSearchCheckpoint(3, 0, 100).encode(),
					newCanonicalHash(createTruncatedHash(344)).encode(),
				)
				_, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents), m.entryCount)
			})

			t.Run("TruncateWhenLastEntryInitEventWithExecMsg", func(t *testing.T) {
				initEvent, err := newInitiatingEvent(logContext{blockNum: 3, logIdx: 0}, 3, 0, createTruncatedHash(1), true)
				require.NoError(t, err)
				store := storeWithEvents(
					newSearchCheckpoint(3, 0, 100).encode(),
					newCanonicalHash(createTruncatedHash(344)).encode(),
					initEvent.encode(),
				)
				_, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents), m.entryCount)
			})

			t.Run("TruncateWhenLastEntryInitEventWithExecLink", func(t *testing.T) {
				initEvent, err := newInitiatingEvent(logContext{blockNum: 3, logIdx: 0}, 3, 0, createTruncatedHash(1), true)
				require.NoError(t, err)
				execMsg := types.ExecutingMessage{
					Chain:     4,
					BlockNum:  10,
					LogIdx:    4,
					Timestamp: 1288,
					Hash:      createTruncatedHash(4),
				}
				require.NoError(t, err)
				linkEvt, err := newExecutingLink(execMsg)
				require.NoError(t, err)
				store := storeWithEvents(
					newSearchCheckpoint(3, 0, 100).encode(),
					newCanonicalHash(createTruncatedHash(344)).encode(),
					initEvent.encode(),
					linkEvt.encode(),
				)
				_, m, err := createDb(t, store)
				require.NoError(t, err)
				require.EqualValues(t, len(prefixEvents), m.entryCount)
			})
		})
	}
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(3), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(4), eth.BlockID{Hash: createHash(74), Number: 74}, 700, 0, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 1, nil))
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
					require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, i, nil))
				}
				require.EqualValues(t, searchCheckpointFrequency, m.entryCount)
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 0, nil))
				require.EqualValues(t, searchCheckpointFrequency+3, m.entryCount, "Should have inserted new checkpoint and extra log")
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(51), Number: 51}, 502, 1, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 1, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(50), Number: 50}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(70), Number: 70}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(70), Number: 70}, 502, 1, nil))
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
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 0, nil))
				require.NoError(t, db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(61), Number: 61}, 502, 1, nil))
				require.NoError(t, db.Rewind(60))
			},
			func(t *testing.T, db *DB, m *stubMetrics) {
				err := db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(59), Number: 59}, 500, 1, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block before rewound head")
				err = db.AddLog(createTruncatedHash(2), eth.BlockID{Hash: createHash(60), Number: 60}, 502, 1, nil)
				require.ErrorIs(t, err, ErrLogOutOfOrder, "Cannot add block that was rewound to")
				err = db.AddLog(createTruncatedHash(1), eth.BlockID{Hash: createHash(60), Number: 61}, 502, 0, nil)
				require.NoError(t, err, "Can re-add deleted block")
			})
	})
}

type stubMetrics struct {
	entryCount           int64
	entriesReadForSearch int64
}

func (s *stubMetrics) RecordDBEntryCount(count int64) {
	s.entryCount = count
}

func (s *stubMetrics) RecordDBSearchEntriesRead(count int64) {
	s.entriesReadForSearch = count
}

var _ Metrics = (*stubMetrics)(nil)

type stubEntryStore struct {
	entries []entrydb.Entry
}

func (s *stubEntryStore) Size() int64 {
	return int64(len(s.entries))
}

func (s *stubEntryStore) LastEntryIdx() entrydb.EntryIdx {
	return entrydb.EntryIdx(s.Size() - 1)
}

func (s *stubEntryStore) Read(idx entrydb.EntryIdx) (entrydb.Entry, error) {
	if idx < entrydb.EntryIdx(len(s.entries)) {
		return s.entries[idx], nil
	}
	return entrydb.Entry{}, io.EOF
}

func (s *stubEntryStore) Append(entries ...entrydb.Entry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

func (s *stubEntryStore) Truncate(idx entrydb.EntryIdx) error {
	s.entries = s.entries[:min(s.Size()-1, int64(idx+1))]
	return nil
}

func (s *stubEntryStore) Close() error {
	return nil
}

var _ EntryStore = (*stubEntryStore)(nil)
