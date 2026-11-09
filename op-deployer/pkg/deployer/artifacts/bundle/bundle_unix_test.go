//go:build unix

package bundle

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFile_RefusesLinksAndPipes(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{".env": "KEY=secret"})
	require.NoError(t, os.Symlink(".env", filepath.Join(dir, "foo.txt")))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o600))
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer root.Close()

	for _, name := range []string{"foo.txt", "pipe"} {
		_, _, err = readFile(root, name)
		require.ErrorContains(t, err, "not a regular file", name)
	}
	content, _, err := readFile(root, ".env")
	require.NoError(t, err)
	require.Equal(t, "KEY=secret", string(content))
}
