package jovian

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// This test will fail if there are any user transactions in the L@ block.
// The tests spawns no user transactions itself,
// so the existence of such transactions indicates a test isolation bug.
func TestForNoUserTransactions(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	t.Require().NoError(err, "Jovian fork must be active for this test")

	for range 10 {
		block := sys.L2EL.WaitForBlock()
		_, txs, err := sys.L2EL.Escape().EthClient().InfoAndTxsByHash(t.Ctx(), block.Hash)
		t.Require().NoError(err)
		t.Require().Len(txs, 1, "Expected no user transactions in block")
	}

}
