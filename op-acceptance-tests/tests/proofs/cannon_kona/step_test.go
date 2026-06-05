package cannon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestExecuteStep_CannonKona(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|10:33:38.442]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/dsl/proofs/game_helper.go:263
	// assertions.go:387:             	            				/optimism/op-devstack/dsl/proofs/game_helper.go:196
	// assertions.go:387:             	            				/optimism/op-devstack/dsl/proofs/fault_dispute_game.go:124
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/proofs/cannon/step_test.go:40
	// assertions.go:387:             	            				/.local/share/mise/installs/go/1.24.13/src/runtime/asm_arm64.s:1223
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	failed to get safe head at L1 block 0xe926b953777546729b2607011590354a6a42dc35e36b7ea68c9dd4b470261541:6: not found
	// assertions.go:387:             	Test:       	TestExecuteStep_CannonKona
	// assertions.go:387:             	Messages:   	Failed to get honest root claim
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSystem(t)

	l1User := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
	blockNum := uint64(3)
	sys.L2CL.Reached(safety.LocalSafe, blockNum, 30)

	game := sys.DisputeGameFactory().StartCannonKonaGame(l1User, proofs.WithL2SequenceNumber(blockNum))
	claim := game.DisputeL2SequenceNumber(l1User, game.RootClaim(), blockNum)
	game.LogGameData()
	claim = claim.WaitForCounterClaim()             // Wait for the honest challenger to counter
	claim = game.DisputeToStep(l1User, claim, 1000) // Skip down to max depth
	game.LogGameData()
	claim.WaitForCountered()
}
