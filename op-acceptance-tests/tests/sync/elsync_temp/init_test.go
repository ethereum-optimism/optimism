package elsync_temp

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithNewSingleChainMultiNodeWithTestSeq(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithNoDiscovery(),
		presets.WithExecutionLayerSyncOnVerifiers(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			// For stopping derivation, not to advance safe heads
			cfg.Stopped = true
		})),
	)
}
