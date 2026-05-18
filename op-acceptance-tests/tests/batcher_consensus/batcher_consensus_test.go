package batcher_consensus

import (
	"testing"

	batcher "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func newBatchConsensusPOC(t devtest.T, proofValid bool) *presets.Minimal {
	return presets.NewMinimalNoFaultProofs(t,
		presets.WithDeployerOptions(sysgo.WithEcotoneAtGenesis),
		presets.WithBatchConsensusMockVerifier(sysgo.DefaultBatchConsensusMockVerifierAddress),
		presets.WithBatchConsensusCommonwareSidecarResult(proofValid),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = batcherFlags.BlobsType
		}),
	)
}

func TestBatchConsensusValidProofAdvancesSafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBatchConsensusPOC(t, true)

	alice := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	alice.Transfer(bob.Address(), eth.OneWei)

	sys.L2CL.Advanced(supervisortypes.LocalSafe, 1, 60)
}

func TestBatchConsensusInvalidProofDoesNotAdvanceSafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBatchConsensusPOC(t, false)

	alice := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	alice.Transfer(bob.Address(), eth.OneWei)

	sys.L2CL.Advanced(supervisortypes.LocalUnsafe, 1, 30)
	sys.L2CL.NotAdvanced(supervisortypes.LocalSafe, 10)
}
