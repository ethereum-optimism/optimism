//go:build !ci

package upgrade

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// preInteropExecError returns true if err matches a known pre-interop
// executing message failure from op-geth or op-reth.
func preInteropExecError(err error) bool {
	msg := err.Error()
	// op-geth: gas estimation hits the uninitialized proxy and returns the revert reason
	if strings.Contains(msg, "implementation not initialized") {
		return true
	}
	// op-reth: gas estimation fails with a generic gas error for the same underlying reason
	if strings.Contains(msg, "intrinsic gas too high") {
		return true
	}
	return false
}

// TestPreNoInbox verifies pre-interop behavior: the CrossL2Inbox is not deployed,
// the derivation pipeline advances local-safe heads, and executing messages fail
// before the interop fork activates.
func TestPreNoInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Use a very large activation delay (24h) so interop never activates during the test.
	// This test only checks pre-interop state and the old 60s offset caused flakiness
	// when the test took too long and crossed the activation boundary.
	// See: https://github.com/ethereum-optimism/optimism/issues/17298
	sys := presets.NewTwoL2SupernodeInterop(t, 86400)
	require := t.Require()

	t.Logger().Info("Starting")

	// Phase 1: Verify CrossL2Inbox is NOT deployed before interop activation
	devtest.RunParallel(t, []*dsl.L2Network{sys.L2A, sys.L2B}, func(t devtest.T, net *dsl.L2Network) {
		interopTime := net.Escape().ChainConfig().InteropTime
		t.Require().NotNil(interopTime)
		pre := net.LatestBlockBeforeTimestamp(t, *interopTime)
		el := net.PrimaryEL()
		codeAddr := common.HexToAddress("0xC0D3C0d3C0D3C0d3c0d3c0D3c0D3C0d3C0D30022")
		implCode, err := el.EthClient().CodeAtHash(t.Ctx(), codeAddr, pre.Hash)
		require.NoError(err)
		require.Len(implCode, 0, "needs to be empty")
		implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, pre.Hash.String())
		require.NoError(err)
		require.Equal(common.Address{}, common.BytesToAddress(implAddrBytes[:]))
	})

	// Phase 2: Verify the derivation pipeline works pre-interop by checking
	// that both chains advance their local-safe heads (batcher submits to L1,
	// supernode derives from it).
	//
	// TODO(#20191): also assert CrossSafe and Finalized advance pre-interop.
	// Currently the supernode stalls both heads at block 0 until interop
	// activates, which would cause a chain-visible stall for any network
	// with interop scheduled in the future. Once fixed, enable:
	//
	//   sys.L2ACL.AdvancedFn(types.CrossSafe, 5, 100),
	//   sys.L2BCL.AdvancedFn(types.CrossSafe, 5, 100),
	//   sys.L2ACL.AdvancedFn(types.Finalized, 1, 100),
	//   sys.L2BCL.AdvancedFn(types.Finalized, 1, 100),
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalSafe, 5, 100),
		sys.L2BCL.AdvancedFn(types.LocalSafe, 5, 100),
	)

	// Phase 3: Try interop before the upgrade, confirm that messages do not get included
	{
		alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
		bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

		interopTimeA := sys.L2A.Escape().ChainConfig().InteropTime
		interopTimeB := sys.L2B.Escape().ChainConfig().InteropTime

		eventLoggerAddress := alice.DeployEventLogger()

		sys.L2B.CatchUpTo(sys.L2A)

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)))

		sys.L2B.WaitForBlock()

		// Send executing message on chain B — should fail because CrossL2Inbox is not initialized.
		execTx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
		execTx.Content.DependOn(&initMsg.Tx.Result)
		execTx.Content.Fn(txintent.ExecuteIndexed(predeploys.CrossL2InboxAddr, &initMsg.Tx.Result, 0))
		execReceipt, err := execTx.PlannedTx.Included.Eval(t.Ctx())
		require.Error(err, "executing message should fail before interop activation")
		require.True(preInteropExecError(err),
			"expected pre-interop exec rejection, got: %v", err)
		require.Nil(execReceipt)

		t.Logger().Info("initReceipt", "msg", initMsg)

		// Confirm we are still pre-interop
		require.False(sys.L2A.IsActivated(*interopTimeA))
		require.False(sys.L2B.IsActivated(*interopTimeB))
		t.Logger().Info("Timestamps", "interopTimeA", *interopTimeA, "interopTimeB", *interopTimeB, "now", time.Now().Unix())
	}

	t.Logger().Info("Done")
}
