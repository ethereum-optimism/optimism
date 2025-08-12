package jovian

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithMinimal(),
		WithJovianEnabled(),
	)
}

// WithJovianEnabled creates a preset option that enables Jovian hardfork at genesis
func WithJovianEnabled() stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(WithJovianAtGenesis()))
}

// Helper function to activate Jovian at genesis
func WithJovianAtGenesis() sysgo.DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtGenesis(rollup.Jovian)
		}
	}
}
