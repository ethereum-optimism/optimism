package safedb

import (
	"context"
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
		require.Equal(t, l1a.Hash, actualL1)
		require.Equal(t, l2a.Hash, actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1a.Number+1)
		require.NoError(t, err)
		require.Equal(t, l1a.Hash, actualL1)
		require.Equal(t, l2a.Hash, actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number)
		require.NoError(t, err)
		require.Equal(t, l1b.Hash, actualL1)
		require.Equal(t, l2b.Hash, actualL2)

		actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number+1)
		require.NoError(t, err)
		require.Equal(t, l1b.Hash, actualL1)
		require.Equal(t, l2b.Hash, actualL2)
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

func TestTruncateDataWhenSafeHeadGoesBackwards(t *testing.T) {
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
	l2c := eth.L2BlockRef{
		Hash:   common.Hash{0x02, 0xcc},
		Number: 21,
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
		Number: 148,
	}
	require.NoError(t, db.SafeHeadUpdated(l2a, l1a))
	require.NoError(t, db.SafeHeadUpdated(l2b, l1b))

	actualL1, actualL2, err := db.SafeHeadAtL1(context.Background(), l1b.Number)
	require.NoError(t, err)
	require.Equal(t, l1b.Hash, actualL1)
	require.Equal(t, l2b.Hash, actualL2)

	require.NoError(t, db.SafeHeadUpdated(l2c, l1c))

	actualL1, actualL2, err = db.SafeHeadAtL1(context.Background(), l1b.Number)
	require.NoError(t, err)
	require.Equal(t, l1c.Hash, actualL1)
	require.Equal(t, l2c.Hash, actualL2)
}
