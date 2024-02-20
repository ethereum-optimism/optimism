package safedb

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestStoreSafeHeads(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
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
	logger := testlog.Logger(t, log.LvlInfo)
	dir := t.TempDir()
	db, err := NewSafeDB(logger, dir)
	require.NoError(t, err)
	defer db.Close()
	_, _, err = db.SafeHeadAtL1(context.Background(), 100)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestTruncateOnSafeHeadReset(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
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
	logger := testlog.Logger(t, log.LvlInfo)
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

func TestKeysFollowNaturalByteOrdering(t *testing.T) {
	vals := []uint64{0, 1, math.MaxUint32 - 1, math.MaxUint32, math.MaxUint32 + 1, math.MaxUint64 - 1, math.MaxUint64}
	for i := 1; i < len(vals); i++ {
		prev := SafeByL1BlockNumKey.Of(vals[i-1])
		cur := SafeByL1BlockNumKey.Of(vals[i])
		require.True(t, slices.Compare(prev, cur) < 0, "Expected %v key %x to be less than %v key %x", vals[i-1], prev, vals[i], cur)
	}
}
