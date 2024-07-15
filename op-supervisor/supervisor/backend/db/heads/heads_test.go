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
		Unsafe:         ChainHead{Index: 10, ID: 100},
		CrossUnsafe:    ChainHead{Index: 9, ID: 99},
		LocalSafe:      ChainHead{Index: 8, ID: 98},
		CrossSafe:      ChainHead{Index: 7, ID: 97},
		LocalFinalized: ChainHead{Index: 6, ID: 96},
		CrossFinalized: ChainHead{Index: 5, ID: 95},
	}
	chainB := types.ChainIDFromUInt64(5)
	chainBHeads := ChainHeads{
		Unsafe:         ChainHead{Index: 90, ID: 9},
		CrossUnsafe:    ChainHead{Index: 80, ID: 8},
		LocalSafe:      ChainHead{Index: 70, ID: 7},
		CrossSafe:      ChainHead{Index: 60, ID: 6},
		LocalFinalized: ChainHead{Index: 50, ID: 5},
		CrossFinalized: ChainHead{Index: 40, ID: 4},
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
		Unsafe:         ChainHead{Index: 10, ID: 100},
		CrossUnsafe:    ChainHead{Index: 9, ID: 99},
		LocalSafe:      ChainHead{Index: 8, ID: 98},
		CrossSafe:      ChainHead{Index: 7, ID: 97},
		LocalFinalized: ChainHead{Index: 6, ID: 96},
		CrossFinalized: ChainHead{Index: 5, ID: 95},
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
	// Write will fail because directory doesn't exist.
	path := filepath.Join(dir, "invalid/heads.json")
	chainA := types.ChainIDFromUInt64(3)
	chainAHeads := ChainHeads{
		Unsafe:         ChainHead{Index: 10, ID: 100},
		CrossUnsafe:    ChainHead{Index: 9, ID: 99},
		LocalSafe:      ChainHead{Index: 8, ID: 98},
		CrossSafe:      ChainHead{Index: 7, ID: 97},
		LocalFinalized: ChainHead{Index: 6, ID: 96},
		CrossFinalized: ChainHead{Index: 5, ID: 95},
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
