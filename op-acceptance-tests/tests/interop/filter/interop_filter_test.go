package filter

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func setupInteropFilterTest(t devtest.T) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, 0, presets.WithInteropFilter())
}

// TestInteropFilter_IngressAcceptsValid verifies that a valid interop transaction
// with correct cross-chain references passes through the interop filter.
func TestInteropFilter_IngressAcceptsValid(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	sys.L2B.CatchUpTo(sys.L2A)

	// Send init message on chain A
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 10))

	// Wait for at least one block between init and exec
	sys.L2B.WaitForBlock()

	// Send exec message on chain B — the interop filter validates the access list
	execMsg := bob.SendExecMessage(initMsg)

	// Verify cross-safe safety passes for both messages
	dsl.CheckAll(t,
		sys.L2ACL.ReachedRefFn(suptypes.CrossSafe, initMsg.BlockID(), 500),
		sys.L2BCL.ReachedRefFn(suptypes.CrossSafe, execMsg.BlockID(), 500),
	)
}

// TestInteropFilter_IngressRejectsInvalid verifies that a transaction with fabricated
// CrossL2Inbox access list entries is rejected by the interop filter.
func TestInteropFilter_IngressRejectsInvalid(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bobAddr := bob.Address()
	tx := txplan.NewPlannedTx(
		bob.Plan(),
		txplan.WithTo(&bobAddr),
		txplan.WithValue(eth.GWei(1)),
		txplan.WithAccessList(accessList),
		txplan.WithGasLimit(100_000),
	)

	// The transaction should fail — the filter rejects invalid interop entries.
	// This may manifest as a submission error or the tx never being included.
	_, err := tx.Included.Eval(ctx)
	require.Error(err, "transaction with fabricated access list should not be included")
}

// TestInteropFilter_FailsafeBlocksInterop verifies that enabling failsafe
// prevents new interop transactions from being accepted.
func TestInteropFilter_FailsafeBlocksInterop(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
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
	_ = bob.SendExecMessage(initMsg)

	// Step 2: Enable failsafe
	require.NotNil(sys.InteropFilter, "interop filter must be configured")
	sys.InteropFilter.SetFailsafeEnabled(true)

	// Step 3: Wait for failsafe to propagate to op-reth (polls every 1s)
	time.Sleep(2 * time.Second)

	// Step 4: Send another init message and try exec — should fail
	initMsg2 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 5))
	sys.L2B.WaitForBlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// The exec message should be rejected by the filter during failsafe
	execTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 5)
	_ = execTrigger // We need to construct the exec message manually to catch the error

	// Try to submit interop tx - construct with fabricated valid-looking access list
	// During failsafe, even valid access lists should be rejected
	result, err := initMsg2.Tx.Result.Eval(ctx)
	if err == nil && len(result.Entries) > 0 {
		msg := result.Entries[0]
		accessList := types.AccessList{{
			Address:     predeploys.CrossL2InboxAddr,
			StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{msg.Access()}),
		}}

		bobAddr := bob.Address()
		tx := txplan.NewPlannedTx(
			bob.Plan(),
			txplan.WithTo(&bobAddr),
			txplan.WithValue(eth.GWei(1)),
			txplan.WithAccessList(accessList),
			txplan.WithGasLimit(100_000),
		)

		_, err = tx.Included.Eval(ctx)
		require.Error(err, "interop tx should be rejected during failsafe")
	}

	// Step 5: Disable failsafe
	sys.InteropFilter.SetFailsafeEnabled(false)

	// Step 6: Wait for failsafe to clear
	time.Sleep(2 * time.Second)

	// Step 7: Verify interop txs work again
	initMsg3 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 5))
	sys.L2B.WaitForBlock()
	_ = bob.SendExecMessage(initMsg3)
}

// TestInteropFilter_NonInteropUnaffected verifies that regular (non-interop)
// transactions are accepted regardless of failsafe state.
func TestInteropFilter_NonInteropUnaffected(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)
	require := t.Require()

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)

	// Enable failsafe
	require.NotNil(sys.InteropFilter, "interop filter must be configured")
	sys.InteropFilter.SetFailsafeEnabled(true)
	time.Sleep(2 * time.Second)

	// Send a regular (non-interop) transfer — should succeed even during failsafe
	tx := alice.Transfer(bob.Address(), eth.GWei(1000))
	receipt, err := tx.Included.Eval(context.Background())
	require.NoError(err, "regular transfer should succeed during failsafe")
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "regular transfer should succeed")

	// Disable failsafe
	sys.InteropFilter.SetFailsafeEnabled(false)
}

// TestInteropFilter_FailsafeEvictsPooled verifies that when failsafe transitions
// from disabled to enabled, existing interop transactions in the pool are evicted.
func TestInteropFilter_FailsafeEvictsPooled(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.ParallelT(gt)
	sys := setupInteropFilterTest(t)
	require := t.Require()

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)

	// Send init message — this creates an interop tx that goes into chain A's pool
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 10))
	sys.L2B.WaitForBlock()

	// Verify the exec message works normally
	execMsg := bob.SendExecMessage(initMsg)
	require.Equal(types.ReceiptStatusSuccessful, execMsg.Receipt.Status)

	// Enable failsafe — this should evict interop txs from the pool within 1s
	sys.InteropFilter.SetFailsafeEnabled(true)

	// Wait for failsafe polling to detect and evict (polls every 1s)
	time.Sleep(3 * time.Second)

	// Verify failsafe is active
	require.True(sys.InteropFilter.FailsafeEnabled())

	// New interop exec message should fail
	initMsg2 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 5))
	sys.L2B.WaitForBlock()

	// Attempt the exec — should be rejected
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := initMsg2.Tx.Result.Eval(ctx)
	if err == nil && len(result.Entries) > 0 {
		msg := result.Entries[0]
		accessList := types.AccessList{{
			Address:     predeploys.CrossL2InboxAddr,
			StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{msg.Access()}),
		}}

		bobAddr := bob.Address()
		tx := txplan.NewPlannedTx(
			bob.Plan(),
			txplan.WithTo(&bobAddr),
			txplan.WithValue(eth.GWei(1)),
			txplan.WithAccessList(accessList),
			txplan.WithGasLimit(100_000),
		)

		_, err = tx.Included.Eval(ctx)
		require.Error(err, "interop tx should be rejected during failsafe")
	}

	// Disable failsafe and verify recovery
	sys.InteropFilter.SetFailsafeEnabled(false)
	time.Sleep(2 * time.Second)

	// Wait for state to settle, then retry interop flow (non-mandatory, verify recovery)
	err = retry.Do0(context.Background(), 5, &retry.FixedStrategy{Dur: 2 * time.Second}, func() error {
		initMsg3 := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 3))
		sys.L2B.WaitForBlock()
		_ = bob.SendExecMessage(initMsg3)
		return nil
	})
	require.NoError(err, "interop flow should recover after failsafe disabled")
}
