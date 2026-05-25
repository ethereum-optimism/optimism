package chain_container

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum/go-ethereum/common"
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
				require.NoError(t, dl.Add(100, hash, 0, eth.Bytes32{}, eth.Bytes32{}))
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
					require.NoError(t, dl.Add(50, h, 0, eth.Bytes32{}, eth.Bytes32{}))
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
				require.NoError(t, dl.Add(10, hash, 0, eth.Bytes32{}, eth.Bytes32{}))
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
				require.NoError(t, dl.Add(200, hash, 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(200, hash, 0, eth.Bytes32{}, eth.Bytes32{})) // Add again
				require.NoError(t, dl.Add(200, hash, 0, eth.Bytes32{}, eth.Bytes32{})) // And again
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
					require.NoError(t, dl.Add(h.height, h.hash, 0, eth.Bytes32{}, eth.Bytes32{}))
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
					require.NoError(t, dl.Add(100, hash, 0, eth.Bytes32{}, eth.Bytes32{}))
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
				require.NoError(t, dl.Add(10, common.HexToHash("0xaaaa"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(30, common.HexToHash("0xbbbb"), 0, eth.Bytes32{}, eth.Bytes32{}))
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
				require.NoError(t, dl.Add(10, common.HexToHash("0x1010"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(10, common.HexToHash("0x1011"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2020"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2021"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(20, common.HexToHash("0x2022"), 0, eth.Bytes32{}, eth.Bytes32{}))
				require.NoError(t, dl.Add(30, common.HexToHash("0x3030"), 0, eth.Bytes32{}, eth.Bytes32{}))
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
func (m *mockVNForInvalidation) FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockVNForInvalidation) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{}, nil
}

var _ virtual_node.VirtualNode = (*mockVNForInvalidation)(nil)

func TestRecordInvalidation(t *testing.T) {
	t.Parallel()

	stateRoot := eth.Bytes32(common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"))
	msgPasserRoot := eth.Bytes32(common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"))
	payloadHash := common.HexToHash("0xbad")

	newContainer := func(t *testing.T) (*simpleChainContainer, *DenyList) {
		t.Helper()
		dir := t.TempDir()
		dl, err := OpenDenyList(filepath.Join(dir, "denylist"))
		require.NoError(t, err)
		t.Cleanup(func() { dl.Close() })
		return &simpleChainContainer{
			denyList: dl,
			log:      testLogger(),
		}, dl
	}

	t.Run("happy path persists deny-list entry with preimages", func(t *testing.T) {
		t.Parallel()
		c, dl := newContainer(t)

		err := c.RecordInvalidation(context.Background(), 5, payloadHash, 100, stateRoot, msgPasserRoot)
		require.NoError(t, err)

		found, err := dl.Contains(5, payloadHash)
		require.NoError(t, err)
		require.True(t, found)

		stored, err := dl.GetOutputV0(5, payloadHash)
		require.NoError(t, err)
		require.Equal(t, &eth.OutputV0{
			StateRoot:                stateRoot,
			MessagePasserStorageRoot: msgPasserRoot,
			BlockHash:                payloadHash,
		}, stored)
	})

	t.Run("genesis block (height 0) rejected", func(t *testing.T) {
		t.Parallel()
		c, dl := newContainer(t)

		err := c.RecordInvalidation(context.Background(), 0, payloadHash, 0, stateRoot, msgPasserRoot)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot invalidate genesis block")

		found, err := dl.Contains(0, payloadHash)
		require.NoError(t, err)
		require.False(t, found, "genesis must not be persisted")
	})

	t.Run("zero state root rejected", func(t *testing.T) {
		t.Parallel()
		c, dl := newContainer(t)

		err := c.RecordInvalidation(context.Background(), 5, payloadHash, 0, eth.Bytes32{}, msgPasserRoot)
		require.Error(t, err)
		require.Contains(t, err.Error(), "zero state root")

		found, err := dl.Contains(5, payloadHash)
		require.NoError(t, err)
		require.False(t, found, "must not persist when preimages are missing")
	})

	t.Run("zero message-passer storage root rejected", func(t *testing.T) {
		t.Parallel()
		c, dl := newContainer(t)

		err := c.RecordInvalidation(context.Background(), 5, payloadHash, 0, stateRoot, eth.Bytes32{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "zero message-passer storage root")

		found, err := dl.Contains(5, payloadHash)
		require.NoError(t, err)
		require.False(t, found, "must not persist when preimages are missing")
	})

	t.Run("nil deny list returns error", func(t *testing.T) {
		t.Parallel()
		c := &simpleChainContainer{log: testLogger()}
		err := c.RecordInvalidation(context.Background(), 5, payloadHash, 0, stateRoot, msgPasserRoot)
		require.Error(t, err)
		require.Contains(t, err.Error(), "deny list not initialized")
	})

	t.Run("does not depend on engine — no rewind, no engine RPC", func(t *testing.T) {
		t.Parallel()
		c, _ := newContainer(t)
		// Explicitly leave c.engine and c.vn as nil. A successful call here
		// is the regression guard: today's rewind branch required both.
		require.NoError(t, c.RecordInvalidation(context.Background(), 7, payloadHash, 0, stateRoot, msgPasserRoot))
	})
}

func TestGetDeniedOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dl, err := OpenDenyList(filepath.Join(dir, "denylist"))
	require.NoError(t, err)
	defer dl.Close()

	testStateRoot := eth.Bytes32(common.HexToHash("0xstate"))
	testMsgPasserRoot := eth.Bytes32(common.HexToHash("0xmsgpasser"))
	hash := common.HexToHash("0xdead")

	c := &simpleChainContainer{
		denyList: dl,
		log:      testLogger(),
	}

	t.Run("returns output after InvalidateBlock", func(t *testing.T) {
		require.NoError(t, dl.Add(100, hash, 0, testStateRoot, testMsgPasserRoot))

		got, err := c.GetDeniedOutput(100, hash)
		require.NoError(t, err)
		want := &eth.OutputV0{
			StateRoot:                testStateRoot,
			MessagePasserStorageRoot: testMsgPasserRoot,
			BlockHash:                hash,
		}
		require.Equal(t, want, got)
	})

	t.Run("returns nil for non-denied block", func(t *testing.T) {
		got, err := c.GetDeniedOutput(999, common.HexToHash("0xunknown"))
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns error when denylist not initialized", func(t *testing.T) {
		noDenyList := &simpleChainContainer{
			log: testLogger(),
		}
		_, err := noDenyList.GetDeniedOutput(100, hash)
		require.Error(t, err)
	})
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
			require.NoError(t, dl.Add(tt.setupHeight, tt.setupHash, 0, eth.Bytes32{}, eth.Bytes32{}))

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

func TestDenyList_CanonicalDeniedHeight(t *testing.T) {
	t.Parallel()

	canonicalAt := func(m map[uint64]common.Hash) func(context.Context, uint64) (common.Hash, error) {
		return func(_ context.Context, h uint64) (common.Hash, error) {
			hash, ok := m[h]
			if !ok {
				return common.Hash{}, errors.New("no canonical at height")
			}
			return hash, nil
		}
	}

	t.Run("empty deny list returns not found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canonicalAt(map[uint64]common.Hash{}))
		require.NoError(t, err)
		require.False(t, found)
		require.Zero(t, height)
	})

	t.Run("single canonical denied returns that height", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		bad := common.HexToHash("0xbad")
		require.NoError(t, dl.Add(100, bad, 0, eth.Bytes32{}, eth.Bytes32{}))

		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canonicalAt(map[uint64]common.Hash{100: bad}))
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(100), height)
	})

	t.Run("denied but not canonical returns not found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		bad := common.HexToHash("0xbad")
		good := common.HexToHash("0xgood")
		require.NoError(t, dl.Add(100, bad, 0, eth.Bytes32{}, eth.Bytes32{}))

		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canonicalAt(map[uint64]common.Hash{100: good}))
		require.NoError(t, err)
		require.False(t, found)
		require.Zero(t, height)
	})

	t.Run("multiple denied heights all canonical returns lowest", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		h50 := common.HexToHash("0x50bad")
		h100 := common.HexToHash("0x100bad")
		h150 := common.HexToHash("0x150bad")
		require.NoError(t, dl.Add(50, h50, 0, eth.Bytes32{}, eth.Bytes32{}))
		require.NoError(t, dl.Add(100, h100, 0, eth.Bytes32{}, eth.Bytes32{}))
		require.NoError(t, dl.Add(150, h150, 0, eth.Bytes32{}, eth.Bytes32{}))

		canon := canonicalAt(map[uint64]common.Hash{50: h50, 100: h100, 150: h150})
		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canon)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(50), height, "lowest still-canonical denied height wins")
	})

	t.Run("highest non-canonical stops iteration (returns not found)", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		h50 := common.HexToHash("0x50bad")
		h150 := common.HexToHash("0x150bad")
		require.NoError(t, dl.Add(50, h50, 0, eth.Bytes32{}, eth.Bytes32{}))
		require.NoError(t, dl.Add(150, h150, 0, eth.Bytes32{}, eth.Bytes32{}))

		// 150 is no longer canonical; 50 still is. By the monotonicity contract
		// we stop at 150 and report not found.
		canon := canonicalAt(map[uint64]common.Hash{50: h50, 150: common.HexToHash("0xreorged")})
		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canon)
		require.NoError(t, err)
		require.False(t, found)
		require.Zero(t, height)
	})

	t.Run("highest canonical lower non-canonical returns highest", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		h50 := common.HexToHash("0x50bad")
		h150 := common.HexToHash("0x150bad")
		require.NoError(t, dl.Add(50, h50, 0, eth.Bytes32{}, eth.Bytes32{}))
		require.NoError(t, dl.Add(150, h150, 0, eth.Bytes32{}, eth.Bytes32{}))

		// 150 canonical, 50 reorged out (replaced by a non-denied hash). Stop at 50,
		// return 150 as the lowest canonical found.
		canon := canonicalAt(map[uint64]common.Hash{50: common.HexToHash("0xnew50"), 150: h150})
		height, found, err := dl.CanonicalDeniedHeight(context.Background(), canon)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(150), height)
	})

	t.Run("canonical probe error stops iteration after recording prior matches", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		h100 := common.HexToHash("0x100bad")
		h200 := common.HexToHash("0x200bad")
		require.NoError(t, dl.Add(100, h100, 0, eth.Bytes32{}, eth.Bytes32{}))
		require.NoError(t, dl.Add(200, h200, 0, eth.Bytes32{}, eth.Bytes32{}))

		// 200 canonical; probe for 100 fails (e.g. EL not yet at that height).
		// We record 200 then stop on the error and return what we have.
		probe := func(_ context.Context, h uint64) (common.Hash, error) {
			switch h {
			case 200:
				return h200, nil
			default:
				return common.Hash{}, errors.New("EL not synced")
			}
		}
		height, found, err := dl.CanonicalDeniedHeight(context.Background(), probe)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), height)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dl, err := OpenDenyList(dir)
		require.NoError(t, err)
		defer dl.Close()

		require.NoError(t, dl.Add(100, common.HexToHash("0xbad"), 0, eth.Bytes32{}, eth.Bytes32{}))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err = dl.CanonicalDeniedHeight(ctx, canonicalAt(map[uint64]common.Hash{100: common.HexToHash("0xbad")}))
		require.ErrorIs(t, err, context.Canceled)
	})
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
				err := dl.Add(height, hash, 0, eth.Bytes32{}, eth.Bytes32{})
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

func TestDenyList_PruneAtOrAfterTimestamp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	hashA := common.HexToHash("0xaaaa")
	hashB := common.HexToHash("0xbbbb")
	hashC := common.HexToHash("0xcccc")
	require.NoError(t, dl.Add(100, hashA, 10, eth.Bytes32{}, eth.Bytes32{}))
	require.NoError(t, dl.Add(100, hashB, 11, eth.Bytes32{}, eth.Bytes32{}))
	require.NoError(t, dl.Add(200, hashC, 12, eth.Bytes32{}, eth.Bytes32{}))

	removed, err := dl.PruneAtOrAfterTimestamp(11)
	require.NoError(t, err)
	require.Equal(t, map[uint64][]common.Hash{
		100: {hashB},
		200: {hashC},
	}, removed)

	// hashA at ts=10 should survive
	records100, err := dl.GetDeniedRecords(100)
	require.NoError(t, err)
	require.Len(t, records100, 1)
	require.Equal(t, hashA, records100[0].PayloadHash)
	require.Equal(t, uint64(10), records100[0].DecisionTimestamp)

	// height 200 should be empty
	records200, err := dl.GetDeniedRecords(200)
	require.NoError(t, err)
	require.Empty(t, records200)
}

func TestDenyList_HasDeniedAtOrAfterTimestamp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	require.NoError(t, dl.Add(100, common.HexToHash("0xaaaa"), 10, eth.Bytes32{}, eth.Bytes32{}))
	require.NoError(t, dl.Add(50, common.HexToHash("0xbbbb"), 12, eth.Bytes32{}, eth.Bytes32{}))
	require.NoError(t, dl.Add(25, common.HexToHash("0xcccc"), 9, eth.Bytes32{}, eth.Bytes32{}))

	ok, err := dl.HasDeniedAtOrAfterTimestamp(11)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = dl.HasDeniedAtOrAfterTimestamp(13)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDenyList_GetOutputV0(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	out := &eth.OutputV0{
		StateRoot:                eth.Bytes32(common.HexToHash("0xstate")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsgpasser")),
		BlockHash:                common.HexToHash("0xblock"),
	}
	hash := common.HexToHash("0x1111")
	require.NoError(t, dl.Add(100, hash, 0, out.StateRoot, out.MessagePasserStorageRoot))

	want := &eth.OutputV0{
		StateRoot:                out.StateRoot,
		MessagePasserStorageRoot: out.MessagePasserStorageRoot,
		BlockHash:                hash,
	}

	t.Run("returns stored OutputV0 for matching hash and height", func(t *testing.T) {
		got, err := dl.GetOutputV0(100, hash)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("returns nil for wrong hash at same height", func(t *testing.T) {
		got, err := dl.GetOutputV0(100, common.HexToHash("0x9999"))
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns nil for correct hash at wrong height", func(t *testing.T) {
		got, err := dl.GetOutputV0(200, hash)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns nil for empty height", func(t *testing.T) {
		got, err := dl.GetOutputV0(999, common.HexToHash("0xabcd"))
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("isolates outputs per hash at same height", func(t *testing.T) {
		out2 := &eth.OutputV0{
			StateRoot:                eth.Bytes32(common.HexToHash("0xstate2")),
			MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsgpasser2")),
			BlockHash:                common.HexToHash("0xblock2"),
		}
		hash2 := common.HexToHash("0x2222")
		require.NoError(t, dl.Add(100, hash2, 0, out2.StateRoot, out2.MessagePasserStorageRoot))

		got1, err := dl.GetOutputV0(100, hash)
		require.NoError(t, err)
		require.Equal(t, want, got1)

		want2 := &eth.OutputV0{
			StateRoot:                out2.StateRoot,
			MessagePasserStorageRoot: out2.MessagePasserStorageRoot,
			BlockHash:                hash2,
		}
		got2, err := dl.GetOutputV0(100, hash2)
		require.NoError(t, err)
		require.Equal(t, want2, got2)
	})
}

func TestDenyList_LastDeniedOutputV0(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	t.Run("returns nil for empty height", func(t *testing.T) {
		got, err := dl.LastDeniedOutputV0(999)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns the single record", func(t *testing.T) {
		hash := common.HexToHash("0xaaaa")
		stateRoot := eth.Bytes32(common.HexToHash("0xstate1"))
		msgPasser := eth.Bytes32(common.HexToHash("0xmp1"))
		require.NoError(t, dl.Add(50, hash, 100, stateRoot, msgPasser))

		got, err := dl.LastDeniedOutputV0(50)
		require.NoError(t, err)
		require.Equal(t, &eth.OutputV0{
			StateRoot:                stateRoot,
			MessagePasserStorageRoot: msgPasser,
			BlockHash:                hash,
		}, got)
	})

	t.Run("returns the last record when multiple exist", func(t *testing.T) {
		firstHash := common.HexToHash("0xbbbb")
		require.NoError(t, dl.Add(60, firstHash, 100, eth.Bytes32{0x01}, eth.Bytes32{0x02}))

		lastHash := common.HexToHash("0xcccc")
		lastState := eth.Bytes32(common.HexToHash("0xlast_state"))
		lastMsgPasser := eth.Bytes32(common.HexToHash("0xlast_mp"))
		require.NoError(t, dl.Add(60, lastHash, 200, lastState, lastMsgPasser))

		got, err := dl.LastDeniedOutputV0(60)
		require.NoError(t, err)
		require.Equal(t, &eth.OutputV0{
			StateRoot:                lastState,
			MessagePasserStorageRoot: lastMsgPasser,
			BlockHash:                lastHash,
		}, got)
	})
}

func TestDenyList_OutputPersistence(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "denylist")

	out := &eth.OutputV0{
		StateRoot:                eth.Bytes32(common.HexToHash("0xpersist_state")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xpersist_msgpasser")),
		BlockHash:                common.HexToHash("0xpersist_block"),
	}
	hash := common.HexToHash("0xdead")
	want := &eth.OutputV0{
		StateRoot:                out.StateRoot,
		MessagePasserStorageRoot: out.MessagePasserStorageRoot,
		BlockHash:                hash,
	}

	// Write and close
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	require.NoError(t, dl.Add(100, hash, 42, out.StateRoot, out.MessagePasserStorageRoot))
	require.NoError(t, dl.Close())

	// Reopen and verify output persisted
	dl2, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl2.Close()

	got, err := dl2.GetOutputV0(100, hash)
	require.NoError(t, err)
	require.Equal(t, want, got)

	records, err := dl2.GetDeniedRecords(100)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, hash, records[0].PayloadHash)
	require.Equal(t, uint64(42), records[0].DecisionTimestamp)
	require.Equal(t, out.StateRoot, records[0].StateRoot)
	require.Equal(t, out.MessagePasserStorageRoot, records[0].MessagePasserStorageRoot)
}

func TestDenyList_GetDeniedRecords_IncludesRoots(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dl, err := OpenDenyList(dir)
	require.NoError(t, err)
	defer dl.Close()

	out1 := &eth.OutputV0{
		StateRoot:                eth.Bytes32(common.HexToHash("0xstate1")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsg1")),
		BlockHash:                common.HexToHash("0xblock1"),
	}
	out2 := &eth.OutputV0{
		StateRoot:                eth.Bytes32(common.HexToHash("0xstate2")),
		MessagePasserStorageRoot: eth.Bytes32(common.HexToHash("0xmsg2")),
		BlockHash:                common.HexToHash("0xblock2"),
	}
	hash1 := common.HexToHash("0xaaaa")
	hash2 := common.HexToHash("0xbbbb")

	require.NoError(t, dl.Add(100, hash1, 10, out1.StateRoot, out1.MessagePasserStorageRoot))
	require.NoError(t, dl.Add(100, hash2, 11, out2.StateRoot, out2.MessagePasserStorageRoot))

	records, err := dl.GetDeniedRecords(100)
	require.NoError(t, err)
	require.Len(t, records, 2)

	recordMap := make(map[common.Hash]DenyRecord)
	for _, r := range records {
		recordMap[r.PayloadHash] = r
	}

	require.Equal(t, out1.StateRoot, recordMap[hash1].StateRoot)
	require.Equal(t, out1.MessagePasserStorageRoot, recordMap[hash1].MessagePasserStorageRoot)
	require.Equal(t, uint64(10), recordMap[hash1].DecisionTimestamp)
	require.Equal(t, out2.StateRoot, recordMap[hash2].StateRoot)
	require.Equal(t, out2.MessagePasserStorageRoot, recordMap[hash2].MessagePasserStorageRoot)
	require.Equal(t, uint64(11), recordMap[hash2].DecisionTimestamp)
}
