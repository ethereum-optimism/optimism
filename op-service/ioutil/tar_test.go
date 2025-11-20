package ioutil

import (
	"archive/tar"
	"bytes"
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

func TestUntar_PathTraversalProtection(t *testing.T) {
	dir := t.TempDir()

	// Create a malicious tar file with path traversal attempts
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a file that tries to traverse outside the extraction directory
	hdr := &tar.Header{
		Name: "../outside.txt",
		Mode: 0644,
		Size: int64(len("malicious content")),
	}
	err := tw.WriteHeader(hdr)
	require.NoError(t, err)
	_, err = tw.Write([]byte("malicious content"))
	require.NoError(t, err)

	// Add another file with absolute path
	hdr = &tar.Header{
		Name: "/absolute/path.txt",
		Mode: 0644,
		Size: int64(len("absolute content")),
	}
	err = tw.WriteHeader(hdr)
	require.NoError(t, err)
	_, err = tw.Write([]byte("absolute content"))
	require.NoError(t, err)

	// Add another file with double dot at start
	hdr = &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: int64(len("passwd content")),
	}
	err = tw.WriteHeader(hdr)
	require.NoError(t, err)
	_, err = tw.Write([]byte("passwd content"))
	require.NoError(t, err)

	err = tw.Close()
	require.NoError(t, err)

	// Try to extract the malicious tar file
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	err = Untar(dir, tr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "path traversal detected")

	// Verify that no malicious files were created outside the directory
	outsideFile := filepath.Join(filepath.Dir(dir), "outside.txt")
	require.NoFileExists(t, outsideFile)

	// Verify that no files were created inside the directory either
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	require.NoError(t, err)
	require.Empty(t, files, "No files should have been extracted due to path traversal protection")
}
