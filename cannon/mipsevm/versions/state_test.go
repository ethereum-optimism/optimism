package versions

import (
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
	"github.com/stretchr/testify/require"
)

func TestNewFromState(t *testing.T) {
	t.Run("singlethreaded-2", func(t *testing.T) {
		actual, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)
		require.IsType(t, &singlethreaded.State{}, actual.FPVMState)
		require.Equal(t, VersionSingleThreaded2, actual.Version)
	})

	t.Run("multithreaded", func(t *testing.T) {
		actual, err := NewFromState(multithreaded.CreateEmptyState())
		require.NoError(t, err)
		require.IsType(t, &multithreaded.State{}, actual.FPVMState)
		require.Equal(t, VersionMultiThreaded, actual.Version)
	})
}

func TestLoadStateFromFile(t *testing.T) {
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

func TestLoadStateFromFile64(t *testing.T) {
	t.Skip("TODO(#12205): Test asserting that cannon64 fails to decode a 32-bit state")
}

func TestVersionsOtherThanZeroDoNotSupportJSON(t *testing.T) {
	tests := []struct {
		version     StateVersion
		createState func() mipsevm.FPVMState
	}{
		{VersionSingleThreaded2, func() mipsevm.FPVMState { return singlethreaded.CreateEmptyState() }},
		{VersionMultiThreaded, func() mipsevm.FPVMState { return multithreaded.CreateEmptyState() }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.version.String(), func(t *testing.T) {
			state, err := NewFromState(test.createState())
			require.NoError(t, err)

			dir := t.TempDir()
			path := filepath.Join(dir, "test.json")
			err = serialize.Write(path, state, 0o644)
			require.ErrorIs(t, err, ErrJsonNotSupported)
		})
	}
}

func writeToFile(t *testing.T, filename string, data serialize.Serializable) string {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, serialize.Write(path, data, 0o644))
	return path
}
