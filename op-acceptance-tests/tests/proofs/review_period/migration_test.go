package review_period

import (
	"testing"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSuperRootMigrationPreservesReviewPeriodDelays checks that the super-root migration, run during
// setup by NewSimpleInterop, leaves both chains' review period delays unchanged.
func TestSuperRootMigrationPreservesReviewPeriodDelays(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
		presets.WithDisputeGameFinalityDelaySeconds(disputeGameFinalityDelaySeconds),
		presets.WithWithdrawalDelaySeconds(withdrawalDelaySeconds),
	)

	var chains []reviewperiod.ChainDelays
	for _, l2 := range sys.L2Networks() {
		systemConfig := l2.Escape().Deployment().SystemConfigProxyAddr()
		delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
		t.Require().NoError(err, "failed to read the review period delays of chain %s", l2.ChainID())
		t.Require().Equal(uint64(proofMaturityDelaySeconds), delays.ProofMaturityDelaySeconds,
			"proof maturity delay of chain %s changed across the migration", l2.ChainID())
		t.Require().Equal(uint64(disputeGameFinalityDelaySeconds), delays.DisputeGameFinalityDelaySeconds,
			"dispute game finality delay of chain %s changed across the migration", l2.ChainID())
		t.Require().Equal(uint64(withdrawalDelaySeconds), delays.WithdrawalDelaySeconds,
			"DelayedWETH withdrawal delay of chain %s changed across the migration", l2.ChainID())
		chains = append(chains, delays)
	}

	// The migration makes the chains share one AnchorStateRegistry and one DelayedWETH.
	t.Require().Equal(chains[0].AnchorStateRegistry, chains[1].AnchorStateRegistry, "chains do not share an AnchorStateRegistry")
	t.Require().Equal(chains[0].DelayedWETH, chains[1].DelayedWETH, "chains do not share a DelayedWETH")
}
