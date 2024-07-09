package backend

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func prepLogDBPath(chainID types.ChainID, datadir string) (string, error) {
	dir, err := prepChainDir(chainID, datadir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "log.db"), nil
}

func prepChainDir(chainID types.ChainID, datadir string) (string, error) {
	dir := filepath.Join(datadir, chainID.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create chain directory %v: %w", dir, err)
	}
	return dir, nil
}
