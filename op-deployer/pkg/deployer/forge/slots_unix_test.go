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

// TestSlotDir checks that the slots are in the user's cache directory, which another user cannot
// create first, and in the shared temporary directory only for a user without a private one.
func TestSlotDir(t *testing.T) {
	setCache := func(dir string) {
		// The cache directory is $XDG_CACHE_HOME on Linux and under $HOME on macOS.
		t.Setenv("XDG_CACHE_HOME", dir)
		t.Setenv("HOME", dir)
	}
	tmpSlots := filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid()))

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
	require.Equal(t, tmpSlots, slotDir())

	// Like a cache directory that other users share.
	shared := t.TempDir()
	setCache(shared)
	require.NotEqual(t, tmpSlots, slotDir())
	parent := filepath.Dir(slotDir())
	require.NoError(t, os.Chmod(parent, 0o777))
	require.Equal(t, tmpSlots, slotDir())
}

// TestSlotPool_RefusesSharedDir checks that the slots are not taken from a directory that another
// user could have created first in the shared temporary directory, to hold its locks or plant links.
func TestSlotPool_RefusesSharedDir(t *testing.T) {
	writable := t.TempDir()
	require.NoError(t, os.Chmod(writable, 0o777))
	_, err := (&slotPool{dir: writable, size: 1}).acquire(context.Background())
	require.ErrorContains(t, err, "not a private directory")

	for _, target := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		link := filepath.Join(t.TempDir(), "slots")
		require.NoError(t, os.Symlink(target, link))
		_, err = (&slotPool{dir: link, size: 1}).acquire(context.Background())
		require.ErrorContains(t, err, "not a private directory")
		require.NoFileExists(t, filepath.Join(target, "0"), "the link must not be followed")
	}

	release, err := (&slotPool{dir: filepath.Join(t.TempDir(), "slots"), size: 1}).acquire(context.Background())
	require.NoError(t, err)
	release()
}
