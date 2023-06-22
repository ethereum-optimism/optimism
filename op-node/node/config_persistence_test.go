package node

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActive(t *testing.T) {
	create := func() *ActiveConfigPersistence {
		dir := t.TempDir()
		config, err := NewConfigPersistence(dir + "/state")
		require.NoError(t, err)
		return config
	}

	t.Run("SequencerStateUnsetWhenFileDoesNotExist", func(t *testing.T) {
		config := create()
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Unset, state)
	})

	t.Run("PersistSequencerStarted", func(t *testing.T) {
		config1 := create()
		require.NoError(t, config1.SequencerStarted())
		state, err := config1.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Started, state)

		config2, err := NewConfigPersistence(config1.file)
		require.NoError(t, err)
		state, err = config2.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Started, state)
	})

	t.Run("PersistSequencerStopped", func(t *testing.T) {
		config1 := create()
		require.NoError(t, config1.SequencerStopped())
		state, err := config1.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Stopped, state)

		config2, err := NewConfigPersistence(config1.file)
		require.NoError(t, err)
		state, err = config2.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Stopped, state)
	})

	t.Run("PersistMultipleChanges", func(t *testing.T) {
		config := create()
		require.NoError(t, config.SequencerStarted())
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Started, state)

		require.NoError(t, config.SequencerStopped())
		state, err = config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Stopped, state)
	})

	t.Run("CreateParentDirs", func(t *testing.T) {
		dir := t.TempDir()
		config, err := NewConfigPersistence(dir + "/some/dir/state")
		require.NoError(t, err)

		// Should be unset before file exists
		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Unset, state)
		require.NoFileExists(t, config.file)

		// Should create directories when updating
		require.NoError(t, config.SequencerStarted())
		require.FileExists(t, config.file)
		state, err = config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, Started, state)
	})
}

func TestDisabledConfigPersistence_AlwaysUnset(t *testing.T) {
	config := DisabledConfigPersistence{}
	state, err := config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, Unset, state)

	require.NoError(t, config.SequencerStarted())
	state, err = config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, Unset, state)

	require.NoError(t, config.SequencerStopped())
	state, err = config.SequencerState()
	require.NoError(t, err)
	require.Equal(t, Unset, state)
}
