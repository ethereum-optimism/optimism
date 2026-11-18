package artifacts

import (
	"archive/tar"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	return extractZstdBundle(destDir, f)
}

func extractZstdBundle(destDir string, r io.Reader) (foundry.StatDirFs, error) {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("could not create zstd reader: %w", err)
	}
	defer zr.Close()

	// Untar into a unique subdirectory to avoid collisions with pre-existing paths.
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure destination dir: %w", err)
	}
	untarPath, err := os.MkdirTemp(destDir, "bundle-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp untar dir: %w", err)
	}
	RegisterForCleanup(untarPath)

	if err := ioutil.Untar(untarPath, tar.NewReader(zr)); err != nil {
		return nil, fmt.Errorf("failed to untar artifacts: %w", err)
	}
	artifactsDir, err := projectArtifacts(untarPath)
	if err != nil {
		return nil, err
	}
	return os.DirFS(artifactsDir).(foundry.StatDirFs), nil
}
