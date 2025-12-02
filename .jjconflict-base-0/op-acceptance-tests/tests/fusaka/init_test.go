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
			// Make the BPO fork happen after Osaka so we can easily use geth's eip4844.CalcBlobFee
			// to calculate the blob base fee using the Osaka parameters.
			sysgo.WithForkAtL1Offset(forks.Osaka, 0),
			sysgo.WithForkAtL1Offset(forks.BPO1, 1),
		),
		sysgo.WithBatcherOption(func(_ stack.L2BatcherID, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = flags.BlobsType
			cfg.TxMgrConfig.CellProofTime = 0 // Force cell proofs to be used
		}),
	)))
}
