package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestPrivateRevertedReplayInvalidatesBlock exercises the projection execution rule
// (spec-sound-profile §E): a carrier transaction that fails invalidates its projection block.
//
// The batcher's test hooks give the import replay a gas limit just above its intrinsic cost, so the
// transaction is valid to include but runs out of gas inside validateMessage, and skip the batcher's
// own carrier pre-run that would otherwise refuse to publish it. Before the rule, the span was
// admitted and the replay silently dropped its ExecutingMessage log. Now op-reth refuses the block,
// Holocene replaces it with a deposit-only block and drops the rest of the span, private safety
// holds below it, and the next span recovers in recovery mode. With the hooks cleared, the message
// is published and appears on the projection.
//
// It runs under the legacy execution-mock-v1 verifier for speed: the execution rule is independent
// of the verifier.
func TestPrivateRevertedReplayInvalidatesBlock(gt *testing.T) {
	testPrivateRevertedReplayInvalidatesBlock(gt, false)
}

// testPrivateRevertedReplayInvalidatesBlock runs the scenario; skipPreRunAlways keeps the carrier
// pre-run disabled for the whole run instead of only while the fault is armed (diagnostics).
func testPrivateRevertedReplayInvalidatesBlock(gt *testing.T, skipPreRunAlways bool) {
	gt.Helper()
	hooks := &bss.PrivateInteropTestHooks{SkipCarrierPreRun: skipPreRunAlways}
	command := privateProjectionExecutor(gt)
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithPrivateInteropExecutionMock(command),
			sysgo.WithPrivateInteropTestHooks(hooks), sysgo.WithoutRenderingInvariantCheck()))
	require := t.Require()
	logger := t.Logger()

	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.SafeL2.Number >= 2
	}, 4*time.Minute, time.Second, "private safety should advance before the fault")

	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	// The initiating message on the counterparty. Its identifier fixes the replay's calldata and
	// access list, hence its exact intrinsic gas.
	send := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](alice.Plan())
	send.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr, DestChainID: bob.ChainID(), Target: common.Address{0xab}})
	sent, err := send.Result.Eval(t.Ctx())
	require.NoError(err)
	require.Len(sent.Entries, 1)
	msg := sent.Entries[0]
	sys.L2A.WaitForBlock()

	replay := &txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: msg}
	calldata, err := replay.EncodeInput()
	require.NoError(err)
	accessList := types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: messages.EncodeAccessList([]messages.Access{msg.Access()})}}
	intrinsic, err := core.IntrinsicGas(calldata, accessList, nil, false, true, true, true)
	require.NoError(err)
	floor, err := core.FloorDataGas(calldata)
	require.NoError(err)
	underGas := max(intrinsic, floor) + 100
	logger.Info("Under-gassing the import replay", "intrinsic", intrinsic, "floor", floor, "gas", underGas)

	// Arm the fault with the batcher stopped, so the hooks never change under a running range.
	sys.L2BatcherB.Stop()
	policy := render.DefaultGasPolicy()
	policy.GasLimitImport = underGas
	hooks.GasPolicyOverride = &policy
	hooks.SkipCarrierPreRun = true
	sys.L2BatcherB.Start()

	relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](bob.Plan())
	relay.Content.DependOn(&send.Result)
	relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 0))
	relayed, err := relay.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	height := bigs.Uint64Strict(relayed.BlockNumber)
	originalPrivate := eth.BlockID{Hash: relayed.BlockHash, Number: height}
	logger.Info("Private import included", "height", height, "tx", relayed.TxHash)

	// The projection derives up to the invalid block, which becomes deposit-only.
	projection := sys.L2BSupernodeEL.Escape().L2EthClient()
	require.Eventually(func() bool {
		safe := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
		return safe.Number >= height
	}, 5*time.Minute, time.Second, "the projection must derive the invalidated height")
	_, txs, err := projection.InfoAndTxsByNumber(t.Ctx(), height)
	require.NoError(err)
	require.NotEmpty(txs)
	for i, tx := range txs {
		require.Equal(uint8(types.DepositTxType), tx.Type(),
			"projection block %d must be deposit-only after its reverted replay; tx %d is a sequencer transaction", height, i)
	}
	logger.Info("The projection replaced the block with a deposit-only block", "height", height, "deposits", len(txs))

	// Private safety holds at or below height-1 until a later span is admitted.
	rpc, err := client.NewRPC(t.Ctx(), logger, sys.PrivateInterop.FollowSource())
	require.NoError(err)
	t.Cleanup(rpc.Close)
	follow, err := sources.NewFollowClient(rpc)
	require.NoError(err)
	laterSpanAdmitted := func() bool {
		safe := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
		for n := height + 1; n <= safe.Number; n++ {
			_, txs, err := projection.InfoAndTxsByNumber(t.Ctx(), n)
			if err == nil && claimInBlock(txs) != nil {
				return true
			}
		}
		return false
	}
	for range 10 {
		status, err := follow.GetFollowStatus(t.Ctx())
		require.NoError(err)
		if laterSpanAdmitted() {
			break
		}
		require.Less(status.SafeL2.Number, height, "private safety must not pass the invalidated block before a later span")
		time.Sleep(time.Second)
	}

	// Clear the fault. The next span is admitted in recovery mode.
	sys.L2BatcherB.Stop()
	hooks.GasPolicyOverride = nil
	hooks.SkipCarrierPreRun = skipPreRunAlways
	sys.L2BatcherB.Start()
	require.Eventually(laterSpanAdmitted, 6*time.Minute, time.Second, "a span after the invalidated block must be admitted")
	var recovery bool
	safe := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
	for n := height + 1; n <= safe.Number && !recovery; n++ {
		_, txs, err := projection.InfoAndTxsByNumber(t.Ctx(), n)
		require.NoError(err)
		if c := claimInBlock(txs); c != nil {
			require.Less(c.AnchorBlock, c.FirstBlock-1, "the first span after the invalidation continues in recovery mode")
			require.Less(c.AnchorBlock, height, "its anchor is the last surviving record, below the invalidated block")
			require.NotEqual(common.Hash{}, c.RecoveryHash)
			recovery = true
		}
	}
	require.True(recovery)
	require.False(sys.L2ELB.IsCanonical(originalPrivate), "the private block carrying the reverted replay's import is replaced")

	// The message appears once the import is private again. The private recovery replaced the
	// block that carried it; the execution client may re-include the original transaction from
	// its pool, and otherwise the relay is sent again.
	var againHeight uint64
	if rec, err := sys.L2ELB.Escape().EthClient().TransactionReceipt(t.Ctx(), relayed.TxHash); err == nil &&
		rec.Status == types.ReceiptStatusSuccessful &&
		sys.L2ELB.IsCanonical(eth.BlockID{Hash: rec.BlockHash, Number: bigs.Uint64Strict(rec.BlockNumber)}) {
		againHeight = bigs.Uint64Strict(rec.BlockNumber)
		logger.Info("The original import was re-included after recovery", "height", againHeight)
	} else {
		again := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](bob.Plan())
		again.Content.DependOn(&send.Result)
		again.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 0))
		relayedAgain, err := again.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, relayedAgain.Status, "the re-relay executes on the private chain")
		againHeight = bigs.Uint64Strict(relayedAgain.BlockNumber)
	}
	require.Eventually(func() bool {
		return sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe).Number >= againHeight
	}, 6*time.Minute, time.Second, "the re-relayed import must be published")
	againRef := sys.L2BSupernodeEL.BlockRefByNumber(againHeight)
	_, receipts, err := projection.FetchReceipts(t.Ctx(), againRef.Hash)
	require.NoError(err)
	found := false
	for _, r := range receipts.Geth() {
		for _, l := range r.Logs {
			if l.Address == predeploys.CrossL2InboxAddr {
				require.Equal(types.ReceiptStatusSuccessful, r.Status, "the published import replay succeeds")
				found = true
			}
		}
	}
	require.True(found, "the projection block at %d carries the ExecutingMessage of the import", againHeight)
}
