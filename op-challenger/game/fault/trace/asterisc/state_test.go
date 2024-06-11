package asterisc

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

		var expected VMState
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

		var expected VMState
		require.NoError(t, json.Unmarshal(testState, &expected))
		require.Equal(t, &expected, state)
	})

	t.Run("InvalidStateWitness", func(t *testing.T) {
		invalidWitnessLen := asteriscWitnessLen - 1
		state := &VMState{
			Step:    10,
			Exited:  true,
			Witness: make([]byte, invalidWitnessLen),
		}
		err := state.validateState()
		require.ErrorContains(t, err, "invalid witness")
	})

	t.Run("InvalidStateHash", func(t *testing.T) {
		state := &VMState{
			Step:    10,
			Exited:  true,
			Witness: make([]byte, asteriscWitnessLen),
		}
		// Unknown exit code
		state.StateHash[0] = 37
		err := state.validateState()
		require.ErrorContains(t, err, "invalid stateHash: unknown exitCode")
		// Exited but ExitCode is VMStatusUnfinished
		state.StateHash[0] = 3
		err = state.validateState()
		require.ErrorContains(t, err, "invalid stateHash: invalid exitCode")
		// Not Exited but ExitCode is not VMStatusUnfinished
		state.Exited = false
		for exitCode := 0; exitCode < 3; exitCode++ {
			state.StateHash[0] = byte(exitCode)
			err = state.validateState()
			require.ErrorContains(t, err, "invalid stateHash: invalid exitCode")
		}
	})
}
