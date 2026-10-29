//go:build unix

package forge

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
