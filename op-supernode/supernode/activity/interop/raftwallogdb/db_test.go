package raftwallogdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func hash(b byte) common.Hash {
	var h common.Hash
	for i := range h {
		h[i] = b
	}
	return h
}

func blockID(num uint64, b byte) eth.BlockID {
	return eth.BlockID{Hash: hash(b), Number: num}
}

func tempDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(t.TempDir(), eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestEmpty(t *testing.T) {
	db := tempDB(t)
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
	_, err := db.FirstSealedBlock()
	require.ErrorIs(t, err, types.ErrFuture)
}

func TestSealAndOpenBlock(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0x00)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 100))

	em := &types.ExecutingMessage{
		ChainID:   eth.ChainIDFromUInt64(20),
		BlockNum:  1,
		LogIdx:    1,
		Timestamp: 200,
		Checksum:  types.MessageChecksum(hash(0xEE)),
	}
	require.NoError(t, db.AddLog(hash(0x01), parent, 0, nil))
	require.NoError(t, db.AddLog(hash(0x02), parent, 1, em))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 200))

	ref, count, msgs, err := db.OpenBlock(1)
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, ref.Hash)
	require.Equal(t, parent.Hash, ref.ParentHash)
	require.Equal(t, uint32(2), count)
	require.Len(t, msgs, 1)
	require.Equal(t, em, msgs[1])
}

func TestContains(t *testing.T) {
	db := tempDB(t)
	chain := eth.ChainIDFromUInt64(10)
	parent := blockID(0, 0x00)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 100))
	logHash := hash(0xAB)
	require.NoError(t, db.AddLog(logHash, parent, 0, nil))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 200))

	good := types.ChecksumArgs{BlockNumber: 1, LogIndex: 0, Timestamp: 200, ChainID: chain, LogHash: logHash}.Checksum()
	seal, err := db.Contains(types.ContainsQuery{BlockNum: 1, LogIdx: 0, Timestamp: 200, Checksum: good})
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, seal.Hash)

	_, err = db.Contains(types.ContainsQuery{BlockNum: 1, LogIdx: 5, Timestamp: 200, Checksum: good})
	require.ErrorIs(t, err, types.ErrConflict)

	_, err = db.Contains(types.ContainsQuery{BlockNum: 10, LogIdx: 0, Timestamp: 50, Checksum: good})
	require.ErrorIs(t, err, types.ErrConflict)
	_, err = db.Contains(types.ContainsQuery{BlockNum: 10, LogIdx: 0, Timestamp: 999, Checksum: good})
	require.ErrorIs(t, err, types.ErrFuture)
}

func TestRewind(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0x00)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	prev := parent
	for n := uint64(1); n <= 5; n++ {
		blk := blockID(n, byte(n))
		require.NoError(t, db.SealBlock(prev.Hash, blk, n*100))
		prev = blk
	}

	target := blockID(3, 0x03)
	require.NoError(t, db.Rewind(target))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, target, latest)
	_, err := db.FindSealedBlock(5)
	require.ErrorIs(t, err, types.ErrFuture)
	_, err = db.FindSealedBlock(3)
	require.NoError(t, err)
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	parent := blockID(0, 0x00)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	require.NoError(t, db.AddLog(hash(0xAA), parent, 0, nil))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 100))
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, blk1, latest)
}

func TestPreSealCrashLosesPending(t *testing.T) {
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	parent := blockID(5, 0x05)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 500))
	require.NoError(t, db.AddLog(hash(0xAA), parent, 0, nil))
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, parent, latest)
}
