package bpo2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/fusaka"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/jovian/bpo2/joviantest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/params/forks"
)

func setupBPO2(t devtest.T) *presets.Minimal {
	resetEnvVars := fusaka.ConfigureDevstackEnvVars()
	t.Cleanup(resetEnvVars)
	return presets.NewMinimal(t,
		presets.WithDeployerOptions(
			sysgo.WithJovianAtGenesis,
			sysgo.WithDefaultBPOBlobSchedule,
			sysgo.WithForkAtL1Genesis(forks.BPO2),
		),
	)
}

func TestDAFootprint(gt *testing.T) {
	joviantest.RunDAFootprint(gt, setupBPO2)
}

func TestMinBaseFee(gt *testing.T) {
	joviantest.RunMinBaseFee(gt, setupBPO2)
}

func TestOperatorFee(gt *testing.T) {
	joviantest.RunOperatorFee(gt, setupBPO2)
}
