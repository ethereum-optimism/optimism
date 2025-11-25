package release

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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
					WithCachePather(NewStaticCachePather(cacheDir)),
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

	t.Run("version check", func(t *testing.T) {
		var checksums map[string]string
		err := json.Unmarshal(versionJSON, &checksums)
		require.NoError(t, err)

		cacheDir := t.TempDir()

		t.Run("should fail if the command fails", func(t *testing.T) {
			provider := NewGithubReleaseDownloader(
				"foundry-rs",
				"foundry",
				"forge",
				WithChecksummerFactory(NewStaticChecksummerFactory(checksums)),
				WithCachePather(NewStaticCachePather(cacheDir)),
				WithOSGetter(NewDefaultOSGetter()),
				WithURLGetter(newForgeURLGetter()),
				WithVersionCheckerFactory(NewStaticCommandVersionCheckerFactory([]string{"-idontexist"}, parseForgeVersionOutput, NewSemverEqualityComparator())),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			binPath, err := provider.Get(ctx, "v1.1.0")
			require.ErrorContains(t, err, "version check failed for binary forge of version v1.1.0: version check failed")
			require.Empty(t, binPath)
		})

		t.Run("should fail if the versions do not match", func(t *testing.T) {
			returnStaticUnmatchingForgeVersion := func(out string) (string, error) {
				return "4.0.0", nil
			}

			provider := NewGithubReleaseDownloader(
				"foundry-rs",
				"foundry",
				"forge",
				WithChecksummerFactory(NewStaticChecksummerFactory(checksums)),
				WithCachePather(NewStaticCachePather(cacheDir)),
				WithOSGetter(NewDefaultOSGetter()),
				WithURLGetter(newForgeURLGetter()),
				WithVersionCheckerFactory(NewStaticCommandVersionCheckerFactory([]string{"-V"}, returnStaticUnmatchingForgeVersion, NewSemverEqualityComparator())),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			binPath, err := provider.Get(ctx, "v1.1.0")
			require.ErrorContains(t, err, "version check failed for binary forge of version v1.1.0: requested version v1.1.0 does not match the actual one 4.0.0")
			require.Empty(t, binPath)
		})

		t.Run("should succeed if the versions match", func(t *testing.T) {
			provider := NewGithubReleaseDownloader(
				"foundry-rs",
				"foundry",
				"forge",
				WithChecksummerFactory(NewStaticChecksummerFactory(checksums)),
				WithCachePather(NewStaticCachePather(cacheDir)),
				WithOSGetter(NewDefaultOSGetter()),
				WithURLGetter(newForgeURLGetter()),
				WithVersionCheckerFactory(NewStaticCommandVersionCheckerFactory([]string{"-V"}, parseForgeVersionOutput, NewSemverEqualityComparator())),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			binPath, err := provider.Get(ctx, "v1.1.0")
			require.NoError(t, err)
			require.NotEmpty(t, binPath)
		})
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
			WithCachePather(NewStaticCachePather(cacheDir)),
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
			WithCachePather(NewStaticCachePather(cacheDir)),
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
		WithCachePather(NewStaticCachePather(cacheDir)),
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

func newForgeURLGetter() GithubReleaseURLGetter {
	defaultURLGetter := NewDefaultURLGetter()

	return func(owner string, repo string, name string, version string, os string, arch string) (string, error) {
		if os == "windows" {
			os = "win32"
		}

		return defaultURLGetter(owner, repo, "foundry", version, os, arch)
	}
}

func parseForgeVersionOutput(out string) (string, error) {
	re := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse version tag from: %q", out)
	}
	return "v" + m[1], nil
}
