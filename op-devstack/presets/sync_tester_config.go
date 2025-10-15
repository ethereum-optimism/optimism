package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func WithSyncTesterELInitialState(fcu eth.FCUState) stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalSyncTesterELOption(sysgo.SyncTesterELOptionFn(
			func(_ devtest.P, id stack.L2ELNodeID, cfg *sysgo.SyncTesterELConfig) {
				cfg.FCUState = fcu
			})))
}

func WithELSyncTarget(elSyncTarget uint64) stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalSyncTesterELOption(sysgo.SyncTesterELOptionFn(
			func(_ devtest.P, id stack.L2ELNodeID, cfg *sysgo.SyncTesterELConfig) {
				cfg.ELSyncActive = true
				cfg.ELSyncTarget = elSyncTarget
			})))
}
