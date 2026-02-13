package presets

import (
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SimpleWithSyncTester struct {
	Minimal

	SyncTester     *dsl.SyncTester
	SyncTesterL2EL *dsl.L2ELNode
	L2CL2          *dsl.L2CLNode
}

// NewSimpleWithSyncTester creates a fresh SimpleWithSyncTester target for the current
// test.
//
// The target is created from the runtime plus any additional preset options.
func NewSimpleWithSyncTester(t devtest.T, opts ...Option) *SimpleWithSyncTester {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSimpleWithSyncTester", opts, simpleWithSyncTesterPresetSupportedOptionKinds)
	out := simpleWithSyncTesterFromRuntime(t, sysgo.NewSimpleWithSyncTesterRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

func WithHardforkSequentialActivation(startFork, endFork forks.Name, delta uint64) Option {
	return WithDeployerOptions(sysgo.WithHardforkSequentialActivation(startFork, endFork, &delta))
}
