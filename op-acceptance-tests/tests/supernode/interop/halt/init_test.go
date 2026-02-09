package halt

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates an isolated two-L2 setup with a shared supernode that has interop enabled.
// This package tests invalid message scenarios that would pollute other tests if run on a shared devnet.
func TestMain(m *testing.M) {
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m, presets.WithTwoL2SupernodeInterop(0))
}
