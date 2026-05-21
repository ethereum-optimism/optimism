package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// NewMinimalNoFaultProofs is a Minimal preset that skips the proposer and
// challenger. It is intended for tests that only exercise the sequencer +
// batcher + derivation loop (e.g. sequencer control-plane features). Skipping
// the challenger avoids requiring cannon prestate artifacts, which are
// expensive to build locally.
//
// DisputeGameFactory() on the returned Minimal is not usable because the
// challenger config is nil.
func NewMinimalNoFaultProofs(t devtest.T, opts ...Option) *Minimal {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewMinimalNoFaultProofs", opts, minimalPresetSupportedOptionKinds)
	out := minimalFromRuntime(t, sysgo.NewMinimalNoFaultProofsRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
