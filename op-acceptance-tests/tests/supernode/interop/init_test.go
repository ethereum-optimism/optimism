package interop

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates a two-L2 setup with a shared supernode that has interop enabled.
// This allows testing of cross-chain message verification at each timestamp.
func TestMain(m *testing.M) {
	// Set the L2CL kind to supernode for all tests in this package
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m,
		presets.WithTwoL2SupernodeInterop(0),
		presets.WithTimeTravel(), // Enable time travel for faster tests
	)
}
