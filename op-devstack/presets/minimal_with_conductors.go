package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MinimalWithConductors struct {
	*Minimal

	ConductorSets map[eth.ChainID]dsl.ConductorSet
}

// NewMinimalWithConductors creates a fresh MinimalWithConductors target for the current
// test.
//
// The target is created from the runtime plus any additional preset options.
func NewMinimalWithConductors(t devtest.T, opts ...Option) *MinimalWithConductors {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewMinimalWithConductors", opts, minimalWithConductorsPresetSupportedOptionKinds)
	out := minimalWithConductorsFromRuntime(t, sysgo.NewMinimalWithConductorsRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
