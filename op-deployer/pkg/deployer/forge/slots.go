package forge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// slotPool limits forge runs host-wide: a script loads every artifact, about 1.3 GB for
// contracts-bedrock. A slot is a file lock, freed by the system when its holder exits.
type slotPool struct {
	dir  string
	size int
}

// runSlots allows one forge run per two CPUs.
var runSlots = &slotPool{
	dir:  filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid())),
	size: max(1, runtime.GOMAXPROCS(0)/2),
}

// acquire waits for a free slot and returns its release function.
func (p *slotPool) acquire(ctx context.Context) (func(), error) {
	// Not MkdirAll: a planted link or file must fail the check below, not the create.
	if err := os.Mkdir(p.dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("failed to create the forge slot directory: %w", err)
	}
	if err := checkPrivateDir(p.dir); err != nil {
		return nil, err
	}
	for {
		for i := range p.size {
			f, err := os.OpenFile(filepath.Join(p.dir, strconv.Itoa(i)), os.O_CREATE|os.O_RDWR, 0o600)
			if err != nil {
				return nil, fmt.Errorf("failed to open a forge slot: %w", err)
			}
			if tryLock(f) {
				return func() { _ = f.Close() }, nil
			}
			_ = f.Close()
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to wait for a forge slot: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
