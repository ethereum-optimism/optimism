package forge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gofrs/flock"
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

// RunMemory is the memory of a forge slot: the peak memory of a forge run on a contracts bundle
// without the AST, as the artifacts package packs every bundle, with room for the contracts to grow.
// TestForgeRunMemory fails when a run nears it. A run on a build with the AST, which op-deployer
// uses only when it cannot copy the build, takes about three times this.
const RunMemory = 512 << 20

// peakRunMemory is the largest peak memory of a forge script run in this process.
var peakRunMemory atomic.Uint64

// PeakRunMemory returns the largest peak memory of a forge script run in this process, in bytes: 0
// before the first run, or where the peak is unknown.
func PeakRunMemory() uint64 {
	return peakRunMemory.Load()
}

// recordRunMemory raises the PeakRunMemory to peak.
func recordRunMemory(peak uint64) {
	for {
		old := peakRunMemory.Load()
		if peak <= old || peakRunMemory.CompareAndSwap(old, peak) {
			return
		}
	}
}

// slotCount returns the number of forge runs that three quarters of memLimit hold, at most two per
// CPU and at least one: one run per RunMemory, and a quarter for the rest of the host. Runs beyond
// the memory get killed, and runs beyond the CPUs take longer and hold their memory longer; two per
// CPU leave room for runs that wait on L1. Where the memory is unknown, memLimit is 0, and it allows
// one run per CPU.
func slotCount(cpus int, memLimit uint64) int {
	if memLimit == 0 {
		return max(1, cpus)
	}
	return max(1, min(2*cpus, int(memLimit/4*3/RunMemory)))
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
			slot := flock.New(filepath.Join(dir, strconv.Itoa(i)))
			// An error other than a busy slot fails: slots on NFS without a lock daemon would
			// otherwise look busy forever.
			locked, err := slot.TryLock()
			if err != nil {
				return nil, fmt.Errorf("failed to lock a forge slot in %s: %w", dir, err)
			}
			if locked {
				return func() { _ = slot.Close() }, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to wait for a forge slot: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
