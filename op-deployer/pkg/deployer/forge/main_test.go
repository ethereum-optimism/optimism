package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain gives the package's tests their own slot pool. Their forge projects are tiny, and the
// host-wide pool is shared with the contracts-bedrock forge runs of other packages, which can hold
// every slot longer than these tests' timeouts.
func TestMain(m *testing.M) {
	os.Exit(runWithPrivateSlots(m))
}

func runWithPrivateSlots(m *testing.M) int {
	dir, err := os.MkdirTemp("", "op-deployer-forge-test-slots-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create the test slot directory: %v\n", err)
		return 1
	}
	defer os.RemoveAll(dir)
	runSlots = &slotPool{dir: filepath.Join(dir, "slots"), size: runSlots.size}
	return m.Run()
}
