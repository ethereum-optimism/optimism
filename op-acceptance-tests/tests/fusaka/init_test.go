package fusaka

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/params/forks"
)

func TestMain(m *testing.M) {
	resetEnvVars := ConfigureDevstackEnvVars()
	defer resetEnvVars()

	presets.DoMain(m, stack.MakeCommon(stack.Combine[*sysgo.Orchestrator](
		sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}),
		sysgo.WithDeployerOptions(
			sysgo.WithDefaultBPOBlobSchedule,
			sysgo.WithForkAtL1Genesis(forks.BPO1),
		),
		sysgo.WithBatcherOption(func(_ stack.L2BatcherID, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = flags.BlobsType
			cfg.TxMgrConfig.CellProofTime = 0 // Force cell proofs to be used
		}),
	)))
}
