package sync_tester_hfs

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleWithSyncTester(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithHardforkSequentialActivation(rollup.Bedrock, rollup.Jovian, 15),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			// For supporting pre-delta batches
			cfg.BatchType = derive.SingularBatchType
			// For supporting pre-Fjord batches
			cfg.CompressionAlgo = derive.Zlib
		})))
}
