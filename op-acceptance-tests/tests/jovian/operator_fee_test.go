package jovian

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

func TestOperatorFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	t.Require().NoError(err, "Jovian fork must be active for this test")
	dsl.RunOperatorFeeTest(t, sys.L2Chain, sys.L1EL, sys.FunderL1, sys.FunderL2)
}
