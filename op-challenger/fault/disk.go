package fault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/exp/slices"
)

const gameDirPrefix = "game-"

// diskManager coordinates the storage of game data on disk.
type diskManager struct {
	datadir string
}

func newDiskManager(dir string) *diskManager {
	return &diskManager{datadir: dir}
}

func (d *diskManager) DirForGame(addr common.Address) string {
	return filepath.Join(d.datadir, gameDirPrefix+addr.Hex())
}

func (d *diskManager) RemoveAllExcept(keep []common.Address) error {
	entries, err := os.ReadDir(d.datadir)
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}
	var result error
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), gameDirPrefix) {
			// Skip files and directories that don't have the game directory prefix.
			// While random content shouldn't be in our datadir, we want to avoid
			// deleting things like OS generated files.
			continue
		}
		name := entry.Name()[len(gameDirPrefix):]
		addr := common.HexToAddress(name)
		if addr == (common.Address{}) {
			// Ignore directories with non-address names.
			continue
		}
		if slices.Contains(keep, addr) {
			// Preserve data for games we should keep.
			continue
		}
		if err := os.RemoveAll(filepath.Join(d.datadir, entry.Name())); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
