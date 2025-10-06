package sync_tester_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Ensures a user L2 transaction appears in blocks and both CLs (one on SyncTester EL) sync past it
func TestSyncTester_UserTxIncludedAndSynced(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()

	sys := presets.NewSimpleWithSyncTester(t)

	// Fund two EOAs on L2
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bobInitial := bob.GetBalance()

	// Send a user transfer on L2 via sequencer path
	amount := eth.OneHundredthEther
	tx := alice.Transfer(bob.Address(), amount)
	receipt, err := tx.Included.Eval(t.Ctx())
	require.NoError(err)
	require.NotNil(receipt)

	// Let both CLs advance beyond the inclusion block
	inclusionNum := receipt.BlockNumber.Uint64()
	target := inclusionNum + 5

	sys.L2CL.Reached(types.LocalUnsafe, target, 120)
	sys.L2CL2.Reached(types.LocalUnsafe, target, 120)

	// Verify receiver balance increased by at least the transfer amount
	bob.WaitForBalance(bobInitial.Add(amount))
}
