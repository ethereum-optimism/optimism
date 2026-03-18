package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SingleChainTwoVerifiers struct {
	SingleChainMultiNode

	L2ELC *dsl.L2ELNode
	L2CLC *dsl.L2CLNode

	TestSequencer *dsl.TestSequencer
}

// NewSingleChainTwoVerifiersWithoutCheck creates a fresh
// SingleChainTwoVerifiers target for the current test.
//
// The target is created from the runtime plus any additional preset options.
func NewSingleChainTwoVerifiersWithoutCheck(t devtest.T, opts ...Option) *SingleChainTwoVerifiers {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainTwoVerifiersWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainTwoVerifiersFromRuntime(t, sysgo.NewSingleChainTwoVerifiersRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
