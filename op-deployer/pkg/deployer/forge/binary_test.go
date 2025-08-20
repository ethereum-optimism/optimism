package forge

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/stretchr/testify/require"
)

// TestAutodetectBinary_ForgeBins tests that the binary can be downloaded from the
// official release channel, and that their checksums are correct.
func TestAutodetectBinary_ForgeBins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in -short mode")
	}

	// Clear out the PATH env var so it forces a download.
	t.Setenv("PATH", "")

	for target, checksum := range checksums {
		t.Run(target, func(t *testing.T) {
			lgr := testlog.Logger(t, slog.LevelInfo)
			split := strings.Split(target, "_")
			tgtOS, tgtArch := split[0], split[1]

			cacheDir := t.TempDir()
			bin, err := AutodetectBinary(
				WithURL(binaryURL(tgtOS, tgtArch)),
				WithCachePather(func() (string, error) { return cacheDir, nil }),
				WithProgressor(ioutil.NewLogProgressor(lgr, "downloading").Progressor),
				WithChecksummer(staticChecksummer(checksum)),
			)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			require.NoError(t, bin.Ensure(ctx))
		})
	}
}

func TestAutodetectBinary_Downloads(t *testing.T) {
	expChecksum, err := os.ReadFile("testdata/foundry.tgz.sha256")
	require.NoError(t, err)

	// Serve the tar archive via an HTTP test server.
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	// Prepare a cache directory within the test's temporary directory.
	cacheDir := t.TempDir()

	t.Run("download OK", func(t *testing.T) {
		var progressed atomic.Bool

		bin, err := AutodetectBinary(
			WithURL(ts.URL+"/foundry.tgz"),
			WithCachePather(func() (string, error) { return cacheDir, nil }),
			WithProgressor(func(curr, total int64) {
				progressed.Store(true)
			}),
			WithChecksummer(staticChecksummer(string(expChecksum))),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, bin.Ensure(ctx))
		require.Equal(t, path.Join(cacheDir, "forge"), bin.Path())
		require.FileExists(t, bin.Path())
		require.True(t, progressed.Load())
	})

	t.Run("invalid checksum", func(t *testing.T) {
		bin, err := AutodetectBinary(
			WithURL(ts.URL+"/foundry.tgz"),
			WithCachePather(func() (string, error) { return "not-a-path", nil }),
			WithChecksummer(staticChecksummer("beep beep")),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.ErrorContains(t, bin.Ensure(ctx), "checksum mismatch")
	})
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
		WithCachePather(func() (string, error) { return forgeDir, nil }),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, bin.Ensure(ctx))
	require.Equal(t, forgePath, bin.Path())
	require.FileExists(t, bin.Path())
}
