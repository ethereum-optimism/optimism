package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

//go:embed forge-artifacts
var embedDir embed.FS

const embeddedArtifactsFile = "artifacts.tgz"

func ExtractEmbedded(destDir string) (foundry.StatDirFs, error) {
	f, err := embedDir.Open(filepath.Join("forge-artifacts", embeddedArtifactsFile))
	if err != nil {
		return nil, fmt.Errorf("could not open embedded artifacts: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	if err := ioutil.Untar(destDir, tr); err != nil {
		return nil, fmt.Errorf("failed to untar embedded artifacts: %w", err)
	}

	forgeArtifactsDir := filepath.Join(destDir, "forge-artifacts")
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

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	if err := ioutil.Untar(destDir, tr); err != nil {
		return nil, fmt.Errorf("failed to untar embedded artifacts: %w", err)
	}

	forgeArtifactsDir := filepath.Join(destDir, "out")
	if _, err := os.Stat(forgeArtifactsDir); err != nil {
		return nil, fmt.Errorf("forge-artifacts directory not found within embedded artifacts: %w", err)
	}

	return os.DirFS(forgeArtifactsDir).(foundry.StatDirFs), nil
}
