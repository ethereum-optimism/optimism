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

func TestSanitizeTarPath(t *testing.T) {
	outDir := t.TempDir()
	tests := []struct {
		name    string
		tarPath string
		want    string
		wantErr string
	}{
		{name: "plain file", tarPath: "file.txt", want: "file.txt"},
		{name: "nested file", tarPath: "dir/file.txt", want: filepath.Join("dir", "file.txt")},
		{name: "dots inside a name", tarPath: "foo..bar", want: "foo..bar"},
		{name: "internal parent that stays inside", tarPath: "a/../b", want: "b"},
		{name: "leading parent", tarPath: "../x", wantErr: "path traversal detected"},
		{name: "deep leading parent", tarPath: "../../../etc/passwd", wantErr: "path traversal detected"},
		{name: "internal parent that escapes", tarPath: "a/../../x", wantErr: "path traversal detected"},
		{name: "absolute path", tarPath: "/etc/passwd", wantErr: "absolute paths are not allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeTarPath(tt.tarPath, outDir)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestUntar_ConfinedToOutDir checks that extraction cannot leave outDir even when the
// entry name passes sanitizeTarPath, here by writing through a symlink inside outDir.
func TestUntar_ConfinedToOutDir(t *testing.T) {
	outDir := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(outDir, "link")))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte("escaped")
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "link/escaped.txt", Mode: 0o644, Size: int64(len(content))}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	err = Untar(outDir, tar.NewReader(&buf))
	require.ErrorContains(t, err, "path escapes from parent")
	require.NoFileExists(t, filepath.Join(outside, "escaped.txt"))
}
