package rustengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdjacentEngineBinary verifies the bundled-release resolution rule: an op-script-engine binary
// sitting next to the running executable is found with no configuration. This is what makes the
// op-deployer Docker image and release archives resolve the shipped engine.
func TestAdjacentEngineBinary(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)
	adjacent := filepath.Join(filepath.Dir(exe), EngineSpec.Binary)

	// Don't clobber a real bundled engine if one is genuinely present next to the test binary.
	if _, err := os.Stat(adjacent); err == nil {
		got, ok := adjacentEngineBinary()
		require.True(t, ok)
		require.Equal(t, adjacent, got)
		return
	}

	// Drop a fake engine next to the test executable and confirm it resolves.
	if err := os.WriteFile(adjacent, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Skipf("test-binary directory not writable, cannot exercise adjacency: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(adjacent) })

	got, ok := adjacentEngineBinary()
	require.True(t, ok, "adjacency resolution must find the bundled engine next to the executable")
	require.Equal(t, adjacent, got)
}

// TestAdjacentEngineBinaryAbsent confirms the resolver reports "not found" (so EngineBinary falls
// through to discovery) when no engine sits next to the executable.
func TestAdjacentEngineBinaryAbsent(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)
	adjacent := filepath.Join(filepath.Dir(exe), EngineSpec.Binary)
	if _, err := os.Stat(adjacent); err == nil {
		t.Skip("an engine is present next to the test binary; cannot exercise the absent case")
	}
	_, ok := adjacentEngineBinary()
	require.False(t, ok)
}
