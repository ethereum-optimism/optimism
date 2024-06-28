package backend

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

func prepLogDBPath(chainID *big.Int, datadir string) (string, error) {
	dir, err := prepChainDir(chainID, datadir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "log.db"), nil
}

func prepChainDir(chainID *big.Int, datadir string) (string, error) {
	dir := filepath.Join(datadir, chainID.Text(10))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create chain directory %v: %w", dir, err)
	}
	return dir, nil
}
