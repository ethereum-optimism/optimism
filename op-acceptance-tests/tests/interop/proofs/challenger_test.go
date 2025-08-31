package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestChallengerPlaysGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(15))
	dgf := sys.DisputeGameFactory()

	game := dgf.StartSuperCannonGame(attacker, proofs.WithRootClaim(badClaim))

	claim := game.RootClaim()                   // This is the bad claim from attacker
	counterClaim := claim.WaitForCounterClaim() // This is the counter-claim from the challenger
	for counterClaim.Depth() <= game.SplitDepth() {
		claim = counterClaim.Attack(attacker, badClaim)
		// Wait for the challenger to counter the attacker's claim, then attack again
		counterClaim = claim.WaitForCounterClaim()
	}
}

func TestChallengerRespondsToMultipleInvalidClaims(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	attacker := sys.FunderL1.NewFundedEOA(eth.TenEther)
	dgf := sys.DisputeGameFactory()

	game := dgf.StartSuperCannonGame(attacker)
	claims := game.PerformMoves(attacker,
		proofs.Move(0, common.Hash{0x01}, true),
		proofs.Move(1, common.Hash{0x03}, true),
		proofs.Move(1, common.Hash{0x02}, false), // Defends invalid claim so won't be countered.
	)

	claims[0].WaitForCounterClaim(claims...)
	claims[1].WaitForCounterClaim(claims...)
	claims[2].VerifyNoCounterClaim()
}

func TestChallengerRespondsToMultipleInvalidClaimsEOA(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	dgf := sys.DisputeGameFactory()
	attacker := dgf.CreateHelperEOA(sys.FunderL1.NewFundedEOA(eth.TenEther))

	game := dgf.StartSuperCannonGame(attacker.EOA)
	claims := attacker.PerformMoves(game.FaultDisputeGame,
		proofs.Move(0, common.Hash{0x01}, true),
		proofs.Move(1, common.Hash{0x03}, true),
		proofs.Move(1, common.Hash{0x02}, false), // Defends invalid claim so won't be countered.
	)

	claims[0].WaitForCounterClaim(claims...)
	claims[1].WaitForCounterClaim(claims...)
	claims[2].VerifyNoCounterClaim()
	for _, claim := range claims {
		require.Equal(t, attacker.Address(), claim.Claimant())
	}
}
