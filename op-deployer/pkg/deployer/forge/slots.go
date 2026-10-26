package forge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// slotPool limits the forge runs on the host, across processes. A forge script loads every artifact
// of the project, about 1.3 GB for contracts-bedrock, and test suites run many pipelines at once in
// many processes. A slot is a lock on a file in dir, which the system frees when its holder exits.
type slotPool struct {
	dir  string
	size int
}

// runSlots allows one forge run per two CPUs of this process.
var runSlots = &slotPool{
	dir:  filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid())),
	size: max(1, runtime.GOMAXPROCS(0)/2),
}

// acquire waits for a free slot, and returns the function that frees it.
func (p *slotPool) acquire(ctx context.Context) (func(), error) {
	if err := os.MkdirAll(p.dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create the forge slot directory: %w", err)
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
