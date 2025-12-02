package pectra

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/jovian"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, stack.Combine(stack.MakeCommon(stack.Combine(
		sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}),
		sysgo.WithDeployerOptions(sysgo.WithJovianAtGenesis),
	))))
}

func TestDAFootprint(t *testing.T) {
	jovian.TestDAFootprint(t)
}

func TestMinBaseFee(t *testing.T) {
	jovian.TestMinBaseFee(t)
}

func TestOperatorFee(t *testing.T) {
	jovian.TestOperatorFee(t)
}
