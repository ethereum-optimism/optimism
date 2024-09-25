package versions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/stretchr/testify/require"
)

func TestDetectVersion(t *testing.T) {
	t.Run("SingleThreadedJSON", func(t *testing.T) {
		state, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)
		path := writeToFile(t, "state.json", state)
		version, err := DetectVersion(path)
		require.NoError(t, err)
		require.Equal(t, VersionSingleThreaded, version)
	})

	t.Run("SingleThreadedBinary", func(t *testing.T) {
		state, err := NewFromState(singlethreaded.CreateEmptyState())
		require.NoError(t, err)
		path := writeToFile(t, "state.bin.gz", state)
		version, err := DetectVersion(path)
		require.NoError(t, err)
		require.Equal(t, VersionSingleThreaded, version)
	})

	t.Run("MultiThreadedBinary", func(t *testing.T) {
		state, err := NewFromState(multithreaded.CreateEmptyState())
		require.NoError(t, err)
		path := writeToFile(t, "state.bin.gz", state)
		version, err := DetectVersion(path)
		require.NoError(t, err)
		require.Equal(t, VersionMultiThreaded, version)
	})
}

func TestDetectVersionInvalid(t *testing.T) {
	t.Run("bad gzip", func(t *testing.T) {
		dir := t.TempDir()
		filename := "state.bin.gz"
		path := filepath.Join(dir, filename)
		require.NoError(t, os.WriteFile(path, []byte("ekans"), 0o644))

		_, err := DetectVersion(path)
		require.ErrorContains(t, err, "failed to open file")
	})

	t.Run("unknown version", func(t *testing.T) {
		dir := t.TempDir()
		filename := "state.bin.gz"
		path := filepath.Join(dir, filename)
		const badVersion = 0xFF
		err := ioutil.WriteCompressedBytes(path, []byte{badVersion}, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		require.NoError(t, err)

		_, err = DetectVersion(path)
		require.ErrorIs(t, err, ErrUnknownVersion)
	})
}
