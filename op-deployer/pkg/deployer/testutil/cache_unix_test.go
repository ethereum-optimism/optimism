//go:build unix

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedCacheDir(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	shared := SharedCacheDir(t)
	require.Equal(t, filepath.Dir(shared), os.TempDir())
	require.Equal(t, shared, SharedCacheDir(t), "the tests share the directory")

	require.NoError(t, os.Chmod(shared, 0o777))
	require.NotEqual(t, shared, SharedCacheDir(t), "a directory that others can write is not shared")
}
