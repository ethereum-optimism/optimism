package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
)

func TestWithdrawalRoot(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	bridge := sys.StandardBridge()
	initialL1Balance := eth.OneThirdEther
	initialL2Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL) // Only receives funds via the deposit
	sys.FunderL2.FundAtLeast(l2User, initialL2Balance)
	l2Client := sys.L2EL.Escape().EthClient()
	withdrawalAmount := eth.OneHundredthEther

	preBlock, err := l2Client.BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(err)
	preBlockHash := preBlock.Hash

	// Get pre-withdrawal storage root of L2ToL1MessagePasser
	preProof, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, nil, preBlockHash.String())
	require.NoError(err)
	preWithdrawalsRoot := preProof.StorageHash

	t.Logger().Info("Pre-withdrawal storage root", "root", preWithdrawalsRoot)
	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance := initialL2Balance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	// Get post-withdrawal state
	postBlockHash := withdrawal.WithdrawalRoot()
	postProof, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, nil, postBlockHash.String())
	require.NoError(err)
	postWithdrawalsRoot := postProof.StorageHash

	t.Logger().Info("Post-withdrawal storage root", "root", postWithdrawalsRoot)

	// Verify that the withdrawal root changed
	require.NotEqual(t, preWithdrawalsRoot, postWithdrawalsRoot, "withdrawal storage root should change after withdrawal initiation")
}
