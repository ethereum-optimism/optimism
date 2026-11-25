// COMPATIBILITY SHIM TEST — safe to delete alongside compat_virtual_parent.go
// once no pre-#20726 databases remain in operation.

package raftwallogdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-core/interop"
)

// seedPreVirtualParentDB writes a pre-#20726 head layout into dir: a virtual
// parent at activation-1 (sealed with a zero parent hash, no logs) followed
// by the activation block with its real parent hash.
func seedPreVirtualParentDB(t *testing.T, dir string, activation eth.BlockID, parentHash common.Hash, ts uint64) {
	t.Helper()
	db, err := Open(dir, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)

	virtualParent := eth.BlockID{Hash: parentHash, Number: activation.Number - 1}
	require.NoError(t, db.SealBlock(common.Hash{}, virtualParent, ts))
	require.NoError(t, db.SealBlock(parentHash, activation, ts))

	require.NoError(t, db.Close())
}

func TestCompat_HidesPreVirtualParentEntry(t *testing.T) {
	dir := t.TempDir()
	activation := blockID(100, 0x64)
	parentHash := hash(0x63)
	seedPreVirtualParentDB(t, dir, activation, parentHash, 1000)

	db, err := Open(dir, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	seal, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, activation.Number, seal.Number, "FirstSealedBlock must skip the virtual parent")
	require.Equal(t, activation.Hash, seal.Hash)

	// OpenBlock on the hidden entry reports it as below the (post-shim) first.
	_, _, _, err = db.OpenBlock(activation.Number - 1)
	require.ErrorIs(t, err, interop.ErrSkipped)

	// Activation block is queryable as the new first block.
	ref, _, _, err := db.OpenBlock(activation.Number)
	require.NoError(t, err)
	require.Equal(t, activation.Hash, ref.Hash)
	require.Equal(t, parentHash, ref.ParentHash)
}

func TestCompat_PostChangeLayoutUnaffected(t *testing.T) {
	dir := t.TempDir()
	{
		db, err := Open(dir, eth.ChainIDFromUInt64(10))
		require.NoError(t, err)
		// Post-#20726: activation block is the first entry, with a real parent hash.
		require.NoError(t, db.SealBlock(hash(0x63), blockID(100, 0x64), 1000))
		require.NoError(t, db.Close())
	}

	db, err := Open(dir, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	seal, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(100), seal.Number)
}

// Only-virtual-parent DB (no activation block yet) is not touched — the shim
// requires a second entry to distinguish a genuine empty-anchor DB from one
// that just happens to have a zero parent hash.
func TestCompat_OnlyVirtualParent_NotHidden(t *testing.T) {
	dir := t.TempDir()
	{
		db, err := Open(dir, eth.ChainIDFromUInt64(10))
		require.NoError(t, err)
		require.NoError(t, db.SealBlock(common.Hash{}, blockID(99, 0x63), 1000))
		require.NoError(t, db.Close())
	}

	db, err := Open(dir, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	seal, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(99), seal.Number)
}

// Genesis-anchored DBs legitimately have a zero parent hash at block 0; the
// shim's d.firstBlock == 0 guard keeps them intact.
func TestCompat_GenesisAnchor_NotHidden(t *testing.T) {
	dir := t.TempDir()
	{
		db, err := Open(dir, eth.ChainIDFromUInt64(10))
		require.NoError(t, err)
		require.NoError(t, db.SealBlock(common.Hash{}, blockID(0, 0xA0), 0))
		require.NoError(t, db.SealBlock(hash(0xA0), blockID(1, 0xA1), 1))
		require.NoError(t, db.Close())
	}

	db, err := Open(dir, eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	seal, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(0), seal.Number)
}
