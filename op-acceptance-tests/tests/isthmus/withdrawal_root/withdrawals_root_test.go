package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
)

func TestWithdrawalRoot(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for this test")

	secondCheck, err := dsl.CheckForChainFork(t.Ctx(), sys.L2Networks(), t.Logger())
	require.NoError(err, "error checking for chain fork")
	defer func() {
		require.NoError(secondCheck(false), "error checking for chain fork")
	}()

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

	// Get the full block info to access WithdrawalsRoot
	preBlockInfo, err := l2Client.InfoByHash(t.Ctx(), preBlockHash)
	require.NoError(err)

	// Get pre-withdrawal storage root of L2ToL1MessagePasser
	preProof, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, nil, preBlockHash.String())
	require.NoError(err)
	preWithdrawalsRoot := preProof.StorageHash.String()
	preBlockWithdrawalsRoot := preBlockInfo.WithdrawalsRoot().String()

	t.Logger().Info("Pre-withdrawal storage root", "root", preWithdrawalsRoot)
	t.Logger().Info("Pre-withdrawal block header withdrawal root", "root", preBlockWithdrawalsRoot)

	// Verify that the pre-withdrawal storage root matches the withdrawal root in the block header
	// According to the isthmus spec, withdrawal roots in the header must be equal to L2ToL1MessagePasser account storage root
	require.Equal(preWithdrawalsRoot, preBlockWithdrawalsRoot, "pre-withdrawal storage root should match block header withdrawal root")

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance := initialL2Balance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	// Get post-withdrawal state
	postBlockHash := withdrawal.WithdrawalRoot()
	postProof, err := l2Client.GetProof(t.Ctx(), predeploys.L2ToL1MessagePasserAddr, nil, postBlockHash.String())
	require.NoError(err)
	postWithdrawalsRoot := postProof.StorageHash.String()

	// Get the full block info to access WithdrawalsRoot for the post-withdrawal block
	postBlockInfo, err := l2Client.InfoByHash(t.Ctx(), postBlockHash)
	require.NoError(err)
	postBlockWithdrawalsRoot := postBlockInfo.WithdrawalsRoot().String()

	t.Logger().Info("Post-withdrawal storage root", "root", postWithdrawalsRoot)
	t.Logger().Info("Post-withdrawal block header withdrawal root", "root", postBlockWithdrawalsRoot)

	// Verify that the post-withdrawal storage root matches the withdrawal root in the block header
	// According to the isthmus spec, withdrawal roots in the header must be equal to L2ToL1MessagePasser account storage root
	require.Equal(postWithdrawalsRoot, postBlockInfo.WithdrawalsRoot().String(), "post-withdrawal storage root should match block header withdrawal root")

	// Verify that the withdrawal root changed
	require.NotEqual(preWithdrawalsRoot, postWithdrawalsRoot, "withdrawal storage root should change after withdrawal initiation")
}
