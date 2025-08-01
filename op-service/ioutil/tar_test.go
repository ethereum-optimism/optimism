package ioutil

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUntar(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Open("testdata/test.tar")
	require.NoError(t, err)
	defer f.Close()

	tr := tar.NewReader(f)
	err = Untar(dir, tr)
	require.NoError(t, err)

	rootFile := filepath.Join(dir, "test.txt")
	content, err := os.ReadFile(rootFile)
	require.NoError(t, err)
	require.Equal(t, "test", string(content))

	nestedFile := filepath.Join(dir, "test", "test.txt")
	content, err = os.ReadFile(nestedFile)
	require.NoError(t, err)
	require.Equal(t, "test", string(content))
}
