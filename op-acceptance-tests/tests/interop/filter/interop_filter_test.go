package filter

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func setupInteropFilterTest(t devtest.T) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, 0, presets.WithInteropFilter())
}

// interopTxRejectedError returns true if err matches any known interop
// transaction rejection from op-geth, op-reth, or the interop filter.
func interopTxRejectedError(err error) bool {
	msg := err.Error()
	// op-geth: generic filter rejection wrapping all causes
	if strings.Contains(msg, "transaction filtered out") {
		return true
	}
	// op-interop-filter: malformed or unrecognized access list entry
	if strings.Contains(msg, "failed to parse access entry") {
		return true
	}
	// op-reth fast-path: cached failsafe state rejects before calling filter
	if strings.Contains(msg, "interop failsafe is active") {
		return true
	}
	// op-interop-filter: failsafe enabled at the filter level
	if strings.Contains(msg, "failsafe is enabled") {
		return true
	}
	return false
}

// TestInteropFilter_IngressRejectsInvalid verifies that a transaction with fabricated
// CrossL2Inbox access list entries is rejected by the interop filter.
func TestInteropFilter_IngressRejectsInvalid(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)
	require := t.Require()

	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	// Construct a fabricated access list entry with a random storage key
	// that the filter won't recognize as a valid cross-chain message
	fakeStorageKey := crypto.Keccak256Hash([]byte("fabricated-inbox-entry"))
	accessList := types.AccessList{{
		Address:     predeploys.CrossL2InboxAddr,
		StorageKeys: []common.Hash{fakeStorageKey},
	}}

	// Send a transaction with the fabricated access list.
	// The interop filter should reject this because the inbox entry doesn't
	// correspond to any real cross-chain message.
	ctx, cancel := context.WithTimeout(gt.Context(), 10*time.Second)
	defer cancel()

	bobAddr := bob.Address()
	elClient := sys.L2ELB.EthClient()
	tx := txplan.NewPlannedTx(
		bob.Plan(),
		// Override retry submission with single-attempt submitter so the
		// filter's rejection propagates immediately instead of retrying
		// until the context expires.
		txplan.WithTransactionSubmitter(elClient),
		txplan.WithTo(&bobAddr),
		txplan.WithValue(eth.GWei(1)),
		txplan.WithAccessList(accessList),
		txplan.WithGasLimit(100_000),
	)

	// The transaction should be explicitly rejected by the interop filter.
	_, err := tx.Submitted.Eval(ctx)
	require.Error(err, "transaction with fabricated access list should not be included")
	require.True(interopTxRejectedError(err),
		"expected interop filter rejection, got: %v", err)
}

// TestInteropFilter_FailsafeLifecycle verifies the full failsafe lifecycle:
// interop txs succeed normally, are blocked when failsafe is enabled,
// and recover after failsafe is disabled.
//
// NOT TESTED: txpool eviction of pending interop txs on failsafe activation.
// Both op-reth (poll_failsafe) and op-geth (startBackgroundInteropFailsafeDetection)
// have background tasks that evict interop txs from the pool when failsafe
// transitions from disabled to enabled. Testing this reliably is difficult because
// it requires stopping the sequencer to prevent block inclusion, but StopSequencer
// only stops the op-node from requesting new payloads — an already in-flight
// engine API payload (forkchoiceUpdated → getPayload → newPayload) can still land
// after StopSequencer returns. This creates a race where the tx may be included in
// a block before failsafe eviction runs, causing flaky assertions. Reliably testing
// eviction would require either a deterministic block builder that bypasses the
// mempool (which defeats the purpose) or an engine API hook to block payload
// completion (which doesn't exist in the test framework).
func TestInteropFilter_FailsafeLifecycle(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)
	require := t.Require()

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)

	// Step 1: Send a valid interop tx — should succeed before failsafe
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 10))
	sys.L2B.WaitForBlock()
	execMsg := bob.SendExecMessage(initMsg)
	require.Equal(types.ReceiptStatusSuccessful, execMsg.Receipt.Status,
		"interop tx should succeed before failsafe")

	// Step 2: Enable failsafe — no wait needed; even if op-reth hasn't polled
	// the change yet, checkAccessList will reject interop txs on the filter side.
	require.NotNil(sys.InteropFilter, "interop filter must be configured")
	sys.InteropFilter.SetFailsafeEnabled(true)

	// Step 3: Send another init message and try exec — should fail
	initMsg2 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 5))
	sys.L2B.WaitForBlock()

	ctx, cancel := context.WithTimeout(gt.Context(), 15*time.Second)
	defer cancel()

	// During failsafe, even valid access lists should be rejected
	result, err := initMsg2.Tx.Result.Eval(ctx)
	require.NoError(err, "init message result must be available")
	require.Greater(len(result.Entries), 0, "init message must have entries")

	msg := result.Entries[0]
	accessList := types.AccessList{{
		Address:     predeploys.CrossL2InboxAddr,
		StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{msg.Access()}),
	}}

	bobAddr := bob.Address()
	elClient := sys.L2ELB.EthClient()
	tx := txplan.NewPlannedTx(
		bob.Plan(),
		txplan.WithTransactionSubmitter(elClient),
		txplan.WithTo(&bobAddr),
		txplan.WithValue(eth.GWei(1)),
		txplan.WithAccessList(accessList),
		txplan.WithGasLimit(100_000),
	)

	_, err = tx.Submitted.Eval(ctx)
	require.Error(err, "interop tx should be rejected during failsafe")
	require.True(interopTxRejectedError(err),
		"expected interop filter rejection, got: %v", err)

	// Step 4: Disable failsafe and wait one block for op-reth's poller to pick up the change
	sys.InteropFilter.SetFailsafeEnabled(false)
	sys.L2B.WaitForBlock()

	// Step 5: Verify interop txs recover after failsafe is disabled
	initMsg3 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 3))
	sys.L2B.WaitForBlock()
	execMsg2 := bob.SendExecMessage(initMsg3)
	require.Equal(types.ReceiptStatusSuccessful, execMsg2.Receipt.Status,
		"interop tx should succeed after failsafe disabled")
}

// TestInteropFilter_NonInteropUnaffected verifies that regular (non-interop)
// transactions are accepted on both chains regardless of failsafe state.
func TestInteropFilter_NonInteropUnaffected(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)
	require := t.Require()

	aliceA := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bobA := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	aliceB := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	bobB := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	// Enable failsafe — takes effect immediately on the filter side
	require.NotNil(sys.InteropFilter, "interop filter must be configured")
	sys.InteropFilter.SetFailsafeEnabled(true)

	// Send regular (non-interop) transfers on both chains — should succeed even during failsafe
	txA := aliceA.Transfer(bobA.Address(), eth.GWei(1000))
	receiptA, err := txA.Included.Eval(gt.Context())
	require.NoError(err, "regular transfer on chain A should succeed during failsafe")
	require.Equal(types.ReceiptStatusSuccessful, receiptA.Status, "regular transfer on chain A should succeed")

	txB := aliceB.Transfer(bobB.Address(), eth.GWei(1000))
	receiptB, err := txB.Included.Eval(gt.Context())
	require.NoError(err, "regular transfer on chain B should succeed during failsafe")
	require.Equal(types.ReceiptStatusSuccessful, receiptB.Status, "regular transfer on chain B should succeed")

	// Disable failsafe
	sys.InteropFilter.SetFailsafeEnabled(false)
}
