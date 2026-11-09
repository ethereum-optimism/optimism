//go:build unix

package bundle

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFile_RefusesLinkUnlessFollowed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env": "KEY=secret"})
	require.NoError(t, os.Symlink(".env", filepath.Join(dir, "foo.txt")))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o600))
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer root.Close()

	_, _, err = readFile(root, "foo.txt", false)
	require.ErrorContains(t, err, "not a regular file")
	content, _, err := readFile(root, "foo.txt", true)
	require.NoError(t, err)
	require.Equal(t, "KEY=secret", string(content))
	_, _, err = readFile(root, "pipe", true)
	require.ErrorContains(t, err, "not a regular file")
}
