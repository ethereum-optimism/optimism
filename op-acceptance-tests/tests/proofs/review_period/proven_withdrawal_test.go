package review_period

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// gameClaimValidity binds the AnchorStateRegistry check the portal runs after proof maturity.
type gameClaimValidity struct {
	IsGameClaimValid func(game common.Address) bindings.TypedCall[bool] `sol:"isGameClaimValid"`
}

// provenWithdrawal returns the portal's record of the proof prover submitted for the withdrawal.
func provenWithdrawal(t devtest.T, sys *presets.Minimal, portal common.Address, w *dsl.Withdrawal, prover *dsl.EOA) bindings.ProvenWithdrawalsResult {
	l2, err := ethclient.DialContext(t.Ctx(), sys.L2EL.Escape().UserRPC())
	t.Require().NoError(err, "failed to dial the L2 RPC")
	defer l2.Close()
	receipt, err := l2.TransactionReceipt(t.Ctx(), w.InitiateTxHash())
	t.Require().NoError(err, "failed to get the withdrawal's L2 receipt")
	ev, err := withdrawals.ParseMessagePassed(receipt)
	t.Require().NoError(err, "the withdrawal's L2 receipt has no MessagePassed event")

	portalBindings := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L1EL.EthClient()), bindings.WithTo(portal), bindings.WithTest(t))
	proven, err := contractio.Read(portalBindings.ProvenWithdrawals(ev.WithdrawalHash, prover.Address()), t.Ctx())
	t.Require().NoError(err, "failed to read the proven withdrawal")
	return proven
}

// TestWithdrawalProvenBeforeOPCMUpgradeKeepsItsProofMaturity checks that a withdrawal proven before an
// OPCM upgrade still can't finalize before its original proof maturity, and finalizes after it without
// being proven again.
func TestWithdrawalProvenBeforeOPCMUpgradeKeepsItsProofMaturity(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newPermissionedMinimal(t,
		sysgo.WithFinalizationPeriodSeconds(1),
		sysgo.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
	)
	bridge := sys.StandardBridge()
	proofNotOldEnough := hexutil.Encode(dsl.ErrorSelector("OptimismPortal_ProofNotOldEnough()"))

	user := sys.FunderL1.NewFundedEOA(eth.OneThirdEther)
	bridge.Deposit(eth.OneTenthEther, user) // funds the ETHLockbox
	withdrawal := bridge.InitiateWithdrawal(eth.OneHundredthEther, user.AsEL(sys.L2EL))
	sys.DisputeGameFactory().WaitForGame()
	withdrawal.Prove(user)
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Before the upgrade, the game is valid and only the proof's age blocks the withdrawal.
	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")
	sys.AdvanceTime(time.Duration(delays.DisputeGameFinalityDelaySeconds+1) * time.Second)
	game := sys.DisputeGameFactory().GameAtIndex(withdrawal.ProvenDisputeGameIndex().Int64())
	asr := bindings.NewBindings[gameClaimValidity](
		bindings.WithClient(sys.L1EL.EthClient()), bindings.WithTo(delays.AnchorStateRegistry), bindings.WithTest(t))
	valid, err := contractio.Read(asr.IsGameClaimValid(game.Address), t.Ctx())
	t.Require().NoError(err, "failed to read isGameClaimValid")
	t.Require().True(valid, "the proven game's claim is not valid")
	t.Require().ErrorContains(errutil.TryAddRevertReason(withdrawal.CheckFinalizable(user)), proofNotOldEnough,
		"the proof should be too young before the upgrade")
	proven := provenWithdrawal(t, sys, delays.OptimismPortal, withdrawal, user)

	upgradeMidTest(t, sys)

	// The upgrade keeps the proof as it was, so its maturity still counts from the original proof.
	t.Require().Equal(proven, provenWithdrawal(t, sys, delays.OptimismPortal, withdrawal, user), "the proven withdrawal changed across the OPCM upgrade")
	t.Require().ErrorContains(errutil.TryAddRevertReason(withdrawal.CheckFinalizable(user)), proofNotOldEnough,
		"the proof should still be too young after the upgrade")

	// Past the original proof's maturity, the withdrawal finalizes without a new proof.
	sys.AdvanceTime(proofMaturityDelaySeconds * time.Second)
	t.Require().NoError(withdrawal.CheckFinalizable(user), "withdrawal is not finalizable after its original proof maturity")
	withdrawal.Finalize(user)
}
