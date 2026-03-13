package interop

import (
	"math"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// FuzzVerifiedDBCommitRewind performs random sequences of commit/rewind operations
// and verifies that the VerifiedDB maintains invariants throughout.
//
// Properties tested:
// P15: Commit(result) succeeds iff result.Timestamp == lastTimestamp + 1 (or first commit)
// P16: After Rewind(ts), LastTimestamp() returns ts - 1 (or uninitialized if all deleted)
// P17: After Rewind(ts), Get(t) errors for all t >= ts
// P18: After Rewind(ts), Commit(ts) succeeds (re-commit from rewind point)
// P19: ErrAlreadyCommitted and ErrNonSequential are correctly distinguished
// P20: JSON round-trip preserves all VerifiedResult fields
func FuzzVerifiedDBCommitRewind(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(12345))
	f.Add(int64(0))
	f.Add(int64(999999))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		twoChains := generateChainIDs(2, 10, 8443)

		// Choose a starting timestamp (activation timestamp)
		activationTS := uint64(rng.Intn(10000))

		// Track committed timestamps in-memory for verification
		committed := make(map[uint64]VerifiedResult)
		nextTS := activationTS

		// Number of operations to perform
		numOps := 5 + rng.Intn(20)

		for i := 0; i < numOps; i++ {
			op := rng.Intn(100)

			switch {
			case op < 50: // 50% chance: commit next sequential timestamp
				result := generateVerifiedResult(rng, nextTS, twoChains)

				err := db.Commit(result)
				require.NoError(t, err, "sequential commit should succeed at ts=%d", nextTS)

				// Verify P20: JSON round-trip preserves all fields
				retrieved, err := db.Get(nextTS)
				require.NoError(t, err)
				require.Equal(t, result.Timestamp, retrieved.Timestamp, "P20: timestamp preserved")
				require.Equal(t, result.L1Inclusion, retrieved.L1Inclusion, "P20: L1Inclusion preserved")
				for _, chainID := range twoChains {
					require.Equal(t, result.L2Heads[chainID], retrieved.L2Heads[chainID], "P20: L2Heads preserved for chain %s", chainID)
				}

				committed[nextTS] = result
				nextTS++

				// Verify LastTimestamp is updated
				lastTS, initialized := db.LastTimestamp()
				require.True(t, initialized)
				require.Equal(t, nextTS-1, lastTS, "LastTimestamp should be the last committed ts")

			case op < 65: // 15% chance: try non-sequential commit (should fail)
				if len(committed) == 0 {
					continue
				}

				// P19: ErrNonSequential - try to commit with a gap
				gapTS := nextTS + uint64(rng.Intn(10)) + 1
				err := db.Commit(generateVerifiedResult(rng, gapTS, twoChains[:1]))
				require.ErrorIs(t, err, ErrNonSequential, "P19: gap commit should return ErrNonSequential")

			case op < 80: // 15% chance: try duplicate commit (should fail)
				if len(committed) == 0 {
					continue
				}

				// P19: ErrAlreadyCommitted - try to commit an already committed timestamp
				var dupTS uint64
				for ts := range committed {
					dupTS = ts
					break
				}
				err := db.Commit(generateVerifiedResult(rng, dupTS, twoChains[:1]))
				require.ErrorIs(t, err, ErrAlreadyCommitted, "P19: duplicate commit should return ErrAlreadyCommitted")

			case op < 95: // 15% chance: rewind
				if len(committed) == 0 {
					continue
				}

				// Pick a random rewind point
				rewindTS := activationTS + uint64(rng.Intn(int(nextTS-activationTS)+1))

				deleted, err := db.Rewind(rewindTS)
				require.NoError(t, err)

				// P16: After Rewind(ts), LastTimestamp should be ts-1 (or uninitialized)
				lastTS, initialized := db.LastTimestamp()
				if rewindTS <= activationTS {
					// Rewound before or at first entry - all should be deleted
					if deleted {
						require.False(t, initialized, "P16: all entries deleted, should be uninitialized")
					}
				} else {
					// Check if there are still entries before rewindTS
					hasEntries := false
					for ts := range committed {
						if ts < rewindTS {
							hasEntries = true
							break
						}
					}
					if hasEntries && deleted {
						require.True(t, initialized)
						require.Equal(t, rewindTS-1, lastTS, "P16: LastTimestamp should be rewindTS-1")
					}
				}

				// P17: After Rewind(ts), Get(t) errors for all t >= ts
				for ts := range committed {
					if ts >= rewindTS {
						_, err := db.Get(ts)
						require.ErrorIs(t, err, ErrNotFound, "P17: ts=%d should be deleted after rewind to %d", ts, rewindTS)

						has, err := db.Has(ts)
						require.NoError(t, err)
						require.False(t, has, "P17: Has(ts=%d) should be false after rewind to %d", ts, rewindTS)
					}
				}

				// Update in-memory tracking
				for ts := range committed {
					if ts >= rewindTS {
						delete(committed, ts)
					}
				}

				// P18: After Rewind(ts), Commit(ts) succeeds (re-commit from rewind point)
				// The next commit should start from the rewind point
				if initialized {
					nextTS = lastTS + 1
				} else {
					// All entries deleted, next commit can start anywhere (first commit)
					nextTS = activationTS + uint64(rng.Intn(100))
				}

			default: // 5% chance: verify existing entries
				for ts, expected := range committed {
					retrieved, err := db.Get(ts)
					require.NoError(t, err, "committed ts=%d should be retrievable", ts)
					require.Equal(t, expected.Timestamp, retrieved.Timestamp)
					require.Equal(t, expected.L1Inclusion, retrieved.L1Inclusion)
					require.Equal(t, len(expected.L2Heads), len(retrieved.L2Heads))
				}
			}
		}

		// Final verification: all tracked entries should still exist
		for ts, expected := range committed {
			has, err := db.Has(ts)
			require.NoError(t, err)
			require.True(t, has, "committed ts=%d should exist in final check", ts)

			retrieved, err := db.Get(ts)
			require.NoError(t, err)
			require.Equal(t, expected.Timestamp, retrieved.Timestamp)
			require.Equal(t, expected.L1Inclusion, retrieved.L1Inclusion)
		}
	})
}

// FuzzVerifiedDBFirstCommit tests that the first commit can be at any timestamp
// and subsequent commits must be sequential.
func FuzzVerifiedDBFirstCommit(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(0))
	f.Add(int64(math.MaxInt64))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))
		dataDir := t.TempDir()

		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db.Close()

		singleChain := []eth.ChainID{eth.ChainIDFromUInt64(10)}

		// First commit at any timestamp should succeed
		firstTS := uint64(rng.Intn(1000000))
		err = db.Commit(generateVerifiedResult(rng, firstTS, singleChain))
		require.NoError(t, err, "first commit should succeed at any timestamp")

		// P15: next must be firstTS + 1
		err = db.Commit(generateVerifiedResult(rng, firstTS+1, singleChain))
		require.NoError(t, err, "P15: sequential commit should succeed")

		// Trying firstTS + 3 should fail with ErrNonSequential
		err = db.Commit(generateVerifiedResult(rng, firstTS+3, singleChain))
		require.ErrorIs(t, err, ErrNonSequential, "P15: non-sequential should fail")

		// Rewind all and recommit
		_, err = db.Rewind(firstTS)
		require.NoError(t, err)

		_, initialized := db.LastTimestamp()
		require.False(t, initialized, "all entries should be deleted after full rewind")

		// P18: first commit after full rewind succeeds at any timestamp
		newTS := uint64(rng.Intn(1000000))
		err = db.Commit(generateVerifiedResult(rng, newTS, singleChain))
		require.NoError(t, err, "P18: first commit after full rewind should succeed")

		lastTS, initialized := db.LastTimestamp()
		require.True(t, initialized)
		require.Equal(t, newTS, lastTS)
	})
}

// FuzzVerifiedDBPersistence tests that data survives close/reopen.
func FuzzVerifiedDBPersistence(f *testing.F) {
	f.Add(int64(42))
	f.Add(int64(0))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))
		dataDir := t.TempDir()

		singleChain := []eth.ChainID{eth.ChainIDFromUInt64(10)}
		startTS := uint64(rng.Intn(10000))
		numCommits := 2 + rng.Intn(8)

		results := make([]VerifiedResult, numCommits)

		// Phase 1: Write data
		db, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)

		for i := 0; i < numCommits; i++ {
			results[i] = generateVerifiedResult(rng, startTS+uint64(i), singleChain)
			err = db.Commit(results[i])
			require.NoError(t, err)
		}
		db.Close()

		// Phase 2: Reopen and verify
		db2, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer db2.Close()

		lastTS, initialized := db2.LastTimestamp()
		require.True(t, initialized)
		require.Equal(t, startTS+uint64(numCommits-1), lastTS)

		for _, expected := range results {
			retrieved, err := db2.Get(expected.Timestamp)
			require.NoError(t, err)
			require.Equal(t, expected.Timestamp, retrieved.Timestamp, "P20: persistence round-trip")
			require.Equal(t, expected.L1Inclusion, retrieved.L1Inclusion, "P20: L1Inclusion persisted")
		}

		// Next commit should continue from last
		err = db2.Commit(generateVerifiedResult(rng, lastTS+1, singleChain))
		require.NoError(t, err, "should continue sequential commits after reopen")
	})
}

