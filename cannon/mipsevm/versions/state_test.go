package versions

import (
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/stretchr/testify/require"
)

func TestNewFromState(t *testing.T) {
	t.Run("singlethreaded", func(t *testing.T) {
		actual, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)
		require.IsType(t, &singlethreaded.State{}, actual.FPVMState)
		require.Equal(t, VersionSingleThreaded, actual.Version)
	})

	t.Run("multithreaded", func(t *testing.T) {
		actual, err := NewFromState(multithreaded.CreateEmptyState())
		require.NoError(t, err)
		require.IsType(t, &multithreaded.State{}, actual.FPVMState)
		require.Equal(t, VersionMultiThreaded, actual.Version)
	})
}

func TestLoadStateFromFile(t *testing.T) {
	t.Run("SinglethreadedFromJSON", func(t *testing.T) {
		expected, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)

		path := writeToFile(t, "state.json", expected)
		actual, err := LoadStateFromFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("SinglethreadedFromBinary", func(t *testing.T) {
		expected, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)

		path := writeToFile(t, "state.bin.gz", expected)
		actual, err := LoadStateFromFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("MultithreadedFromBinary", func(t *testing.T) {
		expected, err := NewFromState(multithreaded.CreateEmptyState())
		require.NoError(t, err)

		path := writeToFile(t, "state.bin.gz", expected)
		actual, err := LoadStateFromFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestMultithreadedDoesNotSupportJSON(t *testing.T) {
	state, err := NewFromState(multithreaded.CreateEmptyState())
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	err = serialize.Write(path, state, 0o644)
	require.ErrorIs(t, err, ErrJsonNotSupported)
}

func writeToFile(t *testing.T, filename string, data serialize.Serializable) string {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, serialize.Write(path, data, 0o644))
	return path
}
