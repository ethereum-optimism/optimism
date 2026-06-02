package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// SingleSupernodeWithSyncTester mirrors SimpleWithSyncTester but replaces the
// verifier op-node CL with a verifier-mode supernode VN. Interop is always
// enabled; activation is controlled by WithInteropActivationDelay (0 = genesis).
type SingleSupernodeWithSyncTester struct {
	Minimal

	Supernode      *dsl.Supernode
	SupernodeCL    *dsl.L2CLNode // per-chain VN proxy
	SyncTester     *dsl.SyncTester
	SyncTesterL2EL *dsl.L2ELNode

	GenesisTime           uint64
	InteropActivationTime uint64
	DelaySeconds          uint64
}

func NewSingleSupernodeWithSyncTester(t devtest.T, opts ...Option) *SingleSupernodeWithSyncTester {
	presetCfg, presetOpts := collectSupportedPresetConfig(
		t,
		"NewSingleSupernodeWithSyncTester",
		opts,
		singleSupernodeWithSyncTesterPresetSupportedOptionKinds,
	)
	out := singleSupernodeWithSyncTesterFromRuntime(t, sysgo.NewSingleSupernodeWithSyncTesterRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
