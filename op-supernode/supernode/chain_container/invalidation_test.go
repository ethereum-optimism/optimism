package chain_container

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDenyList_AddAndContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, dl *DenyList)
		check func(t *testing.T, dl *DenyList)
	}{
		{
			name: "single hash at height",
			setup: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
				require.NoError(t, dl.Add(100, hash))
			},
			check: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
				found, err := dl.Contains(100, hash)
				require.NoError(t, err)
				require.True(t, found, "hash should be found at height 100")
			},
		},
		{
			name: "multiple hashes same height",
			setup: func(t *testing.T, dl *DenyList) {
				hashes := []common.Hash{
					common.HexToHash("0xaaaa"),
					common.HexToHash("0xbbbb"),
					common.HexToHash("0xcccc"),
				}
				for _, h := range hashes {
					require.NoError(t, dl.Add(50, h))
				}
			},
			check: func(t *testing.T, dl *DenyList) {
				hashes := []common.Hash{
					common.HexToHash("0xaaaa"),
					common.HexToHash("0xbbbb"),
					common.HexToHash("0xcccc"),
				}
				for _, h := range hashes {
					found, err := dl.Contains(50, h)
					require.NoError(t, err)
					require.True(t, found, "hash %s should be found at height 50", h)
				}
			},
		},
		{
			name: "hash at wrong height returns false",
			setup: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0xdddd")
				require.NoError(t, dl.Add(10, hash))
			},
			check: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0xdddd")
				// Check at different height
				found, err := dl.Contains(11, hash)
				require.NoError(t, err)
				require.False(t, found, "hash should NOT be found at height 11")

				// Verify it IS at height 10
				found, err = dl.Contains(10, hash)
				require.NoError(t, err)
				require.True(t, found, "hash should be found at height 10")
			},
		},
		{
			name: "duplicate add is idempotent",
			setup: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0xeeee")
				require.NoError(t, dl.Add(200, hash))
				require.NoError(t, dl.Add(200, hash)) // Add again
				require.NoError(t, dl.Add(200, hash)) // And again
			},
			check: func(t *testing.T, dl *DenyList) {
				hash := common.HexToHash("0xeeee")
				hashes, err := dl.GetDeniedHashes(200)
				require.NoError(t, err)
				require.Len(t, hashes, 1, "should only have one entry despite multiple adds")
				require.Equal(t, hash, hashes[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			dl, err := OpenDenyList(dir)
			require.NoError(t, err)
			defer dl.Close()

			tt.setup(t, dl)
			tt.check(t, dl)
		})
	}
}

func TestDenyList_Persistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		check func(t *testing.T, dir string)
	}{
		{
			name: "survives close and reopen",
			setup: func(t *testing.T, dir string) {
				dl, err := OpenDenyList(dir)
				require.NoError(t, err)

				hashes := []struct {
					height uint64
					hash   common.Hash
				}{
					{100, common.HexToHash("0x1111")},
					{100, common.HexToHash("0x2222")},
					{200, common.HexToHash("0x3333")},
					{300, common.HexToHash("0x4444")},
				}
				for _, h := range hashes {
					require.NoError(t, dl.Add(h.height, h.hash))
				}

				require.NoError(t, dl.Close())
			},
			check: func(t *testing.T, dir string) {
				dl, err := OpenDenyList(dir)
				require.NoError(t, err)
				defer dl.Close()

				// Verify all hashes are still present
				found, err := dl.Contains(100, common.HexToHash("0x1111"))
				require.NoError(t, err)
				require.True(t, found)

				found, err = dl.Contains(100, common.HexToHash("0x2222"))
				require.NoError(t, err)
				require.True(t, found)

				found, err = dl.Contains(200, common.HexToHash("0x3333"))
				require.NoError(t, err)
				require.True(t, found)

				found, err = dl.Contains(300, common.HexToHash("0x4444"))
				require.NoError(t, err)
				require.True(t, found)

				// Verify counts
				hashes100, err := dl.GetDeniedHashes(100)
				require.NoError(t, err)
				require.Len(t, hashes100, 2)

				hashes200, err := dl.GetDeniedHashes(200)
				require.NoError(t, err)
				require.Len(t, hashes200, 1)
			},
		},
		{
			name: "empty DB on fresh open",
			setup: func(t *testing.T, dir string) {
				// No setup - fresh directory
			},
			check: func(t *testing.T, dir string) {
				dl, err := OpenDenyList(dir)
				require.NoError(t, err)
				defer dl.Close()

				found, err := dl.Contains(100, common.HexToHash("0xabcd"))
				require.NoError(t, err)
				require.False(t, found, "fresh DB should not contain any hashes")

				hashes, err := dl.GetDeniedHashes(100)
				require.NoError(t, err)
				require.Empty(t, hashes, "fresh DB should return empty slice")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "denylist")

			tt.setup(t, dir)
			tt.check(t, dir)
		})
	}
}

func TestDenyList_GetDeniedHashes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, dl *DenyList)
		check func(t *testing.T, dl *DenyList)
	}{
		{
			name: "returns all hashes at height",
			setup: func(t *testing.T, dl *DenyList) {
				for i := 0; i < 5; i++ {
					hash := common.BigToHash(common.Big1.Add(common.Big1, common.Big0.SetInt64(int64(i))))
					require.NoError(t, dl.Add(100, hash))
				}
			},
			check: func(t *testing.T, dl *DenyList) {
				hashes, err := dl.GetDeniedHashes(100)
				require.NoError(t, err)
				require.Len(t, hashes, 5, "should return all 5 hashes")
			},
		},
		{
			name: "empty for clean height",
			setup: func(t *testing.T, dl *DenyList) {
				// Add hashes at other heights
				require.NoError(t, dl.Add(10, common.HexToHash("0xaaaa")))
				require.NoError(t, dl.Add(30, common.HexToHash("0xbbbb")))
			},
			check: func(t *testing.T, dl *DenyList) {
				hashes, err := dl.GetDeniedHashes(20)
				require.NoError(t, err)
				require.Empty(t, hashes, "height 20 should have no entries")
			},
		},
		{
			name: "isolated by height",
			setup: func(t *testing.T, dl *DenyList) {
				// Add different hashes at different heights
				require.NoError(t, dl.Add(10, common.HexToHash("0x1010")))
				require.NoError(t, dl.Add(10, common.HexToHash("0x1011")))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2020")))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2021")))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2022")))
				require.NoError(t, dl.Add(30, common.HexToHash("0x3030")))
			},
			check: func(t *testing.T, dl *DenyList) {
				hashes10, err := dl.GetDeniedHashes(10)
				require.NoError(t, err)
				require.Len(t, hashes10, 2, "height 10 should have 2 hashes")

				hashes20, err := dl.GetDeniedHashes(20)
				require.NoError(t, err)
				require.Len(t, hashes20, 3, "height 20 should have 3 hashes")

				hashes30, err := dl.GetDeniedHashes(30)
				require.NoError(t, err)
				require.Len(t, hashes30, 1, "height 30 should have 1 hash")

				// Verify specific hashes at height 20
				expected := map[common.Hash]bool{
					common.HexToHash("0x2020"): true,
					common.HexToHash("0x2021"): true,
					common.HexToHash("0x2022"): true,
				}
				for _, h := range hashes20 {
					require.True(t, expected[h], "unexpected hash at height 20: %s", h)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			dl, err := OpenDenyList(dir)
			require.NoError(t, err)
			defer dl.Close()

			tt.setup(t, dl)
			tt.check(t, dl)
		})
	}
}

// mockEngineForInvalidation implements engine_controller.EngineController for invalidation tests
type mockEngineForInvalidation struct {
	blockRef        eth.L2BlockRef
	rewindCalled    bool
	rewindTimestamp uint64
}

func (m *mockEngineForInvalidation) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	return nil, nil
}

func (m *mockEngineForInvalidation) RewindToTimestamp(ctx context.Context, timestamp uint64) error {
	m.rewindCalled = true
	m.rewindTimestamp = timestamp
	return nil
}

func (m *mockEngineForInvalidation) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}

func (m *mockEngineForInvalidation) Close() error {
	return nil
}

func (m *mockEngineForInvalidation) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return m.blockRef, nil
}

// mockVNForInvalidation implements virtual_node.VirtualNode for invalidation tests
type mockVNForInvalidation struct {
	stopErr error
}

func (m *mockVNForInvalidation) Start(ctx context.Context) error { return nil }
func (m *mockVNForInvalidation) Stop(ctx context.Context) error  { return m.stopErr }
func (m *mockVNForInvalidation) LatestSafe(ctx context.Context) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockVNForInvalidation) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockVNForInvalidation) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockVNForInvalidation) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{}, nil
}

var _ virtual_node.VirtualNode = (*mockVNForInvalidation)(nil)

func TestInvalidateBlock(t *testing.T) {
	t.Parallel()

	genesisTime := uint64(1000)
	blockTime := uint64(2)

	tests := []struct {
		name             string
		height           uint64
		payloadHash      common.Hash
		currentBlockHash common.Hash
		engineAvailable  bool
		expectRewind     bool
		expectRewindTs   uint64
	}{
		{
			name:             "current block matches triggers rewind",
			height:           5,
			payloadHash:      common.HexToHash("0xdead"),
			currentBlockHash: common.HexToHash("0xdead"), // Same hash
			engineAvailable:  true,
			expectRewind:     true,
			expectRewindTs:   genesisTime + (4 * blockTime), // height-1 timestamp
		},
		{
			name:             "current block differs no rewind",
			height:           5,
			payloadHash:      common.HexToHash("0xdead"),
			currentBlockHash: common.HexToHash("0xbeef"), // Different hash
			engineAvailable:  true,
			expectRewind:     false,
		},
		{
			name:            "engine unavailable adds to denylist only",
			height:          5,
			payloadHash:     common.HexToHash("0xdead"),
			engineAvailable: false,
			expectRewind:    false,
		},
		{
			name:             "rewind to height-1 timestamp calculated correctly",
			height:           10,
			payloadHash:      common.HexToHash("0xabcd"),
			currentBlockHash: common.HexToHash("0xabcd"),
			engineAvailable:  true,
			expectRewind:     true,
			expectRewindTs:   genesisTime + (9 * blockTime), // height 9
		},
	}

	// Separate test for genesis block (height=0) which should error
	t.Run("genesis block invalidation returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		dl, err := OpenDenyList(filepath.Join(dir, "denylist"))
		require.NoError(t, err)
		defer dl.Close()

		c := &simpleChainContainer{
			denyList: dl,
			log:      testLogger(),
		}

		ctx := context.Background()
		rewound, err := c.InvalidateBlock(ctx, 0, common.HexToHash("0xgenesis"))

		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot invalidate genesis block")
		require.False(t, rewound)

		// Genesis hash should NOT be added to denylist
		found, err := dl.Contains(0, common.HexToHash("0xgenesis"))
		require.NoError(t, err)
		require.False(t, found, "genesis block should not be added to denylist")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			// Create deny list
			dl, err := OpenDenyList(filepath.Join(dir, "denylist"))
			require.NoError(t, err)
			defer dl.Close()

			// Create mock engine
			mockEng := &mockEngineForInvalidation{
				blockRef: eth.L2BlockRef{Hash: tt.currentBlockHash},
			}

			// Create container with minimal config
			c := &simpleChainContainer{
				denyList: dl,
				log:      testLogger(),
				vncfg:    &opnodecfg.Config{},
				vn:       &mockVNForInvalidation{},
			}
			c.vncfg.Rollup.Genesis.L2Time = genesisTime
			c.vncfg.Rollup.BlockTime = blockTime

			if tt.engineAvailable {
				c.engine = mockEng
			}

			// Call InvalidateBlock
			ctx := context.Background()
			rewound, err := c.InvalidateBlock(ctx, tt.height, tt.payloadHash)
			require.NoError(t, err)

			// Verify rewind behavior
			require.Equal(t, tt.expectRewind, rewound, "rewind triggered mismatch")

			if tt.expectRewind && tt.engineAvailable {
				require.True(t, mockEng.rewindCalled, "RewindToTimestamp should have been called")
				require.Equal(t, tt.expectRewindTs, mockEng.rewindTimestamp, "rewind timestamp mismatch")
			}

			// Verify hash was added to denylist regardless
			found, err := dl.Contains(tt.height, tt.payloadHash)
			require.NoError(t, err)
			require.True(t, found, "hash should be in denylist after InvalidateBlock")
		})
	}
}

func TestIsDenied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupHash   common.Hash
		setupHeight uint64
		checkHash   common.Hash
		checkHeight uint64
		expectFound bool
	}{
		{
			name:        "denied block returns true",
			setupHash:   common.HexToHash("0x1234"),
			setupHeight: 100,
			checkHash:   common.HexToHash("0x1234"),
			checkHeight: 100,
			expectFound: true,
		},
		{
			name:        "non-denied returns false",
			setupHash:   common.HexToHash("0x1234"),
			setupHeight: 100,
			checkHash:   common.HexToHash("0x5678"), // Different hash
			checkHeight: 100,
			expectFound: false,
		},
		{
			name:        "wrong height returns false",
			setupHash:   common.HexToHash("0xabcd"),
			setupHeight: 10,
			checkHash:   common.HexToHash("0xabcd"), // Same hash
			checkHeight: 11,                         // Different height
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			dl, err := OpenDenyList(filepath.Join(dir, "denylist"))
			require.NoError(t, err)
			defer dl.Close()

			// Setup
			require.NoError(t, dl.Add(tt.setupHeight, tt.setupHash))

			// Create container
			c := &simpleChainContainer{
				denyList: dl,
				log:      testLogger(),
			}

			// Check
			found, err := c.IsDenied(tt.checkHeight, tt.checkHash)
			require.NoError(t, err)
			require.Equal(t, tt.expectFound, found)
		})
	}
}

func testLogger() gethlog.Logger {
	return gethlog.New()
}

// TestDenyList_ConcurrentAccess verifies the DenyList is safe for concurrent use.
// 10 goroutines each perform 100 Add and Contains operations simultaneously.
func TestDenyList_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	const numAccessors = 10
	const opsPerAccessor = 100

	// Helper to generate deterministic hash from accessor and op index
	makeHash := func(accessorID, opIdx int) common.Hash {
		var h common.Hash
		h[0] = byte(accessorID)
		h[1] = byte(opIdx)
		h[2] = byte(opIdx >> 8)
		return h
	}

	// Each accessor writes to its own height range and reads from all ranges
	var wg sync.WaitGroup
	wg.Add(numAccessors)

	for i := 0; i < numAccessors; i++ {
		go func(accessorID int) {
			defer wg.Done()

			baseHeight := uint64(accessorID * opsPerAccessor)

			for j := 0; j < opsPerAccessor; j++ {
				height := baseHeight + uint64(j)
				hash := makeHash(accessorID, j)

				// Write
				err := dl.Add(height, hash)
				require.NoError(t, err)

				// Read own write
				found, err := dl.Contains(height, hash)
				require.NoError(t, err)
				require.True(t, found, "accessor %d should find its own hash at height %d", accessorID, height)

				// Read from another accessor's range (may or may not exist yet)
				otherAccessor := (accessorID + 1) % numAccessors
				otherHeight := uint64(otherAccessor*opsPerAccessor) + uint64(j/2)
				_, err = dl.Contains(otherHeight, common.Hash{})
				require.NoError(t, err) // Should not error even if not found
			}
		}(i)
	}

	wg.Wait()

	// Verify final state: each accessor should have written opsPerAccessor hashes
	for i := 0; i < numAccessors; i++ {
		baseHeight := uint64(i * opsPerAccessor)
		for j := 0; j < opsPerAccessor; j++ {
			height := baseHeight + uint64(j)
			hash := makeHash(i, j)

			found, err := dl.Contains(height, hash)
			require.NoError(t, err)
			require.True(t, found, "hash from accessor %d at height %d should exist after concurrent access", i, height)
		}
	}
}
