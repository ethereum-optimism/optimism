package privateinterop

import (
	"encoding/json"
	"os"
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
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// This soak retains the production sequencing window and publication cadence.
// The reorg revokes the rejected range before expiry; waiting for full-window
// expiry is a separate, approximately six-hour scenario in this L1 fixture.
func TestPrivateRecoveryAcrossL1ReorgAtProductionCadence(gt *testing.T) {
	if os.Getenv("PRIVATE_INTEROP_REORG_SOAK") != "1" {
		gt.Skip("set PRIVATE_INTEROP_REORG_SOAK=1 for the production recovery/reorg soak")
	}
	runPrivateReorgSoak(gt, 300, 90*time.Minute)
}

type soakHistory struct {
	el    *dsl.L2ELNode
	cl    *dsl.L2CLNode
	block eth.BlockID
	tx    common.Hash
}

type privateReorgSoak struct {
	t        devtest.T
	sys      *presets.TwoL2SupernodeInterop
	follow   *sources.FollowClient
	cadence  uint64
	attempts int
	started  time.Time
	history  []soakHistory
}

func (s *privateReorgSoak) record(phase string, evidence any) {
	data, err := json.Marshal(evidence)
	s.t.Require().NoError(err)
	s.t.Logger().Info("Production recovery soak evidence", "phase", phase,
		"elapsed", time.Since(s.started), "json", string(data))
}

func (s *privateReorgSoak) heads(phase string) {
	t, sys := s.t, s.sys
	private, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
	t.Require().NoError(err)
	public, err := sys.L2ACL.Escape().RollupAPI().SyncStatus(t.Ctx())
	t.Require().NoError(err)
	claimed, err := s.follow.GetFollowStatus(t.Ctx())
	t.Require().NoError(err)
	projection, err := sys.L2BSupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
	t.Require().NoError(err)
	s.record(phase, map[string]any{"private": private, "public": public,
		"projection": projection, "claimed": claimed, "l1": sys.L1EL.BlockRefByLabel(eth.Unsafe)})
}

func runPrivateReorgSoak(gt *testing.T, cadence uint64, minimum time.Duration) {
	gt.Helper()
	t := devtest.SerialT(gt)
	require := t.Require()
	// Reached polls every two seconds: allow two publication ranges plus
	// four minutes of derivation/restart overhead per progress phase.
	s := &privateReorgSoak{t: t, cadence: cadence, started: time.Now(), attempts: int(2*cadence + 120)}
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(3600)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck(), sysgo.WithPrivateInteropCadence(cadence)))
	s.sys = sys
	for _, network := range []*dsl.L2Network{sys.L2A, sys.L2B} {
		require.Equal(uint64(2), network.Escape().RollupConfig().BlockTime)
		require.Equal(uint64(3600), network.Escape().RollupConfig().SeqWindowSize)
	}
	rpc, err := client.NewRPC(t.Ctx(), t.Logger(), sys.PrivateInterop.FollowSource())
	require.NoError(err)
	t.Cleanup(rpc.Close)
	s.follow, err = sources.NewFollowClient(rpc)
	require.NoError(err)
	s.record("configuration", map[string]any{"sequencing_window_l1_blocks": 3600,
		"publication_l2_blocks": cadence, "l2_block_seconds": 2, "minimum_duration": minimum.String(),
		"phase_deadline_seconds": 2 * s.attempts})

	// Start with the existing production-soak workload: retain two complete
	// private ranges, then require publication to catch up without losing state.
	sys.L2BatcherB.Stop()
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	bob := sys.FunderA.NewFundedEOA(eth.OneEther)
	initial, err := alice.Transfer(common.Address{0xea}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	publicInitial, err := bob.Transfer(common.Address{0xeb}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	sys.L2BCL.Reached(safety.LocalUnsafe, 2*cadence, s.attempts)
	s.heads("two_ranges_accumulated")
	sys.L2BatcherB.Start()
	sys.L2BCL.Reached(safety.CrossSafe, 2*cadence, s.attempts)
	initialID := eth.BlockID{Hash: initial.BlockHash, Number: bigs.Uint64Strict(initial.BlockNumber)}
	publicID := eth.BlockID{Hash: publicInitial.BlockHash, Number: bigs.Uint64Strict(publicInitial.BlockNumber)}
	sys.L2BCL.ReachedRef(safety.CrossSafe, initialID, s.attempts)
	sys.L2ACL.ReachedRef(safety.CrossSafe, publicID, s.attempts)
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	l1Parent := sys.L1EL.BlockRefByNumber(l1Head.Number - 1)
	s.record("observed_l1_timing", map[string]any{"head": l1Head, "parent": l1Parent, "block_seconds": l1Head.Time - l1Parent.Time})
	s.record("baseline_transfers", map[string]any{"private": initial, "public": publicInitial})
	s.messages(alice, bob)
	sys.L2BCL.Reached(safety.Finalized, 1, 180)
	s.heads("baseline_cross_safe")

	s.reorg(false)
	s.reorg(true)

	// Hold the source stable across a completed-recovery restart. No new
	// deliberate fault is introduced after this point, so this suffix must live.
	alice = sys.FunderB.NewFundedEOA(eth.OneEther)
	sys.L2BatcherB.Stop()
	sys.L1CL.Stop()
	newer, err := alice.Transfer(common.Address{0xed}, eth.OneGWei).Included.Eval(t.Ctx())
	require.NoError(err)
	newerID := eth.BlockID{Hash: newer.BlockHash, Number: bigs.Uint64Strict(newer.BlockNumber)}
	head := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	s.heads("before_completed_restart")
	s.record("newer_private_history", map[string]any{"head": head, "receipt": newer})
	sys.L2BCL.Stop()
	sys.L2BCL.Start()
	sys.L2BCL.Reached(safety.LocalUnsafe, head.Number+2, 30)
	require.True(sys.L2ELB.IsCanonical(head.ID()), "completed recovery must not replay over the newer suffix")
	require.True(sys.L2ELB.IsCanonical(newerID))
	sys.L1CL.Start()
	sys.L2BatcherB.Start()
	sys.L2BCL.ReachedRef(safety.CrossSafe, newerID, s.attempts)
	s.heads("completed_restart_preserved_history")

	// Restart the supernode separately with its existing databases, then prove
	// fresh publication and messages in both directions still become cross-safe.
	sys.L1CL.Stop()
	s.heads("before_supernode_restart")
	publicBeforeRestart := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	privateBeforeRestart := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.Supernode.Stop()
	sys.Supernode.Start()
	require.Eventually(func() bool {
		status, err := s.follow.GetFollowStatus(t.Ctx())
		if err != nil || status.Recovery == nil {
			return false
		}
		_, err = sys.L2BSupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil
	}, time.Minute, time.Second, "restarted supernode must serve its persisted recovery state")
	s.heads("after_supernode_restart")
	sys.L1CL.Start()
	s.messages(alice, bob)
	require.True(sys.L2ELA.IsCanonical(publicBeforeRestart.ID()), "supernode restart must preserve newer public history")
	require.True(sys.L2ELB.IsCanonical(privateBeforeRestart.ID()), "supernode restart must preserve newer private history")
	s.record("supernode_restart_preserved_history", map[string]any{"public": publicBeforeRestart, "private": privateBeforeRestart})
	for time.Since(s.started) < minimum {
		receipt, err := alice.Transfer(common.Address{0xee}, eth.OneGWei).Included.Eval(t.Ctx())
		require.NoError(err)
		id := eth.BlockID{Hash: receipt.BlockHash, Number: bigs.Uint64Strict(receipt.BlockNumber)}
		sys.L2BCL.ReachedRef(safety.CrossSafe, id, s.attempts)
		s.record("continued_cross_safe_traffic", receipt)
		s.heads("continued_progress")
	}
	require.True(sys.L2ELB.IsCanonical(initialID))
	require.True(sys.L2ELA.IsCanonical(publicID))
	require.True(sys.L2ELB.IsCanonical(newerID))
	for _, h := range s.history {
		h.cl.ReachedRef(safety.CrossSafe, h.block, s.attempts)
		require.True(h.el.IsCanonical(h.block), "surviving message history must remain canonical")
		s.record("drained_message", map[string]any{"transaction": h.tx, "block": h.block})
	}
	s.heads("drained")
}

// Submit both initiating messages before waiting for a publication boundary,
// and both relays before waiting for the next one.
func (s *privateReorgSoak) messages(alice, bob *dsl.EOA) {
	t, sys := s.t, s.sys
	require := t.Require()
	sends := make([]*txintent.IntentTx[*txintent.SendTrigger, *txintent.InteropOutput], 0, 2)
	directions := []struct {
		from, to     *dsl.EOA
		fromCL, toCL *dsl.L2CLNode
		fromEL, toEL *dsl.L2ELNode
	}{
		{bob, alice, sys.L2ACL, sys.L2BCL, sys.L2ELA, sys.L2ELB}, {alice, bob, sys.L2BCL, sys.L2ACL, sys.L2ELB, sys.L2ELA},
	}
	for _, d := range directions {
		send := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](d.from.Plan())
		send.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr,
			DestChainID: d.to.ChainID(), Target: common.Address{0xef}})
		receipt, err := send.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		require.Equal(uint64(1), receipt.Status)
		sends = append(sends, send)
		s.history = append(s.history, soakHistory{el: d.fromEL, cl: d.fromCL, block: eth.BlockID{Hash: receipt.BlockHash, Number: bigs.Uint64Strict(receipt.BlockNumber)}, tx: receipt.TxHash})
		s.record("initiating_message", receipt)
	}
	for i, d := range directions {
		receipt, err := sends[i].PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		d.fromCL.ReachedRef(safety.CrossSafe, eth.BlockID{Hash: receipt.BlockHash, Number: bigs.Uint64Strict(receipt.BlockNumber)}, s.attempts)
	}
	ids := make([]eth.BlockID, 0, 2)
	for i, d := range directions {
		relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](d.to.Plan())
		relay.Content.DependOn(&sends[i].Result)
		relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &sends[i].Result, &sends[i].PlannedTx.Included, 0))
		receipt, err := relay.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(err)
		require.Equal(uint64(1), receipt.Status)
		require.NotEmpty(receipt.Logs)
		require.Equal(predeploys.CrossL2InboxAddr, receipt.Logs[0].Address)
		ids = append(ids, eth.BlockID{Hash: receipt.BlockHash, Number: bigs.Uint64Strict(receipt.BlockNumber)})
		s.history = append(s.history, soakHistory{el: d.toEL, cl: d.toCL, block: ids[len(ids)-1], tx: receipt.TxHash})
		s.record("executing_message", receipt)
	}
	for i, d := range directions {
		d.toCL.ReachedRef(safety.CrossSafe, ids[i], s.attempts)
	}
	s.heads("both_message_directions_cross_safe")
}

func (s *privateReorgSoak) reorg(restart bool) {
	t, sys, follow := s.t, s.sys, s.follow
	require := t.Require()
	record := s.record
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	finalized := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	require.GreaterOrEqual(s.cadence, uint64(30))
	lead := min(uint64(30), s.cadence-10)
	// A previous claim can still be in flight after cross-safe catches up.
	// Select a range only once enough unpublished positions remain for fault
	// preparation; otherwise a second injection can land in the following range.
	require.Eventually(func() bool {
		status, err := follow.GetFollowStatus(t.Ctx())
		if err != nil || status.Recovery == nil || status.Recovery.Prefix != nil {
			return false
		}
		private, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && private.UnsafeL2.Number+lead <= status.LocalSafeL2.Number+s.cadence
	}, time.Duration(2*s.attempts)*time.Second, time.Second, "publication must leave room for the next deliberate fault")
	sys.L2BatcherB.Stop()
	status, err := follow.GetFollowStatus(t.Ctx())
	require.NoError(err)
	require.NotNil(status.Recovery)
	rangeEnd := status.LocalSafeL2.Number + s.cadence
	sys.L2BCL.Reached(safety.LocalUnsafe, rangeEnd-30, s.attempts)
	s.heads("approaching_fault_publication")
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
	// Finish the already-published manual job so a second fault can open a
	// new job. Next skips completed steps and releases this sealed job.
	require.NoError(sys.TestSequencer.Escape().ControlAPI(sys.L2B.ChainID()).Next(t.Ctx()))
	require.Equal(invalid, sys.L2ELB.BlockRefByLabel(eth.Unsafe))
	require.LessOrEqual(invalid.Number, rangeEnd, "inject within the next publication range")
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
	_, replacementTxs, err := sys.L2ELB.Escape().EthClient().InfoAndTxsByHash(t.Ctx(), replayed.Hash)
	require.NoError(err)
	for _, tx := range replacementTxs {
		require.Equal(uint8(0x7e), tx.Type(), "the interrupted recovery block must be deposit-only")
	}
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
	sys.L2BCL.Reached(safety.CrossSafe, before.Recovery.Prefix.Parent.Number+s.cadence, s.attempts)

	require.False(sys.L2ELB.IsCanonical(invalid.ID()))
	require.False(sys.L2ELB.IsCanonical(replayed.ID()))
	afterReceipt, err := sys.L2ELB.Escape().EthClient().TransactionReceipt(t.Ctx(), signed.Hash())
	if err == nil {
		require.False(sys.L2ELB.IsCanonical(eth.BlockID{Hash: afterReceipt.BlockHash, Number: bigs.Uint64Strict(afterReceipt.BlockNumber)}),
			"invalid execution must not reappear in canonical history")
	} else {
		require.ErrorIs(err, ethereum.NotFound)
	}
	replacement := sys.L2ELB.BlockRefByNumber(replayed.Number)
	projection := sys.L2BSupernodeEL.BlockRefByNumber(replayed.Number)
	require.Equal(projection.L1Origin, replacement.L1Origin)
	require.Equal(projection.Time, replacement.Time)
	require.Equal(sys.L1EL.BlockRefByNumber(replacement.L1Origin.Number).ID(), replacement.L1Origin)
	// After its entire claim is revoked, the new plan may permit ordinary
	// sequencing here. Deposit-only content was asserted at interruption above.
	record("reorg_converged", map[string]any{"restart": restart, "replacement": replacement, "projection": projection})
	s.heads("after_reorg_catchup")
}
