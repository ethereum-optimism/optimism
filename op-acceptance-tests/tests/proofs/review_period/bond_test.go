package review_period

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// wethWithdrawalTopic is the topic of WETH98's Withdrawal(address,uint256) event, emitted when DelayedWETH pays out.
var wethWithdrawalTopic = crypto.Keccak256Hash([]byte("Withdrawal(address,uint256)"))

// gameBond binds a fault dispute game's bond claiming.
type gameBond struct {
	Credit      func(recipient common.Address) bindings.TypedCall[*big.Int] `sol:"credit"`
	ClaimCredit func(recipient common.Address) bindings.TypedCall[any]      `sol:"claimCredit"`
}

// TestBondUnlockedBeforeOPCMUpgradeKeepsItsWithdrawalDelay checks that a game bond unlocked in DelayedWETH
// before an OPCM upgrade keeps its amount and timestamp, can't be withdrawn before the original delay, and
// pays out the expected amount after it.
func TestBondUnlockedBeforeOPCMUpgradeKeepsItsWithdrawalDelay(gt *testing.T) {
	t := devtest.ParallelT(gt)
	gameType := gameTypes.PermissionedGameType
	sys := presets.NewMinimal(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(
			sysgo.WithWithdrawalDelaySeconds(withdrawalDelaySeconds),
			// Short game clocks and finality delay, so the bond can be unlocked quickly.
			sysgo.WithDisputeGameFinalityDelaySeconds(4),
			sysgo.WithFaultGameMaxClockDuration(40),
			sysgo.WithFaultGameClockExtension(1),
			sysgo.WithPreimageOracleChallengePeriod(10),
		),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
		presets.WithGameTypeAdded(gameType),
	)
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "failed to derive the dev keys")
	proposer, err := keys.Address(devkeys.ChainOperatorKeys(sys.L2Chain.ChainID().ToBig())(devkeys.ProposerRole))
	t.Require().NoError(err, "failed to derive the proposer address")
	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")

	// An unchallenged game resolves for the proposer, and once it is finalized the proposer's bond can be claimed.
	game := sys.DisputeGameFactory().WaitForGame()
	t.Require().Equal(delays.DelayedWETH, game.WETHAddress(), "the game uses a different DelayedWETH than the SystemConfig")
	sys.AdvanceTime(game.MaxClockDuration())
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	sys.AdvanceTime(time.Duration(delays.DisputeGameFinalityDelaySeconds+1) * time.Second)

	bond := bindings.NewBindings[gameBond](
		bindings.WithClient(sys.L1EL.EthClient()), bindings.WithTo(game.Address), bindings.WithTest(t))
	credit, err := contractio.Read(bond.Credit(proposer), t.Ctx())
	t.Require().NoError(err, "failed to read the proposer's credit")
	t.Require().Positive(credit.Sign(), "the proposer has no credit in the game")
	claimer := sys.FunderL1.NewFundedEOA(eth.OneTenthEther)
	claimCredit := func() *types.Receipt {
		var receipt *types.Receipt
		// Retry, as the head block timestamp needs to catch up after time travel for estimateGas to pass.
		t.Require().Eventually(func() bool {
			receipt, err = contractio.Write(bond.ClaimCredit(proposer), t.Ctx(), claimer.Plan())
			return err == nil && receipt.Status == types.ReceiptStatusSuccessful
		}, 60*time.Second, 100*time.Millisecond, "claimCredit failed")
		return receipt
	}

	// The first claimCredit closes the game and unlocks the proposer's bond in DelayedWETH.
	claimCredit()
	weth := sys.DisputeGameFactory().DelayedWETH(game.WETHAddress())
	unlocked := weth.Withdrawal(game.Address, proposer)
	t.Require().Equal(credit, unlocked.Amount, "DelayedWETH unlocked a different amount than the proposer's credit")

	sysgo.UpgradeOPChain(t, keys, sys.L1Network.ChainID(), sys.L1EL.Escape().UserRPC(),
		sys.L2Chain.ChainID(), systemConfig, gameType)

	pre, post := readDelaysAcrossLastUpgrade(t, sys.L1EL, systemConfig)
	for _, d := range reviewperiod.CompareDelays(pre, post) {
		t.Require().False(d.Changed(), "%s changed across the OPCM upgrade: %d -> %d", d.Name, d.Before, d.After)
	}
	afterUpgrade := weth.Withdrawal(game.Address, proposer)
	t.Require().Equal(unlocked.Amount, afterUpgrade.Amount, "the unlocked amount changed across the OPCM upgrade")
	t.Require().Equal(unlocked.Timestamp, afterUpgrade.Timestamp, "the unlock timestamp changed across the OPCM upgrade")

	// The second claimCredit withdraws from DelayedWETH, which still enforces the original delay.
	_, err = contractio.Read(bond.ClaimCredit(proposer), t.Ctx())
	t.Require().ErrorContains(errutil.TryAddRevertReason(err), "DelayedWETH: withdrawal delay not met",
		"the bond should still be inside its withdrawal delay")

	// Past the original delay, the bond pays out the full credit.
	sys.AdvanceTime(withdrawalDelaySeconds * time.Second)
	receipt := claimCredit()
	var paid *big.Int
	for _, l := range receipt.Logs {
		if l.Address == game.WETHAddress() && len(l.Topics) == 2 && l.Topics[0] == wethWithdrawalTopic {
			t.Require().Equal(game.Address, common.BytesToAddress(l.Topics[1].Bytes()), "DelayedWETH paid out to another account")
			paid = new(big.Int).SetBytes(l.Data)
		}
	}
	t.Require().NotNil(paid, "claimCredit did not withdraw from DelayedWETH")
	t.Require().Equal(credit, paid, "DelayedWETH paid out a different amount than the unlocked bond")
	t.Require().Zero(weth.Withdrawal(game.Address, proposer).Amount.Sign(), "the withdrawal request was not drained")
}
