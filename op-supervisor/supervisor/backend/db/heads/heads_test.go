package heads

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestHeads_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heads.json")
	chainA := types.ChainIDFromUInt64(3)
	chainAHeads := ChainHeads{
		Unsafe:         1,
		CrossUnsafe:    2,
		LocalSafe:      3,
		CrossSafe:      4,
		LocalFinalized: 5,
		CrossFinalized: 6,
	}
	chainB := types.ChainIDFromUInt64(5)
	chainBHeads := ChainHeads{
		Unsafe:         11,
		CrossUnsafe:    12,
		LocalSafe:      13,
		CrossSafe:      14,
		LocalFinalized: 15,
		CrossFinalized: 16,
	}

	orig, err := NewHeadTracker(path)
	require.NoError(t, err)
	err = orig.Apply(OperationFn(func(heads *Heads) error {
		heads.Put(chainA, chainAHeads)
		heads.Put(chainB, chainBHeads)
		return nil
	}))
	require.NoError(t, err)
	require.Equal(t, orig.Current().Get(chainA), chainAHeads)
	require.Equal(t, orig.Current().Get(chainB), chainBHeads)

	loaded, err := NewHeadTracker(path)
	require.NoError(t, err)
	require.EqualValues(t, loaded.Current(), orig.Current())
}

func TestHeads_NoChangesMadeIfOperationFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heads.json")
	chainA := types.ChainIDFromUInt64(3)
	chainAHeads := ChainHeads{
		Unsafe:         1,
		CrossUnsafe:    2,
		LocalSafe:      3,
		CrossSafe:      4,
		LocalFinalized: 5,
		CrossFinalized: 6,
	}

	orig, err := NewHeadTracker(path)
	require.NoError(t, err)
	boom := errors.New("boom")
	err = orig.Apply(OperationFn(func(heads *Heads) error {
		heads.Put(chainA, chainAHeads)
		return boom
	}))
	require.ErrorIs(t, err, boom)
	require.Equal(t, ChainHeads{}, orig.Current().Get(chainA))

	// Should be able to load from disk too
	loaded, err := NewHeadTracker(path)
	require.NoError(t, err)
	require.EqualValues(t, loaded.Current(), orig.Current())
}

func TestHeads_NoChangesMadeIfWriteFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid/heads.json")
	chainA := types.ChainIDFromUInt64(3)
	chainAHeads := ChainHeads{
		Unsafe:         1,
		CrossUnsafe:    2,
		LocalSafe:      3,
		CrossSafe:      4,
		LocalFinalized: 5,
		CrossFinalized: 6,
	}

	orig, err := NewHeadTracker(path)
	require.NoError(t, err)
	err = orig.Apply(OperationFn(func(heads *Heads) error {
		heads.Put(chainA, chainAHeads)
		return nil
	}))
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Equal(t, ChainHeads{}, orig.Current().Get(chainA))
}
