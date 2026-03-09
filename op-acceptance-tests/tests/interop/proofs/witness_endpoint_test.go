package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

// TestWitnessEndpointEnabled verifies that the challenger engages correctly in a dispute game
// when the experimental witness endpoint flag is enabled. The witness endpoint is enabled for
// all tests in this package via the l2_challenger.go configuration.
func TestWitnessEndpointEnabled(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(15))
	dgf := sys.DisputeGameFactory()
	game := dgf.StartSuperCannonKonaGame(attacker, proofs.WithSuperRootFrom(eth.Bytes32(badClaim), eth.Bytes32(badClaim)))

	// Challenger should counter the invalid claim. This exercises the witness endpoint
	// code path: the executor appends --enable-experimental-witness-endpoint to kona
	// subprocess args, and kona uses debug_executePayload for witness collection.
	claim := game.RootClaim()
	counterClaim := claim.WaitForCounterClaim()
	for counterClaim.Depth() <= game.SplitDepth() {
		claim = counterClaim.Attack(attacker, badClaim)
		counterClaim = claim.WaitForCounterClaim()
	}
}
