package fault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
)

// diskManager coordinates
type diskManager struct {
	datadir string
}

func newDiskManager(dir string) *diskManager {
	return &diskManager{datadir: dir}
}

func (d *diskManager) DirForGame(addr common.Address) string {
	return filepath.Join(d.datadir, addr.Hex())
}

func (d *diskManager) RemoveGameData(addr common.Address) error {
	dir := d.DirForGame(addr)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove dir %v: %w", dir, err)
	}
	return nil
}
