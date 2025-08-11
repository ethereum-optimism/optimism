package proofs

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
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
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(15))
	dgf := sys.DisputeGameFactory()

	game := dgf.StartSuperCannonGame(attacker, badClaim)

	claim := game.RootClaim()                   // This is the bad claim from attacker
	counterClaim := claim.WaitForCounterClaim() // This is the counter-claim from the challenger
	for counterClaim.Depth() <= game.SplitDepth() {
		claim = counterClaim.Attack(attacker, badClaim)
		// Wait for the challenger to counter the attacker's claim, then attack again
		counterClaim = claim.WaitForCounterClaim()
	}
}

func TestChallengerRespondsToInvalidClaims(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(types.CrossSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.CrossSafe, 1, 30),
	)

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := sys.FunderL1.NewFundedEOA(eth.Ether(100))
	dgf := sys.DisputeGameFactory()

	gs := dgf.GameState(attacker)
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], 8249249824792999)

	game := dgf.StartSuperCannonGame(attacker, badClaim)
	game.PerformMoves(attacker,
		proofs.Move(0, badClaim, true),
		proofs.Move(1, badClaim, false),
	)

	game.LogGameData()

	//gs.CreateGameWithClaims(attacker, dgf, cTypes.SuperCannonGameType, badClaim, extraData, []proofs.GameStateMove{
	//	proofs.Move(0, badClaim, true),
	//	proofs.Move(1, badClaim, false),
	//})
}
