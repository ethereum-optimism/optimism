package pectra

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/jovian/bpo2/joviantest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func setupPectra(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(t,
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
	)
}

func TestDAFootprint(gt *testing.T) {
	joviantest.RunDAFootprint(gt, setupPectra)
}

func TestMinBaseFee(gt *testing.T) {
	joviantest.RunMinBaseFee(gt, setupPectra)
}

func TestOperatorFee(gt *testing.T) {
	joviantest.RunOperatorFee(gt, setupPectra)
}
