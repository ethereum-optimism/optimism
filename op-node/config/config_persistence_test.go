package config

import (
	"encoding/json"
	"os"
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

	t.Run("PersistSdmPostExecOptIn", func(t *testing.T) {
		config1 := create()
		enabled, set, err := config1.SdmPostExecOptIn()
		require.NoError(t, err)
		require.False(t, enabled)
		require.False(t, set)

		require.NoError(t, config1.SetSdmPostExecOptIn(true))
		enabled, set, err = config1.SdmPostExecOptIn()
		require.NoError(t, err)
		require.True(t, enabled)
		require.True(t, set)

		config2 := NewConfigPersistence(config1.file)
		enabled, set, err = config2.SdmPostExecOptIn()
		require.NoError(t, err)
		require.True(t, enabled)
		require.True(t, set)
	})

	t.Run("MergeSdmAndSequencerState", func(t *testing.T) {
		config := create()
		require.NoError(t, config.SequencerStarted())
		require.NoError(t, config.SetSdmPostExecOptIn(true))

		state, err := config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStarted, state)
		enabled, set, err := config.SdmPostExecOptIn()
		require.NoError(t, err)
		require.True(t, enabled)
		require.True(t, set)

		require.NoError(t, config.SequencerStopped())
		state, err = config.SequencerState()
		require.NoError(t, err)
		require.Equal(t, StateStopped, state)
		enabled, set, err = config.SdmPostExecOptIn()
		require.NoError(t, err)
		require.True(t, enabled)
		require.True(t, set)

		data, err := os.ReadFile(config.file)
		require.NoError(t, err)
		var persisted map[string]bool
		require.NoError(t, json.Unmarshal(data, &persisted))
		require.Equal(t, map[string]bool{"sequencerStarted": false, "sdmPostExecOptIn": true}, persisted)
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

	require.NoError(t, config.SetSdmPostExecOptIn(true))
	enabled, set, err := config.SdmPostExecOptIn()
	require.NoError(t, err)
	require.False(t, enabled)
	require.False(t, set)
}
