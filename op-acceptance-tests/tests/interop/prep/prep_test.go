package msg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestUnscheduledInterop runs against an interop system (i.e. op-nodes are managed by op-supervisor),
// before interop is scheduled with an actual hardfork time.
// And then confirms we can finalize the chains.
func TestUnscheduledInterop(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	t.Logger().Info("Checking that chain A and B can sync, even though interop is not scheduled")
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.Finalized, 5, 100),
		sys.L2CLB.AdvancedFn(types.Finalized, 5, 100),
	)
	// Note: supervisor sync status won't be ready till after interop activation.
}
