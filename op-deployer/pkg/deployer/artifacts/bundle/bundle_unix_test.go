//go:build unix

package bundle

import (
	"io"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrite_SkipsPipes(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, base, map[string]string{CompileCacheFile: testCompileCache})
	require.NoError(t, syscall.Mkfifo(filepath.Join(base, "pipe"), 0o644))
	require.NoError(t, Write(io.Discard, base, nil, ""))
}
