package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestWriteArchiveIsReproducible(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "nested")
	require.NoError(t, os.Mkdir(nested, 0o755))
	aJSON := filepath.Join(base, "a.json")
	bJSON := filepath.Join(nested, "b.json")
	require.NoError(t, os.WriteFile(aJSON, []byte("alpha"), 0o644))
	require.NoError(t, os.WriteFile(bJSON, []byte("beta"), 0o644))

	archive := func(modTime time.Time, fileMode, dirMode os.FileMode) []byte {
		for _, file := range []string{aJSON, bJSON} {
			require.NoError(t, os.Chtimes(file, modTime, modTime))
			require.NoError(t, os.Chmod(file, fileMode))
		}
		require.NoError(t, os.Chtimes(nested, modTime, modTime))
		require.NoError(t, os.Chmod(nested, dirMode))
		var out bytes.Buffer
		require.NoError(t, writeArchive(&out, base, "abc123"))
		return out.Bytes()
	}

	first := archive(time.Unix(1_000_000, 0), 0o644, 0o755)
	second := archive(time.Unix(2_000_000, 0), 0o640, 0o750)
	require.Equal(t, first, second)

	zr, err := zstd.NewReader(bytes.NewReader(first))
	require.NoError(t, err)
	defer zr.Close()

	contents := make(map[string]string)
	var names []string
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
		require.True(t, hdr.ModTime.Equal(time.Unix(0, 0)), "%s has modification time %s", hdr.Name, hdr.ModTime)
		require.Zero(t, hdr.Uid, hdr.Name)
		require.Zero(t, hdr.Gid, hdr.Name)
		require.Empty(t, hdr.Uname, hdr.Name)
		require.Empty(t, hdr.Gname, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			require.Equal(t, int64(0o755), hdr.Mode, hdr.Name)
			continue
		}
		require.Equal(t, int64(0o644), hdr.Mode, hdr.Name)
		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		contents[hdr.Name] = string(content)
	}
	require.Equal(t, []string{"a.json", "nested", "nested/b.json", "COMMIT"}, names)
	require.Equal(t, map[string]string{"a.json": "alpha", "nested/b.json": "beta", "COMMIT": "abc123"}, contents)
}
