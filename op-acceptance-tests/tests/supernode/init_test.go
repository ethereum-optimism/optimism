package supernode

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates a two-L2 setup against the shared backend
func TestMain(m *testing.M) {
	// TODO(inphi): Fix this test. A supernode isn't functionally equivalent to an op-node. This test should explicitly use the supernode preset rather than hack around by setting the l2cl_kind supernode envar.
	//_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	presets.DoMain(m, presets.WithTwoL2Supernode())
}
