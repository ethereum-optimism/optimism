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

// slotPool limits forge runs host-wide: at the start of a run, forge parses every artifact of the
// script's build, which takes much memory and CPU. A slot is a file lock, freed by the system when
// its holder exits.
type slotPool struct {
	// dir holds the slots. Empty means slotDir.
	dir  string
	size int
}

// runSlots allows as many forge runs as the memory of the process holds.
var runSlots = &slotPool{
	size: slotCount(runtime.GOMAXPROCS(0), memoryLimit()),
}

// slotDir returns the slot directory of this user: in the user's cache directory, or in the
// temporary directory for a user who cannot create one there, such as a uid whose home is / in a
// container.
func slotDir() string {
	if cache, err := os.UserCacheDir(); err == nil {
		dir := filepath.Join(cache, "op-deployer", "forge-slots")
		if os.MkdirAll(dir, 0o755) == nil {
			return dir
		}
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("op-deployer-forge-slots-%d", os.Getuid()))
}

// forgeRunMemory is about the peak memory of a forge run on a contracts bundle whose artifacts have
// an AST. The artifacts package packs the bundles of built projects without it, and a run then takes
// a quarter of this or less; sizing for the AST keeps a run safe on any bundle.
const forgeRunMemory = 1536 << 20

// slotCount returns the number of forge runs that three quarters of memLimit hold, at least one: one
// run per forgeRunMemory, and a quarter for the rest of the host. Runs beyond the CPUs only take
// longer, but runs beyond the memory get killed. Where the memory is unknown, memLimit is 0, and it
// allows one run per CPU.
func slotCount(cpus int, memLimit uint64) int {
	if memLimit == 0 {
		return max(1, cpus)
	}
	return max(1, int(memLimit/4*3/forgeRunMemory))
}

// acquire waits for a free slot and returns its release function.
func (p *slotPool) acquire(ctx context.Context) (func(), error) {
	dir := p.dir
	if dir == "" {
		dir = slotDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create the forge slot directory: %w", err)
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
