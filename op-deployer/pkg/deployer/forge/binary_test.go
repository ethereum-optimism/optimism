package forge

import (
	"context"
	"fmt"
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

// TestStandardBinary_ForgeBins tests that the binary can be downloaded from the
// official release channel, and that their checksums are correct.
func TestStandardBinary_ForgeBins(t *testing.T) {
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
			bin, err := NewStandardBinary(
				WithURL(binaryURL(tgtOS, tgtArch)),
				WithCachePather(func() (string, error) { return cacheDir, nil }),
				WithProgressor(ioutil.NewLogProgressor(lgr, "downloading").Progressor),
				WithChecksummer(staticChecksummer(checksum)),
			)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			require.NoError(t, bin.Ensure(ctx))
		})
	}
}

func TestStandardBinary_Downloads(t *testing.T) {
	expChecksum, err := os.ReadFile("testdata/foundry.tgz.sha256")
	require.NoError(t, err)

	// Serve the tar archive via an HTTP test server.
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	// Prepare a cache directory within the test's temporary directory.
	cacheDir := t.TempDir()

	t.Run("download OK", func(t *testing.T) {
		var progressed atomic.Bool

		bin, err := NewStandardBinary(
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
		bin, err := NewStandardBinary(
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

func TestStandardBinary_OnPath(t *testing.T) {
	expChecksum, err := os.ReadFile("testdata/foundry.tgz.sha256")
	require.NoError(t, err)

	// Serve the test tarball so we can force the download path.
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	makeForge := func(dir, versionLine string) string {
		fp := path.Join(dir, "forge")
		script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "%s"
  exit 0
fi
exit 1
`, versionLine)
		require.NoError(t, os.WriteFile(fp, []byte(script), 0o777))
		require.NoError(t, os.Chmod(fp, 0o777))
		return fp
	}

	cases := []struct {
		name          string
		versionLine   string
		expectUsePath bool
	}{
		{
			name:          "match_tag",
			versionLine:   fmt.Sprintf("forge Version: %s-%s", strings.TrimPrefix(StandardVersion, "v"), StandardVersion),
			expectUsePath: true,
		},
		{
			name:          "dev_tag",
			versionLine:   fmt.Sprintf("forge Version: %s-dev", strings.TrimPrefix(StandardVersion, "v")),
			expectUsePath: true,
		},
		{
			name:          "non_standard_tag",
			versionLine:   "forge Version: 0.0.0-v0.0.0",
			expectUsePath: false,
		},
		{
			name:          "garbage_output",
			versionLine:   "forge something unexpected",
			expectUsePath: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			forgeDir := t.TempDir()
			forgePath := makeForge(forgeDir, tc.versionLine)
			t.Setenv("PATH", forgeDir)

			cacheDir := t.TempDir()
			bin, err := NewStandardBinary(
				WithURL(ts.URL+"/foundry.tgz"),
				WithCachePather(func() (string, error) { return cacheDir, nil }),
				WithChecksummer(staticChecksummer(string(expChecksum))),
			)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			require.NoError(t, bin.Ensure(ctx))

			if tc.expectUsePath {
				require.Equal(t, forgePath, bin.Path())
				require.NoFileExists(t, path.Join(cacheDir, "forge"))
			} else {
				require.Equal(t, path.Join(cacheDir, "forge"), bin.Path())
				require.FileExists(t, bin.Path())
			}
		})
	}
}
