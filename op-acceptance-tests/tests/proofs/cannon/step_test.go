package cannon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestExecuteStep_Cannon(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t,
		presets.WithProofs(),
		stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithJovianAtGenesis)),
		presets.WithSafeDBEnabled(),
		presets.WithCompatibleTypes(compat.SysGo),
	)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := uint64(3)
	sys.L2CL.Reached(types.LocalSafe, blockNum, 30)

	game := sys.DisputeGameFactory().StartCannonGame(l1User, proofs.WithL2SequenceNumber(blockNum))
	claim := game.DisputeL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	claim = claim.WaitForCounterClaim()             // Wait for the honest challenger to counter
	claim = game.DisputeToStep(l1User, claim, 1000) // Skip down to max depth
	game.LogGameData()
	claim.WaitForCountered()
}

func TestExecuteStep_CannonKona(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t,
		presets.WithProofs(),
		stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithJovianAtGenesis)),
		presets.WithSafeDBEnabled(),
		presets.WithCompatibleTypes(compat.SysGo),
	)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := uint64(3)
	sys.L2CL.Reached(types.LocalSafe, blockNum, 30)

	game := sys.DisputeGameFactory().StartCannonKonaGame(l1User, proofs.WithL2SequenceNumber(blockNum))
	claim := game.DisputeL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	claim = claim.WaitForCounterClaim()             // Wait for the honest challenger to counter
	claim = game.DisputeToStep(l1User, claim, 1000) // Skip down to max depth
	game.LogGameData()
	claim.WaitForCountered()
}
