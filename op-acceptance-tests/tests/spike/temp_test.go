package safeheaddb_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestTemp(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, 3, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, 3, 30))

	sys.L2CLB.Matched(sys.L2CL, types.LocalUnsafe, 30)
}
