package ioutil

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicWriter_RenameOnClose(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	f, err := NewAtomicWriterCompressed(target, 0755)
	require.NoError(t, err)
	defer f.Close()
	_, err = os.Stat(target)
	require.ErrorIs(t, err, os.ErrNotExist, "should not create target file when created")

	content := ([]byte)("Hello world")
	n, err := f.Write(content)
	require.NoError(t, err)
	require.Equal(t, len(content), n)
	_, err = os.Stat(target)
	require.ErrorIs(t, err, os.ErrNotExist, "should not create target file when writing")

	require.NoError(t, f.Close())
	stat, err := os.Stat(target)
	require.NoError(t, err, "should create target file when closed")
	require.EqualValues(t, fs.FileMode(0755), stat.Mode())

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1, "should not leave temporary files behind")
}

func TestAtomicWriter_MultipleClose(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	f, err := NewAtomicWriterCompressed(target, 0755)
	require.NoError(t, err)

	require.NoError(t, f.Close())
	require.ErrorIs(t, f.Close(), os.ErrClosed)
}

func TestAtomicWriter_ApplyGzip(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		compressed bool
	}{
		{"Uncompressed", "test.notgz", false},
		{"Gzipped", "test.gz", true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0, 0, 0, 0, 0, 0, 0}
			dir := t.TempDir()
			path := filepath.Join(dir, test.filename)
			out, err := NewAtomicWriterCompressed(path, 0o644)
			require.NoError(t, err)
			defer out.Close()
			_, err = out.Write(data)
			require.NoError(t, err)
			require.NoError(t, out.Close())

			writtenData, err := os.ReadFile(path)
			require.NoError(t, err)
			if test.compressed {
				require.NotEqual(t, data, writtenData, "should have compressed data on disk")
			} else {
				require.Equal(t, data, writtenData, "should not have compressed data on disk")
			}

			in, err := OpenDecompressed(path)
			require.NoError(t, err)
			readData, err := io.ReadAll(in)
			require.NoError(t, err)
			require.Equal(t, data, readData)
		})
	}
}
