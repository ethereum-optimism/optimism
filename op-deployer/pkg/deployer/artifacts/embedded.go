package artifacts

import (
	"archive/tar"
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

// Primary filename for embedded artifacts using zstd compression (.tzst).
const embeddedArtifactsZstdShort = "artifacts.tzst"

func ExtractEmbedded(destDir string) (foundry.StatDirFs, error) {
	f, err := embedDir.Open(filepath.Join("forge-artifacts", embeddedArtifactsZstdShort))
	if err != nil {
		return nil, fmt.Errorf("could not open embedded artifacts %q: %w", embeddedArtifactsZstdShort, err)
	}
	defer f.Close()

	zr, zerr := zstd.NewReader(f)
	if zerr != nil {
		return nil, fmt.Errorf("could not create zstd reader: %w", zerr)
	}
	defer zr.Close()
	reader := io.NopCloser(zr)

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

	if !strings.HasSuffix(tarFilePath, ".tzst") {
		return nil, fmt.Errorf("unsupported file format: expected .tzst file, got %q", tarFilePath)
	}

	zr, zerr := zstd.NewReader(f)
	if zerr != nil {
		return nil, fmt.Errorf("could not create zstd reader: %w", zerr)
	}
	defer zr.Close()
	reader := io.NopCloser(zr)

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
