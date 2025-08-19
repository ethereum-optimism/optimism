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

func ExtractEmbedded(dir string) (foundry.StatDirFs, error) {
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
	if err := ioutil.Untar(dir, tr); err != nil {
		return nil, fmt.Errorf("failed to untar embedded artifacts: %w", err)
	}

	return os.DirFS(dir).(foundry.StatDirFs), nil
}
