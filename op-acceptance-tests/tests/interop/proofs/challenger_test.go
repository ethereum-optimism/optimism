package proofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestChallengerPlaysGame(gt *testing.T) {
	// Setup
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	sys.L1Network.WaitForOnline()

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := fundAttackerWallet(t, sys, eth.OneEther.Mul(2))
	helper := proofs.HelperFromInteropPreset(t, sys, sys.L2ChainA)

	game := helper.StartSuperCannonGame(attacker, badClaim)

	claim := game.GetRootClaim()                // This is the bad claim from attacker
	counterClaim := claim.WaitForCounterClaim() // This is the counter-claim from the challenger
	for counterClaim.GetDepth() < game.GetSplitDepth() {
		claim = counterClaim.Attack(attacker, badClaim)
		counterClaim = claim.WaitForCounterClaim()
	}
}

func fundAttackerWallet(t devtest.T, sys *presets.SimpleInterop, fundingAmount eth.ETH) *dsl.EOA {
	wallet := sys.Wallet.NewEOA(sys.L1EL)
	initialBalance := sys.FunderL1.FundAtLeast(wallet, fundingAmount)
	require.GreaterOrEqual(t, initialBalance.ToBig().Int64(), fundingAmount.ToBig().Int64())

	return wallet
}
