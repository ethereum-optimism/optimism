//go:build unix

package forge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSlotDir checks that the slots are in the user's cache directory, and in the temporary
// directory for a user who cannot write one.
func TestSlotDir(t *testing.T) {
	setCache := func(dir string) {
		// The cache directory is $XDG_CACHE_HOME on Linux and under $HOME on macOS.
		t.Setenv("XDG_CACHE_HOME", dir)
		t.Setenv("HOME", dir)
	}

	cache := t.TempDir()
	setCache(cache)
	dir := slotDir()
	require.True(t, strings.HasPrefix(dir, cache+string(filepath.Separator)), "slot dir %s is not in the cache dir %s", dir, cache)
	release, err := (&slotPool{dir: dir, size: 1}).acquire(context.Background())
	require.NoError(t, err)
	release()

	// Like a uid whose home is /, which it cannot write.
	noCache := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(noCache, nil, 0o600))
	setCache(noCache)
	require.Equal(t, filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid())), slotDir())
}

// TestSlotPool_UsesAnyDir checks that the slots are taken from the directory as it is, whatever its
// mode, or through a link, and that missing parents are created: op-deployer trusts its machine.
func TestSlotPool_UsesAnyDir(t *testing.T) {
	groupWritable := t.TempDir()
	require.NoError(t, os.Chmod(groupWritable, 0o775))
	link := filepath.Join(t.TempDir(), "slots")
	require.NoError(t, os.Symlink(t.TempDir(), link))
	for _, dir := range []string{groupWritable, link, filepath.Join(t.TempDir(), "missing", "slots")} {
		release, err := (&slotPool{dir: dir, size: 1}).acquire(context.Background())
		require.NoError(t, err, dir)
		release()
	}
}
