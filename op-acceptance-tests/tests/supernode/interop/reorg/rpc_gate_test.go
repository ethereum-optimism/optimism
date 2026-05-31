package reorg

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/poller"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// reorgWaitAttempts bounds how long we wait for the invalid block to be reorged
// out. ReorgTriggered retries every 2s, so 30 attempts is a ~60s window.
const reorgWaitAttempts = 30

// TestSupernodeInteropRPCGatedDuringReorg proves patient callers do not see
// method-not-found or route-missing responses while supernode restarts chain B's
// virtual node during interop reorg recovery.
func TestSupernodeInteropRPCGatedDuringReorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()

	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	paused := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", paused)

	rng := rand.New(rand.NewSource(12345))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)

	t.Logger().Info("initiating message sent on chain A",
		"block", initMsg.BlockNumber(),
		"hash", initMsg.BlockHash(),
	)

	sys.L2B.WaitForBlock()

	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)
	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber,
		"hash", invalidBlockHash,
		"timestamp", invalidBlockTimestamp,
	)

	require.Eventually(t, func() bool {
		return sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	// Capture the invalid block before reorg so we can wait for it to be replaced.
	invalidRef := sys.L2ELB.BlockRefByNumber(invalidBlockNumber)

	// Continuously poll the supernode's chain-B RPC route across the reorg. The
	// poller stops itself via t.Cleanup at the end of the test.
	statusPoller := poller.StartStatusPoller(sys.L2BCL)
	statusPoller.WaitForNextSuccess()

	sys.Supernode.ResumeInterop()

	// Recovery reorgs the invalid block out and replaces it with a valid one.
	sys.L2ELB.ReorgTriggered(invalidRef, reorgWaitAttempts)

	sys.Supernode.AwaitValidatedTimestamp(invalidBlockTimestamp)
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)

	bruce := sys.FunderB.NewFundedEOA(eth.OneEther)
	tx := bruce.Transfer(alice.Address(), eth.OneHundredthEther)
	txBlock := bigs.Uint64Strict(tx.Included.Value().BlockNumber)
	sys.L2ELB.AssertTxInBlock(txBlock, tx.Included.Value().TxHash)

	txTimestamp := sys.L2B.TimestampForBlockNum(txBlock)
	sys.Supernode.AwaitValidatedTimestamp(txTimestamp)
	sys.L2ELB.AssertTxInBlock(txBlock, tx.Included.Value().TxHash)

	// The route must keep serving across the swap, never returning method-not-found
	// or a route-missing 404.
	statusPoller.WaitForNextSuccess()
	statusPoller.RequireNoRouteErrors()
}
