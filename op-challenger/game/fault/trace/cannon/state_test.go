package cannon

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/stretchr/testify/require"
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

		var expected mipsevm.State
		require.NoError(t, json.Unmarshal(testState, &expected))
		require.Equal(t, &expected, state)
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

		var expected mipsevm.State
		require.NoError(t, json.Unmarshal(testState, &expected))
		require.Equal(t, &expected, state)
	})
}
