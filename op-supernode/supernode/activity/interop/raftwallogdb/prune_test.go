package raftwallogdb

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// Prune drops oldest entries below a timestamp horizon while preserving the tip,
// a contiguous prefix, and forward-sealing — see Prune's safety contract.

func TestPrune_DropsPrefixBelowHorizon(t *testing.T) {
	db := tempDB(t)
	// Blocks 1..10, timestamps 100..1000.
	sealRange(t, db, blockID(0, 0), 1, 10)

	// Horizon 550: blocks with ts<550 are 1..5 (ts 100..500); block 6 (ts 600) stays.
	pruned, err := db.Prune(550)
	require.NoError(t, err)
	require.Equal(t, uint64(5), pruned)

	// Pruned entries are gone; the boundary and tip survive.
	_, err = db.FindSealedBlock(5)
	require.ErrorIs(t, err, interop.ErrSkipped)
	got, err := db.FindSealedBlock(6)
	require.NoError(t, err)
	require.Equal(t, uint64(6), got.Number)
	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(6), first.Number)
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(10), latest.Number)
}

func TestPrune_NeverRemovesTip(t *testing.T) {
	db := tempDB(t)
	sealRange(t, db, blockID(0, 0), 1, 10)

	// Horizon far above every timestamp: all but the tip qualify, but the tip
	// (block 10) must be retained so new seals can chain onto it.
	pruned, err := db.Prune(1_000_000)
	require.NoError(t, err)
	require.Equal(t, uint64(9), pruned) // 1..9 pruned, 10 kept

	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(10), first.Number)
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(10), latest.Number)

	// Single remaining block: further pruning is a no-op (cannot remove the tip).
	pruned, err = db.Prune(1_000_000)
	require.NoError(t, err)
	require.Zero(t, pruned)
}

func TestPrune_NoOpCases(t *testing.T) {
	t.Run("empty db", func(t *testing.T) {
		db := tempDB(t)
		pruned, err := db.Prune(1000)
		require.NoError(t, err)
		require.Zero(t, pruned)
	})
	t.Run("nothing below horizon", func(t *testing.T) {
		db := tempDB(t)
		sealRange(t, db, blockID(0, 0), 1, 5) // ts 100..500
		pruned, err := db.Prune(100)          // block 1 ts==100 is not < 100
		require.NoError(t, err)
		require.Zero(t, pruned)
		first, err := db.FirstSealedBlock()
		require.NoError(t, err)
		require.Equal(t, uint64(1), first.Number)
	})
}

func TestPrune_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	chainID := eth.ChainIDFromUInt64(10)

	db, err := Open(dir, chainID)
	require.NoError(t, err)
	sealRange(t, db, blockID(0, 0), 1, 10)
	pruned, err := db.Prune(550)
	require.NoError(t, err)
	require.Equal(t, uint64(5), pruned)
	require.NoError(t, db.Close())

	// refreshCache must derive the advanced firstBlock from the WAL's FirstIndex.
	db2, err := Open(dir, chainID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db2.Close() })
	first, err := db2.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(6), first.Number)
	_, err = db2.FindSealedBlock(5)
	require.ErrorIs(t, err, interop.ErrSkipped)
}

func TestPrune_ForwardSealingStillWorks(t *testing.T) {
	db := tempDB(t)
	last := sealRange(t, db, blockID(0, 0), 1, 10)
	_, err := db.Prune(550)
	require.NoError(t, err)

	// Sealing block 11 onto the (unpruned) tip must still succeed.
	blk11 := blockID(11, 11)
	require.NoError(t, db.SealBlock(last.Hash, blk11, 1100))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(11), latest.Number)
}
