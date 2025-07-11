package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestChallengerPlaysGame(gt *testing.T) {
	// Setup
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(2))
	dgf := sys.DisputeGameFactory()

	game := dgf.StartSuperCannonGame(attacker, badClaim)

	// Wait for the challenger to counter the bad root claim
	claim := game.RootClaim()
	claim.WaitForCounterClaim()
}
