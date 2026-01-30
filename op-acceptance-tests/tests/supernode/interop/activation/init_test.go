package activation

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// InteropActivationDelay is the delay in seconds from genesis to interop activation.
// This is set to 20 seconds to allow several blocks to be produced before interop kicks in.
const InteropActivationDelay = uint64(20)

// TestMain creates a two-L2 setup with a shared supernode that has interop enabled
// AFTER genesis (delayed by InteropActivationDelay seconds).
// This allows testing that safety proceeds normally before interop activation.
func TestMain(m *testing.M) {
	// Set the L2CL kind to supernode for all tests in this package
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m, presets.WithTwoL2SupernodeInterop(InteropActivationDelay))
}
