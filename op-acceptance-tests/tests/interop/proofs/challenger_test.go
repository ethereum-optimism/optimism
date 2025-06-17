package proofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestChallengerPlaysGame(gt *testing.T) {
	// Setup
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	sys.L1Network.WaitForOnline()

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(2))
	helper := proofs.DisputeGameFactoryForNetwork(t, sys.L1Network, sys.L1EL.EthClient(), sys.L2ChainA.DisputeGameFactoryProxyAddr(), sys.Supervisor)

	game := helper.StartSuperCannonGame(attacker, badClaim)

	// Wait for the challenger to counter the bad root claim
	claim := game.GetRootClaim()
	claim.WaitForCounterClaim()
}
