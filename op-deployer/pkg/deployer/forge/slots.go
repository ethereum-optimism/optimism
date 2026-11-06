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
	// dir holds the slots. Empty means slotDir.
	dir  string
	size int
}

// runSlots allows one forge run per two CPUs, within the memory limit of the process's container.
var runSlots = &slotPool{
	size: slotCount(runtime.GOMAXPROCS(0), memoryLimit()),
}

// slotDir returns the slot directory of this user. It is in the user's cache directory, which no
// other user can create first. A user without a private one there, such as a uid whose home is /
// in a container, gets a directory in the shared temporary directory, which acquire checks.
func slotDir() string {
	if cache, err := os.UserCacheDir(); err == nil {
		parent := filepath.Join(cache, "op-deployer")
		if err := os.MkdirAll(parent, 0o700); err == nil && checkPrivateDir(parent) == nil {
			return filepath.Join(parent, "forge-slots")
		}
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid()))
}

// forgeRunMemory is about the memory that a forge run on contracts-bedrock takes.
const forgeRunMemory = 1536 << 20

// slotCount returns the number of forge runs that cpus and memLimit allow, at least one: one run
// per two CPUs and per forgeRunMemory. A memLimit of zero means no limit.
func slotCount(cpus int, memLimit uint64) int {
	n := cpus / 2
	if memLimit > 0 {
		n = min(n, int(memLimit/forgeRunMemory))
	}
	return max(1, n)
}

// acquire waits for a free slot and returns its release function.
func (p *slotPool) acquire(ctx context.Context) (func(), error) {
	dir := p.dir
	if dir == "" {
		dir = slotDir()
	}
	// Not MkdirAll: a planted link or file must fail the check below, not the create.
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("failed to create the forge slot directory: %w", err)
	}
	if err := checkPrivateDir(dir); err != nil {
		return nil, err
	}
	for {
		for i := range p.size {
			f, err := os.OpenFile(filepath.Join(dir, strconv.Itoa(i)), os.O_CREATE|os.O_RDWR, 0o600)
			if err != nil {
				return nil, fmt.Errorf("failed to open a forge slot: %w", err)
			}
			locked, err := tryLock(f)
			if locked {
				return func() { _ = f.Close() }, nil
			}
			_ = f.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to lock a forge slot in %s: %w", dir, err)
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to wait for a forge slot: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
