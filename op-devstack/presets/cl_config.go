package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func WithExecutionLayerSyncOnVerifiers() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
			func(_ devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
				cfg.SyncMode = sync.ELSync
			})))
}

func WithConsensusLayerSync() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
			func(_ devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
				cfg.SyncMode = sync.CLSync
			})))
}

func WithSafeDBEnabled() stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
			func(_ devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
				cfg.SafeDB = true
			})))
}
