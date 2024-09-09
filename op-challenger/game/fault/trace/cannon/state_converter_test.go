package cannon

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
)

//go:embed test_data/state.json
var testState []byte

func TestLoadState(t *testing.T) {
	t.Run("Uncompressed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.json")
		require.NoError(t, os.WriteFile(path, testState, 0644))

		state, err := parseState(path)
		require.NoError(t, err)

		expected := loadExpectedState(t)
		require.Equal(t, expected, state)
	})

	t.Run("Gzipped", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.json.gz")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		require.NoError(t, err)
		defer f.Close()
		writer := gzip.NewWriter(f)
		_, err = writer.Write(testState)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		state, err := parseState(path)
		require.NoError(t, err)

		expected := loadExpectedState(t)
		require.Equal(t, expected, state)
	})

	t.Run("Binary", func(t *testing.T) {
		expected := loadExpectedState(t)

		path := writeState(t, "state.bin", expected)

		state, err := parseState(path)
		require.NoError(t, err)
		require.Equal(t, expected, state)
	})

	t.Run("BinaryGzip", func(t *testing.T) {
		expected := loadExpectedState(t)

		path := writeState(t, "state.bin.gz", expected)

		state, err := parseState(path)
		require.NoError(t, err)
		require.Equal(t, expected, state)
	})
}

func writeState(t *testing.T, filename string, state versions.VersionedState) string {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, serialize.Write(path, state, 0644))
	return path
}

func loadExpectedState(t *testing.T) versions.VersionedState {
	var expected singlethreaded.State
	require.NoError(t, json.Unmarshal(testState, &expected))
	state, err := versions.NewFromState(&expected)
	require.NoError(t, err)
	return state
}
