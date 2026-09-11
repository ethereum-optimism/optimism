package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type MinimalWithConductors struct {
	*Minimal

	Conductors dsl.ConductorSet
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
