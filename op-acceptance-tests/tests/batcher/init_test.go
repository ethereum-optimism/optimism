package batcher

import (
	"testing"
	"time"

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
		presets.WithNoDiscovery(),
		presets.WithTimeTravel(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			cfg.Stopped = true

			// set the blob max size to 40_000 bytes for test purposes
			cfg.MaxL1TxSize = 40_000
			cfg.TestUseMaxTxSizeForBlobs = true

			cfg.PollInterval = 1000 * time.Millisecond

			cfg.MaxChannelDuration = 50
			cfg.MaxPendingTransactions = 7
		})),
	)
}
