package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

//go:embed forge-artifacts
var embedDir embed.FS

// Primary filenames for embedded artifacts. Prefer zstd (.tzst); support legacy gzip (.tgz).
const embeddedArtifactsZstdShort = "artifacts.tzst"
const embeddedArtifactsGzip = "artifacts.tgz"

func ExtractEmbedded(destDir string) (foundry.StatDirFs, error) {
	var (
		f    io.ReadCloser
		err  error
		comp string
	)
	// Prefer zstd, fall back to gzip for legacy bundles
	if rf, openErr := embedDir.Open(filepath.Join("forge-artifacts", embeddedArtifactsZstdShort)); openErr == nil {
		f = rf
		comp = "zstd"
	} else if rf, openErr2 := embedDir.Open(filepath.Join("forge-artifacts", embeddedArtifactsGzip)); openErr2 == nil {
		f = rf
		comp = "gzip"
	} else {
		return nil, fmt.Errorf("could not open embedded artifacts: tried %q, %q", embeddedArtifactsZstdShort, embeddedArtifactsGzip)
	}
	defer f.Close()

	var reader io.ReadCloser
	switch comp {
	case "zstd":
		zr, zerr := zstd.NewReader(f)
		if zerr != nil {
			return nil, fmt.Errorf("could not create zstd reader: %w", zerr)
		}
		reader = io.NopCloser(zr)
		defer zr.Close()
	default:
		gzr, gerr := gzip.NewReader(f)
		if gerr != nil {
			return nil, fmt.Errorf("could not create gzip reader: %w", gerr)
		}
		reader = gzr
		defer gzr.Close()
	}

	// Untar into a unique subdirectory to avoid collisions with pre-existing paths
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure destination dir: %w", err)
	}
	untarPath, err := os.MkdirTemp(destDir, "bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp untar dir: %w", err)
	}

	tr := tar.NewReader(reader)
	if err := ioutil.Untar(untarPath, tr); err != nil {
		return nil, fmt.Errorf("failed to untar embedded artifacts: %w", err)
	}

	forgeArtifactsDir := filepath.Join(untarPath, "forge-artifacts")
	if _, err := os.Stat(forgeArtifactsDir); err != nil {
		return nil, fmt.Errorf("forge-artifacts directory not found within embedded artifacts: %w", err)
	}

	return os.DirFS(forgeArtifactsDir).(foundry.StatDirFs), nil
}

func ExtractFromFile(destDir string, tarFilePath string) (foundry.StatDirFs, error) {
	f, err := os.Open(tarFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open tar file: %w", err)
	}
	defer f.Close()

	var reader io.ReadCloser
	if strings.HasSuffix(tarFilePath, ".tar.zst") || strings.HasSuffix(tarFilePath, ".tzst") || strings.HasSuffix(tarFilePath, ".zst") {
		zr, zerr := zstd.NewReader(f)
		if zerr != nil {
			return nil, fmt.Errorf("could not create zstd reader: %w", zerr)
		}
		reader = io.NopCloser(zr)
		defer zr.Close()
	} else {
		gzr, gerr := gzip.NewReader(f)
		if gerr != nil {
			return nil, fmt.Errorf("could not create gzip reader: %w", gerr)
		}
		reader = gzr
		defer gzr.Close()
	}

	// Untar into a unique subdirectory to avoid collisions with pre-existing paths
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure destination dir: %w", err)
	}
	untarPath, err := os.MkdirTemp(destDir, "bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp untar dir: %w", err)
	}

	tr := tar.NewReader(reader)
	if err := ioutil.Untar(untarPath, tr); err != nil {
		return nil, fmt.Errorf("failed to untar embedded artifacts: %w", err)
	}

	forgeArtifactsDir := filepath.Join(untarPath, "out")
	if _, err := os.Stat(forgeArtifactsDir); err != nil {
		return nil, fmt.Errorf("forge-artifacts directory not found within embedded artifacts: %w", err)
	}

	return os.DirFS(forgeArtifactsDir).(foundry.StatDirFs), nil
}
