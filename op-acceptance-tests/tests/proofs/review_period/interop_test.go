package review_period

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestLoweringOneChainsProofMaturityOnlyAffectsThatChain checks that in an interop set, which shares the
// ETHLockbox, AnchorStateRegistry and DelayedWETH, proof maturity stays per chain: lowering it on chain A
// lets A's withdrawal finalize while B's, proven alongside it, waits out its own unchanged maturity.
func TestLoweringOneChainsProofMaturityOnlyAffectsThatChain(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(
			sysgo.WithProofMaturityDelaySeconds(2*24*60*60),
			// Short game clocks and finality delay, so only the proof maturity blocks finalization.
			// The game requires max(2*clockExtension, clockExtension+preimageOracleChallengePeriod) <= maxClockDuration.
			sysgo.WithDisputeGameFinalityDelaySeconds(4),
			sysgo.WithFaultGameMaxClockDuration(40),
			sysgo.WithFaultGameClockExtension(1),
			sysgo.WithPreimageOracleChallengePeriod(10),
		),
	)
	sys.L1Network.WaitForOnline()

	l1User := sys.FunderL1.NewFundedEOA(eth.HalfEther)
	proveOn := func(l2 *dsl.L2Network, el *dsl.L2ELNode) *dsl.Withdrawal {
		bridge := sys.StandardBridge(l2)
		bridge.Deposit(eth.OneTenthEther, l1User)
		l2.WaitForBlock()
		withdrawal := bridge.InitiateWithdrawal(eth.OneHundredthEther, l1User.AsEL(el))
		withdrawal.Prove(l1User)
		return withdrawal
	}
	withdrawalA := proveOn(sys.L2ChainA, sys.L2ELA)
	withdrawalB := proveOn(sys.L2ChainB, sys.L2ELB)
	sys.AdvanceTime(sys.StandardBridge(sys.L2ChainA).GameResolutionDelay())
	withdrawalA.WaitForDisputeGameResolved()
	withdrawalB.WaitForDisputeGameResolved()

	systemConfigA := sys.L2ChainA.Escape().Deployment().SystemConfigProxyAddr()
	delaysA, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfigA, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read chain A's review period delays")
	newDelayOwner(t, sys.L1EL, sys.L1Network.ChainID(), sys.L2ChainA, sys.FunderL1).SetProofMaturityDelay(delaysA, 24*60*60)

	// A day later, A's withdrawal is past its new maturity and B's is still inside its two days.
	sys.AdvanceTime(24 * time.Hour)
	t.Require().NoError(withdrawalA.CheckFinalizable(l1User), "chain A's withdrawal is not finalizable after lowering its proof maturity")
	t.Require().ErrorContains(errutil.TryAddRevertReason(withdrawalB.CheckFinalizable(l1User)),
		hexutil.Encode(dsl.ErrorSelector("OptimismPortal_ProofNotOldEnough()")), "chain B's proof should still be too young")
	withdrawalA.Finalize(l1User)

	// Another day later, B's withdrawal is past its unchanged two days and finalizes too.
	sys.AdvanceTime(24 * time.Hour)
	t.Require().NoError(withdrawalB.CheckFinalizable(l1User), "chain B's withdrawal is not finalizable after its unchanged proof maturity")
	withdrawalB.Finalize(l1User)
}
