package release

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	_ "embed"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/forge_versions.json
var versionJSON []byte

// TestStandardBinary_ForgeBins tests that the binary can be downloaded from the
// official release channel, and that their checksums are correct.
func TestGithubReleaseDownloader_Forge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in -short mode")
	}

	t.Run("supported targets", func(t *testing.T) {
		var checksums map[string]string
		err := json.Unmarshal(versionJSON, &checksums)
		require.NoError(t, err)

		cacheDir := t.TempDir()

		for target := range checksums {
			t.Run(target, func(t *testing.T) {
				split := strings.Split(target, "_")
				tgtOS, tgtArch := split[0], split[1]

				provider := NewGithubReleaseDownloader(
					"foundry-rs",
					"foundry",
					"forge",
					WithChecksummerFactory(NewStaticChecksummerFactory(checksums)),
					WithCachePather(newStaticCachePather(cacheDir)),
					WithOSGetter(newStaticOSGetter(tgtOS, tgtArch)),
					WithURLGetter(newForgeURLGetter()),
				)

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				binPath, err := provider.Get(ctx, "v1.1.0")
				require.NoError(t, err)
				require.NotEmpty(t, binPath)
			})
		}
	})

	t.Run("invalid checksum", func(t *testing.T) {
		cacheDir := t.TempDir()
		provider := NewGithubReleaseDownloader(
			"foundry-rs",
			"foundry",
			"forge",
			WithChecksummerFactory(NewStaticChecksummerFactory(map[string]string{
				"darwin_amd64": "invalidchecksum",
			})),
			WithCachePather(newStaticCachePather(cacheDir)),
			WithOSGetter(newStaticOSGetter("darwin", "amd64")),
			WithURLGetter(newForgeURLGetter()),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		binPath, err := provider.Get(ctx, "v1.4.4")
		require.ErrorContains(t, err, "checksum mismatch: checksum mismatch: expected invalidchecksum, got")
		require.Empty(t, binPath)
	})

	t.Run("missing checksum", func(t *testing.T) {
		cacheDir := t.TempDir()
		provider := NewGithubReleaseDownloader(
			"foundry-rs",
			"foundry",
			"forge",
			WithChecksummerFactory(NewStaticChecksummerFactory(map[string]string{})),
			WithCachePather(newStaticCachePather(cacheDir)),
			WithOSGetter(newStaticOSGetter("linux", "amd64")),
			WithURLGetter(newForgeURLGetter()),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		binPath, err := provider.Get(ctx, "v1.4.4")
		require.ErrorContains(t, err, "failed to get checksummer: no checksum found for os linux arch amd64")
		require.Empty(t, binPath)
	})
}

func TestGithubReleaseDownloader_OpDeployer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in -short mode")
	}

	cacheDir := t.TempDir()
	provider := NewGithubReleaseDownloader(
		"ethereum-optimism",
		"optimism",
		"op-deployer",
		WithCachePather(newStaticCachePather(cacheDir)),
		WithURLGetter(NewOPStackURLGetter()),
		WithBinaryLocator(NewOPStackBinaryLocator()),
	)

	bin, err := provider.Get(context.Background(), "0.5.0-rc.2")

	require.NoError(t, err)
	require.NotEmpty(t, bin)
}

func newStaticOSGetter(os, arch string) GithubReleaseOSGetter {
	return func() (string, string, error) {
		return os, arch, nil
	}
}

func newStaticCachePather(cachePath string) GithubReleaseCachePather {
	return func() (string, error) {
		return cachePath, nil
	}
}

func newForgeURLGetter() GithubReleaseURLGetter {
	defaultURLGetter := NewDefaultURLGetter()

	return func(owner string, repo string, name string, version string, os string, arch string) (string, error) {
		if os == "windows" {
			os = "win32"
		}

		return defaultURLGetter(owner, repo, "foundry", version, os, arch)
	}
}
