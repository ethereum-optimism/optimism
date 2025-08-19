package forge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutodetectBinary_Downloads(t *testing.T) {
	// Clear out the PATH env var so it forces a download.
	t.Setenv("PATH", "")

	// Serve the tar archive via an HTTP test server.
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	// Prepare a cache directory within the test's temporary directory.
	cacheDir := t.TempDir()

	var progressed atomic.Bool

	bin, err := AutodetectBinary(
		WithURL(ts.URL+"/foundry.tgz"),
		WithCacheDirProvider(func() (string, error) { return cacheDir, nil }),
		WithProgressor(func(curr, total int64) {
			progressed.Store(true)
		}),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, bin.Ensure(ctx))
	require.Equal(t, path.Join(cacheDir, "forge"), bin.Path())
	require.FileExists(t, bin.Path())
	require.True(t, progressed.Load())
}

func TestAutodetectBinary_OnPath(t *testing.T) {
	forgeDir := t.TempDir()
	forgePath := path.Join(forgeDir, "forge")
	_, err := os.Create(forgePath)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(forgePath, 0777))

	// Set the PATH env var to the directory we just created to prevent a download.
	t.Setenv("PATH", forgeDir)

	bin, err := AutodetectBinary(
		WithURL(""),
		WithCacheDirProvider(func() (string, error) { return forgeDir, nil }),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, bin.Ensure(ctx))
	require.Equal(t, forgePath, bin.Path())
	require.FileExists(t, bin.Path())
}
