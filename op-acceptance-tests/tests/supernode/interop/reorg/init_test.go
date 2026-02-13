package reorg

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates an isolated two-L2 setup with a shared supernode that has interop enabled
// but paused at timestamp 1. This prevents automatic interop verification, allowing tests to
// control when verification happens. This package tests block invalidation and reorg scenarios
// that would pollute other tests if run on a shared devnet.
func TestMain(m *testing.M) {
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m, presets.WithTwoL2SupernodeInteropPaused(0))
}
