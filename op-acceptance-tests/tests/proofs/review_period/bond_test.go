package review_period

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// wethWithdrawalTopic is the topic of WETH98's Withdrawal(address,uint256) event, emitted when DelayedWETH pays out.
var wethWithdrawalTopic = crypto.Keccak256Hash([]byte("Withdrawal(address,uint256)"))

// TestBondUnlockedBeforeOPCMUpgradeKeepsItsWithdrawalDelay checks that a game bond unlocked in DelayedWETH
// before an OPCM upgrade keeps its amount and timestamp, can't be withdrawn before the original delay, and
// pays out the expected amount after it.
func TestBondUnlockedBeforeOPCMUpgradeKeepsItsWithdrawalDelay(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newPermissionedMinimal(t, sysgo.WithWithdrawalDelaySeconds(withdrawalDelaySeconds))
	proposer := sys.L2Chain.Escape().Keys().Address(devkeys.ChainOperatorKeys(sys.L2Chain.ChainID().ToBig())(devkeys.ProposerRole))
	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")

	// An unchallenged game resolves for the proposer, and once it is finalized the proposer's bond can be claimed.
	// Nothing else claims it: the challenger only claims its own bonds, and the proposer doesn't claim at all.
	game := sys.DisputeGameFactory().WaitForGame()
	t.Require().Equal(delays.DelayedWETH, game.WETHAddress(), "the game uses a different DelayedWETH than the SystemConfig")
	sys.AdvanceTime(game.MaxClockDuration())
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	sys.AdvanceTime(time.Duration(delays.DisputeGameFinalityDelaySeconds+1) * time.Second)
	credit := game.Credit(proposer)
	t.Require().False(credit.IsZero(), "the proposer has no credit in the game")

	// claimCredit is permissionless, so any account can send it for the proposer.
	claimer := sys.FunderL1.NewFundedEOA(eth.OneTenthEther)

	// The first claimCredit closes the game and unlocks the proposer's bond in DelayedWETH.
	game.ClaimCredit(claimer, proposer)
	weth := sys.DisputeGameFactory().DelayedWETH(game.WETHAddress())
	unlocked := weth.Withdrawal(game.Address, proposer)
	t.Require().Equal(credit, eth.WeiBig(unlocked.Amount), "DelayedWETH unlocked a different amount than the proposer's credit")

	upgradeMidTest(t, sys)

	afterUpgrade := weth.Withdrawal(game.Address, proposer)
	t.Require().Zero(unlocked.Amount.Cmp(afterUpgrade.Amount), "the unlocked amount changed across the OPCM upgrade")
	t.Require().Zero(unlocked.Timestamp.Cmp(afterUpgrade.Timestamp), "the unlock timestamp changed across the OPCM upgrade")

	// The second claimCredit withdraws from DelayedWETH, which still enforces the original delay.
	t.Require().ErrorContains(errutil.TryAddRevertReason(game.CheckClaimCredit(proposer)), "DelayedWETH: withdrawal delay not met",
		"the bond should still be inside its withdrawal delay")

	// Past the original delay, the bond pays out the full credit. The proposer keeps spending gas on new
	// games, so the payout is read from DelayedWETH's event rather than from the proposer's balance.
	sys.AdvanceTime(withdrawalDelaySeconds * time.Second)
	receipt := game.ClaimCredit(claimer, proposer)
	var paid *big.Int
	for _, l := range receipt.Logs {
		if l.Address == game.WETHAddress() && len(l.Topics) == 2 && l.Topics[0] == wethWithdrawalTopic {
			t.Require().Equal(game.Address, common.BytesToAddress(l.Topics[1].Bytes()), "DelayedWETH paid out to another account")
			paid = new(big.Int).SetBytes(l.Data)
		}
	}
	t.Require().NotNil(paid, "claimCredit did not withdraw from DelayedWETH")
	t.Require().Equal(credit, eth.WeiBig(paid), "DelayedWETH paid out a different amount than the unlocked bond")
	t.Require().Zero(weth.Withdrawal(game.Address, proposer).Amount.Sign(), "the withdrawal request was not drained")
}
