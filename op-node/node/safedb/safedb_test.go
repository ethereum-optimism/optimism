package safedb

import (
	"context"
	"math"
	"math/rand"
	"slices"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStoreSafeHeads(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()
	l2a := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xaa},
		Number: 20,
	}
	l2b := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xbb},
		Number: 25,
	}
	l1a := eth.BlockID{
		Hash:   common.Hash{0x01, 0xaa},
		Number: 100,
	}
	l1b := eth.BlockID{
		Hash:   common.Hash{0x01, 0xbb},
		Number: 150,
	}
	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))

	verifySafeHeads := func(db *SafeDB) {
		_, _, err = db.SafeHeadAtL1(context.Background(), l1a.Number-1)
		require.ErrorIs(t, err, ErrNotFound)

		actualL1, actualL2, err := db.SafeHeadAtL1(context.Background(), l1a.Number)
		require.NoError(t, err)
		require.Equal(t, l1a, actualL1)
		require.Equal(t, l2a.ID(), actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1a.Number+1)
		require.NoError(t, err)
		require.Equal(t, l1a, actualL1)
		require.Equal(t, l2a.ID(), actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number)
		require.NoError(t, err)
		require.Equal(t, l1b, actualL1)
		require.Equal(t, l2b.ID(), actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number+1)
		require.NoError(t, err)
		require.Equal(t, l1b, actualL1)
		require.Equal(t, l2b.ID(), actualL2)
	}
	// Verify loading the safe heads with the already open DB
	verifySafeHeads(db)

	// Close the DB and open a new instance
	require.NoError(t, db.Close())
	newDB, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	// Verify the data is reloaded correctly
	verifySafeHeads(newDB)
}

func TestSafeHeadAtL1_EmptyDatabase(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()
	_, _, err = db.SafeHeadAtL1(context.Background(), 100)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestFirstEntry_EmptyDatabase(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()
	_, _, err = db.FirstEntry(context.Background())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestFirstEntry_ReturnsLowestL1(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l2a := eth.L2BlockRef{Hash: common.Hash{0x02, 0xaa}, Number: 20}
	l2b := eth.L2BlockRef{Hash: common.Hash{0x02, 0xbb}, Number: 25}
	l1a := eth.BlockID{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l1b := eth.BlockID{Hash: common.Hash{0x01, 0xbb}, Number: 150}

	// Insert out of order to confirm we return the lowest L1 block, not the
	// first-inserted entry.
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))
	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))

	actualL1, actualL2, err := db.FirstEntry(context.Background())
	require.NoError(t, err)
	require.Equal(t, l1a, actualL1)
	require.Equal(t, l2a.ID(), actualL2)
}

func TestFirstEntry_StableAfterResetAhead(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l1a := eth.BlockID{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l1b := eth.BlockID{Hash: common.Hash{0x01, 0xbb}, Number: 150}
	l2a := eth.L2BlockRef{Hash: common.Hash{0x02, 0xaa}, Number: 20, L1Origin: l1a}
	l2b := eth.L2BlockRef{Hash: common.Hash{0x02, 0xbb}, Number: 25, L1Origin: l1b}

	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))

	// Reset to l2b truncates entries at or after l2b; the l2a entry remains
	// and must still be the first.
	require.NoError(t, db.SafeHeadReset(l2b))

	actualL1, actualL2, err := db.FirstEntry(context.Background())
	require.NoError(t, err)
	require.Equal(t, l1a, actualL1)
	require.Equal(t, l2a.ID(), actualL2)
}

func TestTruncateOnSafeHeadReset(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l2a := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xaa},
		Number: 20,
		L1Origin: eth.BlockID{
			Number: 60,
		},
	}
	l2b := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xbb},
		Number: 22,
		L1Origin: eth.BlockID{
			Number: 90,
		},
	}
	l2c := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 25,
		L1Origin: eth.BlockID{
			Number: 110,
		},
	}
	l2d := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 30,
		L1Origin: eth.BlockID{
			Number: 120,
		},
	}
	l1a := eth.BlockID{
		Hash:   common.Hash{0x01, 0xaa},
		Number: 100,
	}
	l1b := eth.BlockID{
		Hash:   common.Hash{0x01, 0xbb},
		Number: 150,
	}
	l1c := eth.BlockID{
		Hash:   common.Hash{0x01, 0xcc},
		Number: 160,
	}

	// Add some entries
	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2c, l1b))
	require.NoError(t, db.SafeHeadUpdated(l2d, l1c))

	// Then reset to between the two existing entries
	require.NoError(t, db.SafeHeadReset(l2b))

	// Only the reset safe head is now safe at the previous L1 block number
	actualL1, actualL2, err := db.SafeHeadAtL1(context.Background(), l1b.Number)
	require.NoError(t, err)
	require.Equal(t, l1b, actualL1)
	require.Equal(t, l2b.ID(), actualL2)

	actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1c.Number)
	require.NoError(t, err)
	require.Equal(t, l1b, actualL1)
	require.Equal(t, l2b.ID(), actualL2)

	// l2a is still safe from its original update
	actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1a.Number)
	require.NoError(t, err)
	require.Equal(t, l1a, actualL1)
	require.Equal(t, l2a.ID(), actualL2)
}

func TestTruncateOnSafeHeadReset_BeforeFirstEntry(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l2b := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xbb},
		Number: 22,
		L1Origin: eth.BlockID{
			Number: 90,
		},
	}
	l2c := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 25,
		L1Origin: eth.BlockID{
			Number: 110,
		},
	}
	l2d := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 30,
		L1Origin: eth.BlockID{
			Number: 120,
		},
	}
	l1a := eth.BlockID{
		Hash:   common.Hash{0x01, 0xaa},
		Number: 100,
	}
	l1b := eth.BlockID{
		Hash:   common.Hash{0x01, 0xbb},
		Number: 150,
	}
	l1c := eth.BlockID{
		Hash:   common.Hash{0x01, 0xcc},
		Number: 160,
	}

	// Add some entries
	require.NoError(t, db.SafeHeadUpdated(l2c, l1b))
	require.NoError(t, db.SafeHeadUpdated(l2d, l1c))

	// Then reset to between the two existing entries
	require.NoError(t, db.SafeHeadReset(l2b))

	// All entries got removed
	_, _, err = db.SafeHeadAtL1(context.Background(), l1a.Number)
	require.ErrorIs(t, err, ErrNotFound)
	_, _, err = db.SafeHeadAtL1(context.Background(), l1b.Number)
	require.ErrorIs(t, err, ErrNotFound)
	_, _, err = db.SafeHeadAtL1(context.Background(), l1c.Number)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestTruncateOnSafeHeadReset_AfterLastEntry(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l2a := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xaa},
		Number: 20,
		L1Origin: eth.BlockID{
			Number: 60,
		},
	}
	l2b := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xbb},
		Number: 22,
		L1Origin: eth.BlockID{
			Number: 90,
		},
	}
	l2c := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 25,
		L1Origin: eth.BlockID{
			Number: 110,
		},
	}
	l1a := eth.BlockID{
		Hash:   common.Hash{0x01, 0xaa},
		Number: 100,
	}
	l1b := eth.BlockID{
		Hash:   common.Hash{0x01, 0xbb},
		Number: 150,
	}
	l1c := eth.BlockID{
		Hash:   common.Hash{0x01, 0xcc},
		Number: 160,
	}

	// Add some entries
	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))
	require.NoError(t, db.SafeHeadUpdated(l2c, l1c))

	verifySafeHeads := func() {
		// Everything is still safe
		actualL1, actualL2, err := db.SafeHeadAtL1(context.Background(), l1a.Number)
		require.NoError(t, err)
		require.Equal(t, l1a, actualL1)
		require.Equal(t, l2a.ID(), actualL2)

		// Everything is still safe
		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number)
		require.NoError(t, err)
		require.Equal(t, l1b, actualL1)
		require.Equal(t, l2b.ID(), actualL2)

		// Everything is still safe
		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1c.Number)
		require.NoError(t, err)
		require.Equal(t, l1c, actualL1)
		require.Equal(t, l2c.ID(), actualL2)
	}
	verifySafeHeads()

	// Then reset to an L2 block after all entries with an origin after all L1 entries
	require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xdd},
		Number: 30,
		L1Origin: eth.BlockID{
			Number: l1c.Number + 1,
		},
	}))
	verifySafeHeads()

	// Then reset to an L2 block after all entries with an origin before some L1 entries
	require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xdd},
		Number: 30,
		L1Origin: eth.BlockID{
			Number: l1b.Number - 1,
		},
	}))
	verifySafeHeads()
}

func TestL1AtSafeHead(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()

	l1a := eth.BlockID{Hash: common.Hash{0x01, 0xaa}, Number: 100}
	l1b := eth.BlockID{Hash: common.Hash{0x01, 0xbb}, Number: 110}
	l1c := eth.BlockID{Hash: common.Hash{0x01, 0xcc}, Number: 120}
	l2a := eth.L2BlockRef{Hash: common.Hash{0x02, 0xaa}, Number: 500}
	l2b := eth.L2BlockRef{Hash: common.Hash{0x02, 0xbb}, Number: 510}
	l2c := eth.L2BlockRef{Hash: common.Hash{0x02, 0xcc}, Number: 520}

	t.Run("EmptyDB", func(t *testing.T) {
		_, _, err := db.L1AtSafeHead(context.Background(), 500)
		require.ErrorIs(t, err, ErrL1AtSafeHeadNotFound)
	})

	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))
	require.NoError(t, db.SafeHeadUpdated(l2c, l1c))

	t.Run("TargetEqualsFirstL2", func(t *testing.T) {
		// The first entry's L1 is a real recorded transition, so target ==
		// firstL2 must resolve to firstL1 without a +1 workaround.
		l1, l2, err := db.L1AtSafeHead(context.Background(), l2a.Number)
		require.NoError(t, err)
		require.Equal(t, l1a, l1)
		require.Equal(t, l2a.ID(), l2)
	})

	t.Run("TargetBetweenEntries", func(t *testing.T) {
		l1, l2, err := db.L1AtSafeHead(context.Background(), 505)
		require.NoError(t, err)
		require.Equal(t, l1b, l1)
		require.Equal(t, l2b.ID(), l2)
	})

	t.Run("TargetEqualsRecordedL2", func(t *testing.T) {
		l1, l2, err := db.L1AtSafeHead(context.Background(), l2b.Number)
		require.NoError(t, err)
		require.Equal(t, l1b, l1)
		require.Equal(t, l2b.ID(), l2)
	})

	t.Run("TargetEqualsLatestL2", func(t *testing.T) {
		l1, l2, err := db.L1AtSafeHead(context.Background(), l2c.Number)
		require.NoError(t, err)
		require.Equal(t, l1c, l1)
		require.Equal(t, l2c.ID(), l2)
	})

	t.Run("TargetAboveLatest", func(t *testing.T) {
		_, _, err := db.L1AtSafeHead(context.Background(), l2c.Number+1)
		require.ErrorIs(t, err, ErrL1AtSafeHeadNotFound)
	})

	t.Run("TargetBelowFirst", func(t *testing.T) {
		_, _, err := db.L1AtSafeHead(context.Background(), l2a.Number-1)
		require.ErrorIs(t, err, ErrL1AtSafeHeadUnavailable)
	})
}

// l1AtSafeHeadScan is the backwards linear scan L1AtSafeHead used before it
// became a binary search. It is the reference for TestL1AtSafeHead_MatchesScan.
func l1AtSafeHeadScan(d *SafeDB, targetL2Num uint64) (l1 eth.BlockID, safeHead eth.BlockID, err error) {
	iter, err := d.db.NewIter(safeByL1BlockNumKey.IterRange())
	if err != nil {
		return
	}
	defer iter.Close()
	if valid := iter.Last(); !valid {
		err = ErrL1AtSafeHeadNotFound
		return
	}
	cursorL1, cursorL2, err := decodeEntry(iter)
	if err != nil {
		return
	}
	if targetL2Num > cursorL2.Number {
		err = ErrL1AtSafeHeadNotFound
		return
	}
	for iter.Prev() {
		prevL1, prevL2, derr := decodeEntry(iter)
		if derr != nil {
			err = derr
			return
		}
		if prevL2.Number < targetL2Num {
			return cursorL1, cursorL2, nil
		}
		cursorL1, cursorL2 = prevL1, prevL2
	}
	if cursorL2.Number == targetL2Num {
		return cursorL1, cursorL2, nil
	}
	err = ErrL1AtSafeHeadUnavailable
	return
}

func TestL1AtSafeHead_MatchesScan(t *testing.T) {
	for _, tc := range []struct {
		name      string
		entries   int
		maxL1Gap  uint64 // L1 numbers are sparse: not every L1 block moves the safe head
		maxL2Step uint64 // 0 steps leave the same safe head recorded at several L1 blocks
	}{
		{name: "single", entries: 1, maxL1Gap: 1, maxL2Step: 1},
		{name: "two", entries: 2, maxL1Gap: 3, maxL2Step: 3},
		{name: "dense", entries: 300, maxL1Gap: 1, maxL2Step: 1},
		{name: "sparse", entries: 300, maxL1Gap: 50, maxL2Step: 8},
		{name: "repeated-safe-heads", entries: 300, maxL1Gap: 4, maxL2Step: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := NewSafeDB(testlog.Logger(t, log.LevelError), t.TempDir())
			require.NoError(t, err)
			defer db.Close()
			db.writeOpts = pebble.NoSync

			rng := rand.New(rand.NewSource(int64(tc.entries)*1000 + int64(tc.maxL1Gap)))
			l1Num, l2Num := uint64(1000), uint64(5000)
			firstL2 := l2Num
			type entry struct{ l1, l2 uint64 }
			var entries []entry
			add := func(n int) {
				for i := 0; i < n; i++ {
					id := len(entries)
					l1 := eth.BlockID{Hash: common.Hash{0x01, byte(id), byte(id >> 8)}, Number: l1Num}
					l2 := eth.L2BlockRef{Hash: common.Hash{0x02, byte(id), byte(id >> 8)}, Number: l2Num}
					require.NoError(t, db.SafeHeadUpdated(l2, l1))
					entries = append(entries, entry{l1: l1Num, l2: l2Num})
					l1Num += 1 + uint64(rng.Int63n(int64(tc.maxL1Gap)))
					l2Num += uint64(rng.Int63n(int64(tc.maxL2Step) + 1))
				}
			}
			compare := func() {
				for target := firstL2 - 3; target <= l2Num+3; target++ {
					wantL1, wantL2, wantErr := l1AtSafeHeadScan(db, target)
					gotL1, gotL2, gotErr := db.L1AtSafeHead(context.Background(), target)
					require.Equal(t, wantErr, gotErr, "target %d", target)
					require.Equal(t, wantL1, gotL1, "target %d", target)
					require.Equal(t, wantL2, gotL2, "target %d", target)
				}
			}

			add(tc.entries)
			compare()

			// Reset the safe head to the middle of the recorded history, then record on top of it.
			mid := entries[len(entries)/2]
			require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{
				Hash:     common.Hash{0x03},
				Number:   mid.l2,
				L1Origin: eth.BlockID{Number: mid.l1 - 1},
			}))
			compare()
			l1Num, l2Num = mid.l1+1, mid.l2+1
			add(tc.entries)
			compare()
		})
	}
}

func TestL1AtSafeHead_Disabled(t *testing.T) {
	_, _, err := Disabled.L1AtSafeHead(context.Background(), 500)
	require.ErrorIs(t, err, ErrNotEnabled)
}

func TestKeysFollowNaturalByteOrdering(t *testing.T) {
	vals := []uint64{0, 1, math.MaxUint32 - 1, math.MaxUint32, math.MaxUint32 + 1, math.MaxUint64 - 1, math.MaxUint64}
	for i := 1; i < len(vals); i++ {
		prev := safeByL1BlockNumKey.Of(vals[i-1])
		cur := safeByL1BlockNumKey.Of(vals[i])
		require.True(t, slices.Compare(prev, cur) < 0, "Expected %v key %x to be less than %v key %x", vals[i-1], prev, vals[i], cur)
	}
}

func TestDecodeSafeByL1BlockNum(t *testing.T) {
	l1 := eth.BlockID{
		Hash:   common.Hash{0x01},
		Number: 84298,
	}
	l2 := eth.BlockID{
		Hash:   common.Hash{0x02},
		Number: 3224,
	}
	validKey := safeByL1BlockNumKey.Of(l1.Number)
	validValue := safeByL1BlockNumValue(l1, l2)

	t.Run("Roundtrip", func(t *testing.T) {
		actualL1, actualL2, err := decodeSafeByL1BlockNum(validKey, validValue)
		require.NoError(t, err)
		require.Equal(t, l1, actualL1)
		require.Equal(t, l2, actualL2)
	})

	t.Run("ErrorOnEmptyKey", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum([]byte{}, validValue)
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnTooShortKey", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum([]byte{1, 2, 3, 4}, validValue)
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnTooLongKey", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum(append(validKey, 2), validValue)
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnWrongKeyPrefix", func(t *testing.T) {
		invalidKey := slices.Clone(validKey)
		invalidKey[0] = 49
		_, _, err := decodeSafeByL1BlockNum(invalidKey, validValue)
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnEmptyValue", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum(validKey, []byte{})
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnTooShortValue", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum(validKey, []byte{1, 2, 3, 4})
		require.ErrorIs(t, err, ErrInvalidEntry)
	})

	t.Run("ErrorOnTooLongValue", func(t *testing.T) {
		_, _, err := decodeSafeByL1BlockNum(validKey, append(validKey, 2))
		require.ErrorIs(t, err, ErrInvalidEntry)
	})
}
