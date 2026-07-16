package super

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSuperPermissionedWithdrawal(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInteropSuperRootAtGenesis(t, presets.WithTimeTravelEnabled())
	sys.L1Network.WaitForOnline()

	bridge := sys.StandardBridge(sys.L2ChainA)
	bridge.VerifyRespectedGameType(gameTypes.SuperPermissionedGameType)

	initialL1Balance := eth.HalfEther
	depositAmount := eth.OneThirdEther
	withdrawalAmount := eth.OneTenthEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2ELA)

	deposit := bridge.Deposit(depositAmount, l1User)
	l1User.VerifyBalanceExact(initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost()))
	l2User.VerifyBalanceExact(depositAmount)

	sys.L2ChainA.WaitForBlock()

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	game := sys.DisputeGameFactory().WaitForGame()
	t.Require().Equal(gameTypes.SuperPermissionedGameType, game.GameType())
	minimumAnchorSequence := game.L2SequenceNumber()

	withdrawal.Prove(l1User)
	l2User.VerifyBalanceExact(depositAmount.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost()))

	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))
	withdrawal.Finalize(l1User)

	l1User.VerifyBalanceExact(initialL1Balance.
		Sub(depositAmount).
		Sub(deposit.GasCost()).
		Sub(withdrawal.ProveGasCost()).
		Sub(withdrawal.FinalizeGasCost()).
		Add(withdrawalAmount))
	sys.AnchorStateRegistry().WaitForAnchorGame(gameTypes.SuperPermissionedGameType, minimumAnchorSequence)
}
