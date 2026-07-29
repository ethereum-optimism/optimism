package fusaka

import (
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/params/forks"
)

func newMinimalFusaka(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(t,
		L1GethOption(),
		presets.WithDeployerOptions(
			sysgo.WithDefaultBPOBlobSchedule,
			// Make the BPO fork happen after Osaka so we can easily use geth's eip4844.CalcBlobFee
			// to calculate the blob base fee using the Osaka parameters.
			sysgo.WithForkAtL1Offset(forks.Osaka, 0),
			sysgo.WithForkAtL1Offset(forks.BPO1, 1),
		),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = flags.BlobsType
			cfg.TxMgrConfig.CellProofTime = 0 // Force cell proofs to be used
		}),
	)
}
