package filter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func newStatefulTestBackend(t *testing.T, statePath string, manual bool) *Backend {
	return NewBackend(context.Background(), BackendParams{
		Logger:            testlog.Logger(t, log.LevelCrit),
		Metrics:           metrics.NoopMetrics,
		Chains:            make(map[eth.ChainID]ChainIngester),
		CrossValidator:    &mockCrossValidator{},
		ManualFailsafe:    manual,
		FailsafeStatePath: statePath,
	})
}

// restartBackend simulates a process restart: the new backend sees only the stored state.
func restartBackend(t *testing.T, statePath string, flagEnabled bool) *Backend {
	manual, err := initialManualFailsafe(testlog.Logger(t, log.LevelCrit), statePath, flagEnabled)
	require.NoError(t, err)
	return newStatefulTestBackend(t, statePath, manual)
}

func TestManualFailsafeSurvivesRestart(t *testing.T) {
	statePath := FailsafeStatePath(t.TempDir())

	first := newStatefulTestBackend(t, statePath, false)
	require.NoError(t, first.SetFailsafeEnabled(true))

	second := restartBackend(t, statePath, false)
	require.True(t, second.FailsafeEnabled(), "manual failsafe must survive a restart")

	require.NoError(t, second.SetFailsafeEnabled(false))
	third := restartBackend(t, statePath, false)
	require.False(t, third.FailsafeEnabled(), "a cleared failsafe must stay cleared after a restart")
}

func TestManualFailsafeStartFlag(t *testing.T) {
	statePath := FailsafeStatePath(t.TempDir())

	b := restartBackend(t, statePath, true)
	require.True(t, b.FailsafeEnabled())

	// An operator can clear the failsafe, but the flag turns it on again at the next start.
	require.NoError(t, b.SetFailsafeEnabled(false))
	require.False(t, b.FailsafeEnabled())
	require.True(t, restartBackend(t, statePath, true).FailsafeEnabled())
}

func TestReadManualFailsafeMissingFile(t *testing.T) {
	enabled, err := readManualFailsafe(FailsafeStatePath(t.TempDir()))
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestReadManualFailsafeCorruptFile(t *testing.T) {
	statePath := FailsafeStatePath(t.TempDir())
	require.NoError(t, os.WriteFile(statePath, []byte("not json"), 0o644))
	_, err := readManualFailsafe(statePath)
	require.Error(t, err)
}

func TestInitialManualFailsafeCorruptFile(t *testing.T) {
	corruptStatePath := func(t *testing.T) string {
		statePath := FailsafeStatePath(t.TempDir())
		require.NoError(t, os.WriteFile(statePath, []byte("not json"), 0o644))
		return statePath
	}

	t.Run("FlagUnset", func(t *testing.T) {
		_, err := initialManualFailsafe(testlog.Logger(t, log.LevelCrit), corruptStatePath(t), false)
		require.Error(t, err)
	})

	t.Run("FlagSet", func(t *testing.T) {
		statePath := corruptStatePath(t)
		b := restartBackend(t, statePath, true)
		require.True(t, b.FailsafeEnabled())

		// A change through the admin RPC replaces the corrupt file.
		require.NoError(t, b.SetFailsafeEnabled(false))
		require.False(t, restartBackend(t, statePath, false).FailsafeEnabled())
	})
}

func TestSetFailsafeEnabledWriteError(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "missing-dir", failsafeStateFile)
	b := newStatefulTestBackend(t, statePath, false)

	// The in-memory change applies even when the write fails.
	require.Error(t, b.SetFailsafeEnabled(true))
	require.True(t, b.FailsafeEnabled())
}

func TestSetFailsafeEnabledWithoutStatePath(t *testing.T) {
	b := newStatefulTestBackend(t, "", false)
	require.NoError(t, b.SetFailsafeEnabled(true))
	require.True(t, b.FailsafeEnabled())
}
