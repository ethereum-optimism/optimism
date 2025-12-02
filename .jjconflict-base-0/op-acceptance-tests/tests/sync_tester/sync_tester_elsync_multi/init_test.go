package multi

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
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithSimpleWithSyncTester(),
		presets.WithELSyncActive(),
		presets.WithCompatibleTypes(compat.SysGo),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			// For stopping derivation, not to advance safe heads
			cfg.Stopped = true
		})),
	)
}
