package node

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActive(t *testing.T) {
	create := func() *ActiveConfigPersistence {
		dir := t.TempDir()
		config := NewConfigPersistence(dir + "/state")
		return config
	}

	t.Run("SequencerStateUnsetWhenFileDoesNotExist", func(t *testing.T) {
		config := create()
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateUnset, state)
	})

	t.Run("PersistSequencerStarted", func(t *testing.T) {
		config1 := create()
		require.NoError(t, config1.SequencerStarted())
		state, err := config1.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStarted, state)

		config2 := NewConfigPersistence(config1.file)
		state, err = config2.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStarted, state)
	})

	t.Run("PersistSequencerStopped", func(t *testing.T) {
		config1 := create()
		require.NoError(t, config1.SequencerStopped())
		state, err := config1.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStopped, state)

		config2 := NewConfigPersistence(config1.file)
		state, err = config2.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStopped, state)
	})

	t.Run("PersistMultipleChanges", func(t *testing.T) {
		config := create()
		require.NoError(t, config.SequencerStarted())
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStarted, state)

		require.NoError(t, config.SequencerStopped())
		state, err = config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStopped, state)
	})

	t.Run("CreateParentDirs", func(t *testing.T) {
		dir := t.TempDir()
		config := NewConfigPersistence(dir + "/some/dir/state")

		// Should be unset before file exists
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateUnset, state)
		require.NoFileExists(t, config.file)

		// Should create directories when updating
		require.NoError(t, config.SequencerStarted())
		require.FileExists(t, config.file)
		state, err = config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStarted, state)
	})
}

func TestDisabledConfigPersistence_AlwaysUnset(t *testing.T) {
	config := DisabledConfigPersistence{}
	state, err := config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, StateUnset, state)

	require.NoError(t, config.SequencerStarted())
	state, err = config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, StateUnset, state)

	require.NoError(t, config.SequencerStopped())
	state, err = config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, StateUnset, state)
}
