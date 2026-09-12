package privateinterop

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
)

func TestPrivateRecoveryFollowsL1Reorg(gt *testing.T) {
	testPrivateRecoveryReorg(gt, false)
}

func TestPrivateRecoveryRestartDiscardsReorgedProgress(gt *testing.T) {
	testPrivateRecoveryReorg(gt, true)
}

func testPrivateRecoveryReorg(gt *testing.T, restart bool) {
	gt.Helper()
	t := devtest.SerialT(gt)
	require := t.Require()
	record := func(phase string, evidence any) {
		data, err := json.Marshal(evidence)
		require.NoError(err)
		t.Logger().Info("Recovery reorg evidence", "phase", phase, "json", string(data))
	}
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck(), sysgo.WithPrivateInteropCadence(12)))
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	sys.L2BCL.Advanced(safety.CrossSafe, 20, 120)
	sys.L2BCL.Reached(safety.Finalized, 1, 180)
	finalized := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	record("baseline", map[string]any{"finalized_private": finalized, "restart": restart, "sequencing_window_l1_blocks": 10, "publication_l2_blocks": 12})
	rpc, err := client.NewRPC(t.Ctx(), t.Logger(), sys.PrivateInterop.FollowSource())
	require.NoError(err)
	t.Cleanup(rpc.Close)
	follow, err := sources.NewFollowClient(rpc)
	require.NoError(err)

	// Prepare the replacement before the invalid range exists. Reorged L1
	// transactions return to the mempool: building afterward could restore the
	// very claim whose withdrawal this test must exercise. Keep all followers
	// offline during preparation so the temporary branch cannot influence them.
	sys.L2BatcherB.Stop()
	sys.L2BatcherA.Stop()
	sys.L1CL.Stop()
	sys.L2BCL.Stop()
	sys.L2ACL.Stop()
	sys.Supernode.Stop()
	forkParent := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	const replacementLength = uint64(18) // Less than the manual builder's 20-block finality distance.
	for range replacementLength {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
	}
	replacementParent := sys.L1EL.BlockRefByNumber(forkParent.Number + replacementLength - 1)
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), forkParent.Hash)
	removed := sys.L1EL.BlockRefByNumber(forkParent.Number + 1)
	sys.Supernode.Start()
	sys.L2ACL.Start()
	sys.L2BCL.Start()
	sys.L1CL.Start()
	sys.L2BatcherA.Start()

	// Put private state changes in the surviving prefix. Removing its claim
	// must then change private hashes even if the old L1 origins stay canonical.
	sys.L2ELB.WaitForL1Origin(removed.Number)
	marker, err := alice.Transfer(common.Address{0xec}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	parent := sys.L2BCL.StopSequencer()
	source := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	bad := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](alice.Plan())
	bad.Content.Set(&txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: messages.Message{
		Identifier: messages.Identifier{Origin: alice.Address(), BlockNumber: source.Number, LogIndex: 1024,
			Timestamp: source.Time, ChainID: sys.L2A.ChainID()}, PayloadHash: common.Hash{1}}})
	signed, err := bad.PlannedTx.Signed.Eval(t.Ctx())
	require.NoError(err)
	raw, err := signed.MarshalBinary()
	require.NoError(err)
	sys.TestSequencer.SequenceBlockWithTxs(t, sys.L2B.ChainID(), parent, [][]byte{raw})
	invalid := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	receipt := sys.L2ELB.WaitForReceipt(signed.Hash())
	require.Equal(uint64(1), receipt.Status)
	require.NotEmpty(receipt.Logs)
	require.Equal(predeploys.CrossL2InboxAddr, receipt.Logs[0].Address)
	record("invalid_execution", map[string]any{"receipt": receipt, "block": invalid, "marker_receipt": marker})
	sys.L2BCL.ReachedRef(safety.LocalUnsafe, invalid.ID(), 30)
	sys.L2BCL.StartSequencer()
	sys.L2BatcherB.Start()

	var before *sources.FollowStatus
	var replayed eth.L2BlockRef
	require.Eventually(func() bool {
		status, err := follow.GetFollowStatus(t.Ctx())
		if err != nil || status.Recovery == nil || status.Recovery.Prefix == nil {
			return false
		}
		private, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		if err != nil || private.LocalSafeL2.Number < invalid.Number || private.LocalSafeL2.Number > status.Recovery.Prefix.Parent.Number {
			return false
		}
		ref, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByNumber(t.Ctx(), invalid.Number)
		if err != nil || ref.Hash == invalid.Hash || ref.ParentHash != invalid.ParentHash {
			return false
		}
		before, replayed = status, ref
		return true
	}, 3*time.Minute, 100*time.Millisecond, "execute a replacement while the reserved recovery range remains incomplete")
	require.GreaterOrEqual(replayed.L1Origin.Number, removed.Number, "the reorg must remove inputs of an executed recovery block")
	sys.L2BatcherB.Stop()
	sys.L2BatcherA.Stop()
	sys.L1CL.Stop()
	if restart {
		sys.L2BCL.Stop() // The journal still belongs to the old public branch.
	}
	require.Less(sys.L1EL.BlockRefByLabel(eth.Unsafe).Number, replacementParent.Number+1,
		"the prepared replacement must still be longer than the publication fork")
	require.LessOrEqual(sys.L1EL.BlockRefByLabel(eth.Finalized).Number, forkParent.Number)
	t.Logger().Info("Reorg during private recovery", "restart", restart, "invalid", invalid,
		"replayed", replayed, "marker", marker.TxHash, "recovery", before.Recovery, "l1_fork_parent", forkParent)
	record("before_reorg", map[string]any{"status": before, "replayed": replayed, "l1_fork_parent": forkParent, "removed_l1": removed, "replacement_parent": replacementParent})
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), replacementParent.Hash)
	sys.L1EL.ReorgTriggered(removed, 10)
	require.Eventually(func() bool {
		_, err := follow.RecoveryBlock(t.Ctx(), replayed.Number, before.Recovery.Target.ID())
		if err == nil {
			return false
		}
		after, err := follow.GetFollowStatus(t.Ctx())
		changed := err == nil && after.Recovery != nil &&
			(after.Recovery.Anchor != before.Recovery.Anchor || after.Recovery.Target != before.Recovery.Target || after.Recovery.Prefix == nil)
		if changed {
			record("revoked_snapshot", after)
		}
		return changed
	}, time.Minute, 100*time.Millisecond, "the L1 reorg must revoke the actual recovery snapshot")
	if restart {
		sys.L2BCL.Start() // Reuse the same EL database and recovery journal.
	}
	require.Eventually(func() bool {
		ref, err := sys.L2ELB.Escape().L2EthClient().L2BlockRefByNumber(t.Ctx(), replayed.Number)
		return err != nil || ref.Hash != replayed.Hash
	}, time.Minute, 100*time.Millisecond, "discard private recovery executed against the abandoned publication")
	require.True(sys.L2ELB.IsCanonical(finalized.ID()), "the reorg must preserve finalized private ancestry")
	sys.L1CL.Start()
	sys.L2BatcherA.Start()
	sys.L2BatcherB.Start()
	sys.L2BCL.Reached(safety.CrossSafe, before.Recovery.Prefix.Parent.Number+12, 240)

	// Use a fresh sender: the marker and invalid execution may legitimately
	// disappear, so their sender's pre-reorg nonce is not a survival requirement.
	fresh := sys.FunderB.NewFundedEOA(eth.OneEther)
	post, err := fresh.Transfer(common.Address{0xed}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	sys.L2BCL.ReachedRef(safety.CrossSafe, eth.BlockID{Hash: post.BlockHash, Number: bigs.Uint64Strict(post.BlockNumber)}, 180)
	publicSender := sys.FunderA.NewFundedEOA(eth.OneEther)
	for _, direction := range []struct {
		from, to     *dsl.EOA
		fromCL, toCL *dsl.L2CLNode
	}{
		{publicSender, fresh, sys.L2ACL, sys.L2BCL}, {fresh, publicSender, sys.L2BCL, sys.L2ACL},
	} {
		send := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](direction.from.Plan())
		send.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr,
			DestChainID: direction.to.ChainID(), Target: common.Address{0xee}})
		sent, err := send.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		require.Equal(uint64(1), sent.Status)
		relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](direction.to.Plan())
		relay.Content.DependOn(&send.Result)
		relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 0))
		received, err := relay.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		require.Equal(uint64(1), received.Status)
		require.NotEmpty(received.Logs)
		require.Equal(predeploys.CrossL2InboxAddr, received.Logs[0].Address)
		direction.fromCL.ReachedRef(safety.CrossSafe, eth.BlockID{Hash: sent.BlockHash, Number: bigs.Uint64Strict(sent.BlockNumber)}, 180)
		direction.toCL.ReachedRef(safety.CrossSafe, eth.BlockID{Hash: received.BlockHash, Number: bigs.Uint64Strict(received.BlockNumber)}, 180)
		record("cross_safe_message", map[string]any{"sent": sent, "received": received})
	}
	replacement := sys.L2ELB.BlockRefByNumber(replayed.Number)
	projection := sys.L2BSupernodeEL.BlockRefByNumber(replayed.Number)
	require.Equal(projection.L1Origin, replacement.L1Origin)
	require.Equal(projection.Time, replacement.Time)
	require.Equal(sys.L1EL.BlockRefByNumber(replacement.L1Origin.Number).ID(), replacement.L1Origin)
	_, txs, err := sys.L2ELB.Escape().EthClient().InfoAndTxsByHash(t.Ctx(), replacement.Hash)
	require.NoError(err)
	for _, tx := range txs {
		require.Equal(uint8(0x7e), tx.Type(), "recovery must execute deposit-only replacements")
	}
	require.False(sys.L2ELB.IsCanonical(invalid.ID()))
	require.False(sys.L2ELB.IsCanonical(replayed.ID()))
	after, err := follow.GetFollowStatus(t.Ctx())
	require.NoError(err)
	record("converged", map[string]any{"status": after, "replacement": replacement, "projection": projection, "post_recovery_receipt": post})
	t.Logger().Info("Private recovery converged after L1 reorg", "restart", restart, "transaction", post.TxHash, "block", post.BlockHash)
}
