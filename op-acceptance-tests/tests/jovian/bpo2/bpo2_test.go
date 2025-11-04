package bpo2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/fusaka"
	jovian "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/jovian"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/params/forks"
)

func TestMain(m *testing.M) {
	resetEnvVars := fusaka.ConfigureDevstackEnvVars()
	defer resetEnvVars()
	presets.DoMain(m, stack.MakeCommon(stack.Combine(
		sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}),
		sysgo.WithDeployerOptions(
			sysgo.WithJovianAtGenesis,
			sysgo.WithDefaultBPOBlobSchedule,
			sysgo.WithForkAtL1Genesis(forks.BPO2),
		),
	)))
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
