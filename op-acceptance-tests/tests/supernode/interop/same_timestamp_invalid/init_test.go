package same_timestamp_invalid

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates an isolated two-L2 setup with a shared supernode that has interop enabled.
// This package tests that executing messages referencing initiating messages from the same
// timestamp are correctly detected as invalid and replaced.
func TestMain(m *testing.M) {
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m, presets.WithTwoL2SupernodeInterop(0))
}
