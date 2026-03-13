package chain_container

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

// FuzzDenyListAddContains performs random sequences of Add and Contains operations
// and verifies DenyList invariants.
//
// Properties:
// P21: Contains(h, hash) returns true iff Add(h, hash) was previously called
// P22: Add is idempotent
// P23: Hashes at different heights are isolated
// P24: Concatenated 32-byte hash storage handles boundary alignment correctly
func FuzzDenyListAddContains(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(12345))
	f.Add(int64(0))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		dl := NewMemoryDenyList()

		// Track all adds in-memory for verification
		added := make(map[uint64]map[common.Hash]bool)

		numOps := 10 + rng.Intn(50)
		numHeights := 1 + rng.Intn(10)  // use limited height range to get collisions
		numHashes := 1 + rng.Intn(20)

		// Pre-generate some heights and hashes for reuse
		heights := make([]uint64, numHeights)
		for i := range heights {
			heights[i] = uint64(rng.Intn(1000))
		}
		hashes := make([]common.Hash, numHashes)
		for i := range hashes {
			rng.Read(hashes[i][:])
		}

		for i := 0; i < numOps; i++ {
			op := rng.Intn(100)
			height := heights[rng.Intn(numHeights)]
			hash := hashes[rng.Intn(numHashes)]

			switch {
			case op < 50: // 50% Add
				err := dl.Add(height, hash)
				require.NoError(t, err)

				if added[height] == nil {
					added[height] = make(map[common.Hash]bool)
				}
				added[height][hash] = true

				// P21: Immediately verify add
				found, err := dl.Contains(height, hash)
				require.NoError(t, err)
				require.True(t, found, "P21: hash should be found immediately after Add at height %d", height)

			case op < 70: // 20% Contains
				found, err := dl.Contains(height, hash)
				require.NoError(t, err)

				// P21: Contains iff previously Added
				wasAdded := added[height] != nil && added[height][hash]
				require.Equal(t, wasAdded, found, "P21: Contains(%d, %s) should match tracked state", height, hash)

			case op < 85: // 15% Duplicate Add (P22: idempotency)
				if added[height] != nil && len(added[height]) > 0 {
					// Pick a hash that was already added at this height
					var existingHash common.Hash
					for h := range added[height] {
						existingHash = h
						break
					}

					// Add again — should be idempotent
					err := dl.Add(height, existingHash)
					require.NoError(t, err)

					// P22: GetDeniedHashes should still have same count
					deniedHashes, err := dl.GetDeniedHashes(height)
					require.NoError(t, err)
					require.Equal(t, len(added[height]), len(deniedHashes),
						"P22: duplicate Add should not increase hash count at height %d", height)
				}

			default: // 15% GetDeniedHashes & isolation check
				deniedHashes, err := dl.GetDeniedHashes(height)
				require.NoError(t, err)

				expectedCount := 0
				if added[height] != nil {
					expectedCount = len(added[height])
				}
				require.Equal(t, expectedCount, len(deniedHashes),
					"P23: GetDeniedHashes count should match tracked state at height %d", height)

				// P23: Verify each returned hash was actually added at this height
				for _, h := range deniedHashes {
					require.True(t, added[height][h],
						"P23: returned hash %s was not added at height %d (isolation violation)", h, height)
				}

				// P24: Verify no hash from another height leaks in
				for _, h := range deniedHashes {
					for otherHeight, otherHashes := range added {
						if otherHeight != height && otherHashes[h] {
							// Hash exists at another height too — that's fine
							// But verify it's actually at THIS height too
							require.True(t, added[height][h],
								"P24: hash %s at height %d might be a boundary alignment issue", h, height)
						}
					}
				}
			}
		}

		// Final verification: check all tracked state
		for height, hashSet := range added {
			for hash := range hashSet {
				found, err := dl.Contains(height, hash)
				require.NoError(t, err)
				require.True(t, found, "P21: final check - hash %s should exist at height %d", hash, height)
			}

			deniedHashes, err := dl.GetDeniedHashes(height)
			require.NoError(t, err)
			require.Equal(t, len(hashSet), len(deniedHashes),
				"P24: final check - hash count mismatch at height %d", height)
		}
	})
}

// FuzzDenyListConcurrent tests thread safety of the DenyList by running
// parallel Add and Contains operations from multiple goroutines.
func FuzzDenyListConcurrent(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		dl := NewMemoryDenyList()

		numWorkers := 2 + rng.Intn(6) // 2-7 workers
		opsPerWorker := 10 + rng.Intn(40)

		// Pre-generate hashes for each worker to avoid rng contention
		type workerData struct {
			heights []uint64
			hashes  []common.Hash
		}
		workers := make([]workerData, numWorkers)
		for i := range workers {
			wd := workerData{
				heights: make([]uint64, opsPerWorker),
				hashes:  make([]common.Hash, opsPerWorker),
			}
			for j := 0; j < opsPerWorker; j++ {
				wd.heights[j] = uint64(i*100 + rng.Intn(50)) // partially overlapping ranges
				rng.Read(wd.hashes[j][:])
			}
			workers[i] = wd
		}

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				wd := workers[workerID]

				for j := 0; j < opsPerWorker; j++ {
					height := wd.heights[j]
					hash := wd.hashes[j]

					// Add
					err := dl.Add(height, hash)
					assert.NoError(t, err, "worker %d: Add should not error", workerID)

					// Read-after-write should always find it
					found, err := dl.Contains(height, hash)
					assert.NoError(t, err, "worker %d: Contains should not error", workerID)
					assert.True(t, found, "worker %d: should find own hash at height %d", workerID, height)

					// Read from another worker's range (should not error)
					otherWorker := (workerID + 1) % numWorkers
					otherHeight := workers[otherWorker].heights[j%len(workers[otherWorker].heights)]
					_, err = dl.Contains(otherHeight, common.Hash{})
					assert.NoError(t, err, "worker %d: cross-range Contains should not error", workerID)
				}
			}(i)
		}

		wg.Wait()

		// Verify all writes are visible after all goroutines complete
		for workerID := 0; workerID < numWorkers; workerID++ {
			wd := workers[workerID]
			for j := 0; j < opsPerWorker; j++ {
				found, err := dl.Contains(wd.heights[j], wd.hashes[j])
				require.NoError(t, err)
				require.True(t, found,
					"worker %d op %d: hash should be visible after concurrent writes complete", workerID, j)
			}
		}
	})
}

// FuzzDenyListBoundaryAlignment tests the real bbolt-backed DenyList's
// concatenated 32-byte hash storage for boundary alignment correctness.
//
// The production DenyList stores multiple hashes at a single height as
// raw concatenated bytes: hash1[32] || hash2[32] || ... || hashN[32].
// Contains and GetDeniedHashes iterate with i += 32 steps. If the stored
// data is ever misaligned (len % 32 != 0), trailing bytes are silently
// dropped, causing data loss.
//
// This test:
// 1. Uses the REAL bbolt-backed DenyList (not MemoryDenyList)
// 2. Adds many hashes per height to stress the concatenation
// 3. Verifies raw storage is always 32-byte aligned
// 4. Verifies every hash survives the round-trip through byte storage
// 5. Tests adjacent heights to verify key isolation in bbolt
//
// Properties:
// P24: Concatenated 32-byte hash storage handles boundary alignment correctly
func FuzzDenyListBoundaryAlignment(f *testing.F) {
	f.Add(int64(1), 5)
	f.Add(int64(42), 20)
	f.Add(int64(100), 1)
	f.Add(int64(0), 50)

	f.Fuzz(func(t *testing.T, seed int64, hashesPerHeight int) {
		rng := rand.New(rand.NewSource(seed))

		if hashesPerHeight < 1 {
			hashesPerHeight = 1
		}
		if hashesPerHeight > 64 {
			hashesPerHeight = 64
		}

		dataDir := t.TempDir()
		dl, err := OpenDenyList(dataDir)
		require.NoError(t, err)
		defer dl.Close()

		// Use adjacent heights to test key isolation
		numHeights := 2 + rng.Intn(5)
		baseHeight := uint64(rng.Intn(10000))

		// Track what we add: height -> ordered list of unique hashes
		added := make(map[uint64][]common.Hash)

		for h := 0; h < numHeights; h++ {
			height := baseHeight + uint64(h)
			seen := make(map[common.Hash]bool)

			for i := 0; i < hashesPerHeight; i++ {
				var hash common.Hash
				rng.Read(hash[:])

				// Skip duplicates within same height
				if seen[hash] {
					continue
				}
				seen[hash] = true

				err := dl.Add(height, hash)
				require.NoError(t, err)
				added[height] = append(added[height], hash)
			}
		}

		// Verify 1: Raw storage alignment
		// Read the bbolt database directly to check that every value
		// is exactly len % 32 == 0 (no partial hashes).
		err = dl.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(denyListBucketName)
			require.NotNil(t, b)

			for height := range added {
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, height)

				raw := b.Get(key)
				require.NotNil(t, raw,
					"P24: raw data should exist at height %d", height)
				require.Equal(t, 0, len(raw)%common.HashLength,
					"P24: raw storage length %d is not 32-byte aligned at height %d (has %d trailing bytes)",
					len(raw), height, len(raw)%common.HashLength)

				// Verify the raw byte count matches expected hash count
				expectedCount := len(added[height])
				actualCount := len(raw) / common.HashLength
				require.Equal(t, expectedCount, actualCount,
					"P24: raw storage at height %d has %d hashes but expected %d",
					height, actualCount, expectedCount)
			}
			return nil
		})
		require.NoError(t, err)

		// Verify 2: Every added hash is retrievable via Contains
		for height, hashes := range added {
			for _, hash := range hashes {
				found, err := dl.Contains(height, hash)
				require.NoError(t, err)
				require.True(t, found,
					"P24: hash %s should be found at height %d after Add", hash, height)
			}
		}

		// Verify 3: GetDeniedHashes returns exactly the right set
		for height, expectedHashes := range added {
			gotHashes, err := dl.GetDeniedHashes(height)
			require.NoError(t, err)
			require.Equal(t, len(expectedHashes), len(gotHashes),
				"P24: GetDeniedHashes count mismatch at height %d", height)

			gotSet := make(map[common.Hash]bool)
			for _, h := range gotHashes {
				gotSet[h] = true
			}
			for _, h := range expectedHashes {
				require.True(t, gotSet[h],
					"P24: expected hash %s not found in GetDeniedHashes at height %d", h, height)
			}
		}

		// Verify 4: Height isolation — hashes at one height don't leak to adjacent
		for height, hashes := range added {
			for _, hash := range hashes {
				for otherHeight := range added {
					if otherHeight == height {
						continue
					}
					// Hash should only be found at otherHeight if it was also added there
					found, err := dl.Contains(otherHeight, hash)
					require.NoError(t, err)

					wasAddedAtOther := false
					for _, oh := range added[otherHeight] {
						if oh == hash {
							wasAddedAtOther = true
							break
						}
					}
					require.Equal(t, wasAddedAtOther, found,
						"P24: hash %s isolation violation between heights %d and %d",
						hash, height, otherHeight)
				}
			}
		}

		// Verify 5: Duplicate adds don't corrupt alignment
		// Re-add every hash and verify storage is unchanged
		for height, hashes := range added {
			for _, hash := range hashes {
				err := dl.Add(height, hash)
				require.NoError(t, err)
			}
		}

		err = dl.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(denyListBucketName)
			for height, expectedHashes := range added {
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, height)

				raw := b.Get(key)
				require.Equal(t, 0, len(raw)%common.HashLength,
					"P24: alignment broken after duplicate adds at height %d", height)
				require.Equal(t, len(expectedHashes), len(raw)/common.HashLength,
					"P24: duplicate adds changed hash count at height %d", height)
			}
			return nil
		})
		require.NoError(t, err)
	})
}
