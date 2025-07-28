package execmultiplexing

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestMain(m *testing.M) {
	nodes := 3
	presets.DoMain(m, presets.WithMultiELNodes(nodes),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}

func TestExecMultiplexing(gt *testing.T) {
	t := devtest.SerialT(gt)

	nodes := 3
	sys := presets.NewMultiELNodes(t, nodes)

	dsl.CheckAll(t, sys.L2CL.AdvancedFn(types.LocalUnsafe, 5, 30))
}
