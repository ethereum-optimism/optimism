package elsync

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainMultiNode(),
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithSyncModeReqRespSync(),
		presets.WithNoDiscovery(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		})),
	)
}
