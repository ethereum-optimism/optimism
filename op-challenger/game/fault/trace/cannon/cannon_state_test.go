package cannon

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/stretchr/testify/require"
)

//go:embed test_data/state.bin
var testState []byte

func TestLoadState(t *testing.T) {
	t.Run("Uncompressed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.bin")
		require.NoError(t, os.WriteFile(path, testState, 0644))

		state, err := parseState(path)
		require.NoError(t, err)

		expected := &mipsevm.State{}
		err = expected.Deserialize(bytes.NewReader(testState))
		require.NoError(t, err)
		require.Equal(t, expected, state)
	})

	t.Run("Gzipped", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.bin.gz")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		require.NoError(t, err)
		defer f.Close()
		writer := gzip.NewWriter(f)
		_, err = writer.Write(testState)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		state, err := parseState(path)
		require.NoError(t, err)

		expected := &mipsevm.State{}
		err = expected.Deserialize(bytes.NewReader(testState))
		require.NoError(t, err)
		require.Equal(t, expected, state)
	})
}
