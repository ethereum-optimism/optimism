package opcon

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// The tests in this file exercise op-con-node's SEQUENCER mode: op-con-node
// produces the unsafe chain itself by driving its paired EL's build loop
// (noTxPool=false — the EL packs its own mempool behind the forced deposit
// prefix), while still deriving its safe chain from L1 like a verifier.
//
// Devstack routes sequencer slots to op-node by default even when
// DEVSTACK_L2CL_KIND=op-con-node (the historical verifier-only behavior), so
// each test opts its sequencer slot in with sysgo.L2CLOpConSequencer(). To run
// against op-con-node:
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencer ./op-acceptance-tests/tests/opcon/
//
// With the default op-node CL kind the option is a no-op and each test is an
// ordinary op-node sequencer check, so they remain valid suite additions
// rather than op-con-node-only tests.

// opConSequencerOpt opts the sequencer slot into op-con-node's sequencing mode
// (no-op for other CL kinds — see the package comment above).
func opConSequencerOpt() presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOpConSequencer())
}

// TestOpConSequencerAdvancesUnsafeHead is the op-con-node sequencer analogue of
// the base advance test (TestCLAdvance): the sequencer must produce a steady
// unsafe chain on its own — the minimal liveness property of sequencing mode.
// It exercises op-con-node's whole build pipeline: heartbeat -> SQL build
// request (unsafe-head cursor + L1-origin selection) -> engine FCU+attrs ->
// getPayload -> unsafe ingress -> head advance.
func TestOpConSequencerAdvancesUnsafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())

	// The sequencer produces a run of unsafe blocks on its own cadence.
	sys.L2CL.Advanced(types.LocalUnsafe, 5, 60)
}

// TestOpConSequencerBatchesDeriveSafeHead is the sequencer-side flip of
// TestOpConVerifierDerivesSafeHead: with op-con-node in the SEQUENCER slot, the
// blocks it builds must round-trip through the batcher and L1 derivation. The
// batcher submits the sequencer's blocks to L1; the sequencer's own safe head
// (it derives from L1 like a verifier) must catch up to its unsafe chain, and
// the verifier must derive the same safe chain and converge.
//
// This is the strongest end-to-end check of sequencing mode: a block that
// derivation would reject (bad deposit prefix, wrong L1-origin selection, bad
// timestamps) shows up here as a safe-head divergence or stall.
func TestOpConSequencerBatchesDeriveSafeHead(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t, opConSequencerOpt())

	// The sequencer produces blocks, the batcher lands them on L1, and the
	// sequencer's own L1 derivation confirms them as safe.
	sys.L2CL.Advanced(types.LocalSafe, 1, 60)

	// The verifier derives the same safe chain from L1 and converges.
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)
}

// TestOpConSequencerIncludesUserTransactions is the op-con-node sequencer
// analogue of the base transfer test (TestTransfer): a user transaction
// submitted to the sequencer's EL mempool must be included in a sequenced
// block. This pins the noTxPool=false build path — the sequencer supplies only
// the forced deposit prefix and the EL appends mempool transactions behind it
// (a noTxPool=true regression would produce deposit-only blocks and the
// transfer would never land).
func TestOpConSequencerIncludesUserTransactions(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())

	alice := sys.FunderL2.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	bobBalance := bob.GetBalance()

	transferAmount := eth.OneHundredthEther
	alice.Transfer(bob.Address(), transferAmount)
	bob.WaitForBalance(bobBalance.Add(transferAmount))
}

// TestOpConSequencerIncludesDeposits is the op-con-node sequencer analogue of
// the base deposit test (TestL1ToL2Deposit): a deposit made on L1 must be
// force-included in a sequenced L2 block once the sequencer selects an L1
// origin at or past the deposit's L1 block. This pins the sequencer's forced
// transaction prefix (L1-info + user deposits from the selected origin) and its
// drift-based L1-origin selection — if origins stopped advancing, the deposit
// would never be included.
func TestOpConSequencerIncludesDeposits(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())

	sys.L1Network.WaitForOnline()

	// Fund Alice on L1 and deposit through the portal.
	fundingAmount := eth.ThreeHundredthsEther
	alice := sys.FunderL1.NewFundedEOA(fundingAmount)
	alice.WaitForBalance(fundingAmount)

	aliceL2 := alice.AsEL(sys.L2EL)
	initialL2Balance := aliceL2.GetBalance()

	rollupConfig := sys.L2Chain.Escape().RollupConfig()
	portalAddr := rollupConfig.DepositContractAddress
	depositAmount := eth.OneHundredthEther

	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L2EL.Escape().EthClient()),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t))
	args := portal.DepositTransaction(alice.Address(), depositAmount, 300_000, false, []byte{})
	receipt := contract.Write(alice, args, txplan.WithValue(depositAmount))

	// The sequencer's L1 origin advances past the deposit's L1 block, forcing
	// the deposit into a sequenced block.
	t.Require().Eventually(func() bool {
		head := sys.L2CL.HeadBlockRef(types.LocalUnsafe)
		return head.L1Origin.Number >= bigs.Uint64Strict(receipt.BlockNumber)
	}, sys.L1EL.TransactionTimeout(), time.Second, "awaiting deposit to be processed by L2")

	aliceL2.WaitForBalance(initialL2Balance.Add(depositAmount))
}

// TestOpConSequencerStopsAndResumes exercises the sequencer control plane
// (admin_stopSequencer / admin_startSequencer / admin_sequencerActive — the
// same RPCs op-node serves, called through the shared DSL): stopping the
// sequencer halts unsafe block production, and restarting it resumes
// production from the halted head. There is no dedicated op-node acceptance
// test for this loop (it only appears inside larger scenarios), so this also
// covers the op-node path when run with the default CL kind.
func TestOpConSequencerStopsAndResumes(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())
	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime

	// The sequencer is live before we stop it.
	sys.L2CL.Advanced(types.LocalUnsafe, 2, 60)

	// Stop sequencing. In-flight builds may still land just after the stop is
	// acknowledged, so first wait for the unsafe head to hold still across one
	// block time, then require it to stay there for several more.
	sys.L2CL.StopSequencer()
	var stalledUnsafe uint64
	require.Eventually(t, func() bool {
		head := sys.L2CL.SyncStatus().UnsafeL2.Number
		if stalledUnsafe == 0 || head != stalledUnsafe {
			stalledUnsafe = head
			return false
		}
		return true
	}, time.Duration(blockTime*20)*time.Second, time.Duration(blockTime)*time.Second,
		"unsafe head did not stop advancing after StopSequencer")

	const stableSamples = 3
	for range stableSamples {
		time.Sleep(time.Duration(blockTime) * time.Second)
		require.Equal(t, stalledUnsafe, sys.L2CL.SyncStatus().UnsafeL2.Number,
			"sequencer kept producing blocks after StopSequencer")
	}

	// Restart sequencing from the halted head; production resumes.
	sys.L2CL.StartSequencer()
	require.Eventually(t, func() bool {
		return sys.L2CL.SyncStatus().UnsafeL2.Number > stalledUnsafe
	}, time.Duration(blockTime*30)*time.Second, time.Second,
		"sequencer did not resume producing blocks after StartSequencer")
}

// TestOpConSequencerRecoverModeStaysLive is the op-con-node sequencer analogue
// of the op-node recover-mode test (TestRecoverModeWhenChainHealthy): enabling
// recover mode (admin_setRecoverMode) on a healthy chain must not stall block
// production. In recover mode op-con-node sequences deposit-only blocks
// (user transactions are excluded), but the unsafe chain keeps advancing.
func TestOpConSequencerRecoverModeStaysLive(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())

	t.Require().NoError(sys.L2CL.SetSequencerRecoverMode(true))

	blockTime := sys.L2Chain.Escape().RollupConfig().BlockTime
	numL2Blocks := uint64(10)
	waitTime := time.Duration(blockTime*numL2Blocks+15) * time.Second

	start := sys.L2CL.SyncStatus().UnsafeL2.Number
	require.Eventually(t, func() bool {
		return sys.L2CL.SyncStatus().UnsafeL2.Number >= start+numL2Blocks
	}, waitTime, time.Duration(blockTime)*time.Second,
		"unsafe head stalled with recover mode enabled on a healthy chain")
}
