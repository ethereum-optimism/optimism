package reorg

import (
	"context"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// These tests exercise op-conductor's reorg-recovery mode end-to-end against a real
// op-reth backend: a real `reth_subscribeChainNotifications` reorg notification must
// drive the leader to commit the post-reorg unsafe head into the Raft FSM, and that
// state must replicate cluster-wide. They depend on the `reth` namespace, which op-geth
// does not serve, so they skip on any non-op-reth backend.

const (
	reorgObservedMetric  = "op_conductor_reorgs_observed_count"
	reorgCommittedMetric = "op_conductor_reorgs_committed_count"

	// deepReorgBuildPerBlock budgets wall-clock time per L2 block while burying the invalid
	// block under reorgDepth descendants and while the chain re-derives them after the
	// reorg. L2 blocks are produced in real time, so a deep reorg needs a long window.
	deepReorgBuildPerBlock = 4 * time.Second
	// deepReorgBuildBase is fixed slack on top of the per-block build/settle budget.
	deepReorgBuildBase = 120 * time.Second
)

// deepReorgBuildTimeout returns a generous timeout for building or re-deriving reorgDepth
// L2 blocks at real-time block production.
func deepReorgBuildTimeout(reorgDepth uint64) time.Duration {
	return time.Duration(reorgDepth)*deepReorgBuildPerBlock + deepReorgBuildBase
}

// requireOpReth skips the test unless the EL backend is op-reth. The acceptance CI
// matrix runs the same package under an op-geth job (which leaves DEVSTACK_L2EL_KIND
// unset) and op-reth jobs (which set it).
func requireOpReth(t devtest.T) {
	if os.Getenv("DEVSTACK_L2EL_KIND") != "op-reth" {
		t.Skip("conductor reorg-recovery requires the op-reth reth_subscribeChainNotifications subscription")
	}
}

// reorgTrigger describes the invalid-message reorg that was forced on chain B.
type reorgTrigger struct {
	invalidBlockNumber    uint64
	invalidBlockHash      common.Hash
	invalidBlockTimestamp uint64
}

// triggerInvalidMessageReorgOnB forces a same-height, single-block reorg on chain B by
// including an invalid executing message and resuming interop, which invalidates and
// replaces the offending block. It returns once the EL has replaced the block. Mirrors the
// trigger in invalid_message_reorg_test.go.
func triggerInvalidMessageReorgOnB(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors) reorgTrigger {
	return triggerInvalidMessageReorgOnBWithDepth(t, sys, 0)
}

// triggerInvalidMessageReorgOnBWithDepth is triggerInvalidMessageReorgOnB with an optional
// reorg depth: while interop stays paused, the chain is extended reorgDepth blocks past the
// invalid executing message before interop is resumed. Resuming then invalidates the buried
// block and re-derives every descendant, producing a reorg that spans ~reorgDepth blocks
// (the EL unsafe head transiently rewinds to the replacement before climbing back). A depth
// of 0 reproduces the shallow same-height replacement.
func triggerInvalidMessageReorgOnBWithDepth(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors, reorgDepth uint64) reorgTrigger {
	ctx := t.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()

	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	paused := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", paused)

	rng := rand.New(rand.NewSource(12345))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()

	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)
	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber, "hash", invalidBlockHash, "timestamp", invalidBlockTimestamp, "depth", reorgDepth)

	require.Eventually(t, func() bool {
		return sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	// Bury the invalid block under reorgDepth descendants while interop is paused, so the
	// invalidation on resume rewinds a deep span rather than a single block. Capture a deep
	// descendant's hash to later prove the reorg actually cascaded that far.
	var deepDescendant eth.BlockRef
	if reorgDepth > 0 {
		target := invalidBlockNumber + reorgDepth
		require.Eventually(t, func() bool {
			return sys.L2BCL.SyncStatus().UnsafeL2.Number >= target
		}, deepReorgBuildTimeout(reorgDepth), time.Second,
			"chain B must extend reorgDepth blocks past the invalid block while interop is paused")
		var err error
		deepDescendant, err = sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber+reorgDepth-1)
		require.NoError(t, err, "read deep descendant before resume")
		t.Logger().Info("buried invalid block under descendants",
			"invalid", invalidBlockNumber, "depth", reorgDepth,
			"unsafe_head", sys.L2BCL.SyncStatus().UnsafeL2.Number, "deep_descendant", deepDescendant.Hash)
	}

	sys.Supernode.ResumeInterop()
	replaceTimeout := 60 * time.Second
	if reorgDepth > 0 {
		replaceTimeout = deepReorgBuildTimeout(reorgDepth)
	}
	require.Eventually(t, func() bool {
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			return false
		}
		return currentBlock.Hash != invalidBlockHash
	}, replaceTimeout, time.Second, "the invalid block at the reorg height should be replaced at the EL")

	if reorgDepth > 0 {
		require.Eventually(t, func() bool {
			cur, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, deepDescendant.Number)
			return err == nil && cur.Hash != deepDescendant.Hash
		}, replaceTimeout, time.Second,
			"a deep descendant block must be reorged, proving the reorg spanned reorgDepth blocks")
		t.Logger().Info("deep reorg confirmed: descendant reorged", "descendant_block", deepDescendant.Number)
	}

	return reorgTrigger{
		invalidBlockNumber:    invalidBlockNumber,
		invalidBlockHash:      invalidBlockHash,
		invalidBlockTimestamp: invalidBlockTimestamp,
	}
}

func conductorsForChainB(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors) dsl.ConductorSet {
	conductors := sys.ConductorSets[sys.L2B.Escape().ChainID()]
	require.NotEmpty(t, conductors, "expected conductors for chain B")
	return conductors
}

func findLeader(t devtest.T, conductors dsl.ConductorSet) *dsl.Conductor {
	var leader *dsl.Conductor
	require.Eventually(t, func() bool {
		for _, c := range conductors {
			if c.IsLeader() {
				leader = c
				return true
			}
		}
		return false
	}, 30*time.Second, 500*time.Millisecond, "a conductor should be leader")
	return leader
}

// assertLeaderHeadConverges asserts the leader's recorded unsafe head sits on the
// canonical EL chain at or beyond the reorg height — i.e. the FSM followed the reorg.
// LatestUnsafePayload is a leader-only linearizable read (it applies a Raft barrier), so
// only the leader can serve it; Raft guarantees the committed entry replicates to
// followers, and Scenario B independently confirms a former follower has it after a
// leadership transfer.
func assertLeaderHeadConverges(t devtest.T, conductors dsl.ConductorSet, elClient ethBlockByNumber, minNumber uint64) {
	ctx := t.Ctx()
	require.Eventually(t, func() bool {
		leader := currentLeader(ctx, conductors)
		if leader == nil {
			return false
		}
		head, err := leaderUnsafePayload(ctx, leader)
		if err != nil || head == nil {
			return false
		}
		num := uint64(head.ExecutionPayload.BlockNumber)
		if num < minNumber {
			return false
		}
		ref, err := elClient.BlockRefByNumber(ctx, num)
		return err == nil && ref.Hash == head.ExecutionPayload.BlockHash
	}, 90*time.Second, time.Second, "the leader's FSM head should converge to the canonical post-reorg chain")
}

// currentLeader returns the conductor that currently reports itself leader, or nil. The
// Leader RPC is a local check (no Raft barrier), so it is safe to call on followers.
func currentLeader(ctx context.Context, conductors dsl.ConductorSet) *dsl.Conductor {
	for _, c := range conductors {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		isLeader, err := c.Escape().RpcAPI().Leader(callCtx)
		cancel()
		if err == nil && isLeader {
			return c
		}
	}
	return nil
}

// leaderUnsafePayload reads the leader's recorded unsafe head without the DSL's
// fail-on-error wrapper, so callers can poll inside require.Eventually.
func leaderUnsafePayload(ctx context.Context, leader *dsl.Conductor) (*eth.ExecutionPayloadEnvelope, error) {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return leader.Escape().RpcAPI().LatestUnsafePayload(callCtx)
}

type ethBlockByNumber interface {
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)
}

type ethLiveClient interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
}

// assertSequencerLive asserts the cluster is left with a working sequencer after the reorg
// stabilizes: some conductor is leader + healthy + actively sequencing, and chain B's unsafe
// head keeps advancing. Convergence alone does not prove this — if a leadership transfer
// stalls (e.g. on a head mismatch during the reorg) every EL can still converge on the
// post-reorg chain via gossip/derivation while no conductor is active, halting block
// production. Without this check that wedge passes the convergence assertions silently.
func assertSequencerLive(t devtest.T, conductors dsl.ConductorSet, elClient ethLiveClient) {
	ctx := t.Ctx()
	require.Eventually(t, func() bool {
		for _, c := range conductors {
			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			leader, lerr := c.Escape().RpcAPI().Leader(callCtx)
			healthy, herr := c.Escape().RpcAPI().SequencerHealthy(callCtx)
			active, aerr := c.Escape().RpcAPI().Active(callCtx)
			cancel()
			if lerr == nil && herr == nil && aerr == nil && leader && healthy && active {
				return true
			}
		}
		return false
	}, 90*time.Second, time.Second,
		"after the reorg a conductor must be leader+healthy+active (a live sequencer)")

	start, err := elClient.InfoByLabel(ctx, eth.Unsafe)
	require.NoError(t, err, "read chain B unsafe head for liveness baseline")
	const minAdvance = 3
	target := start.NumberU64() + minAdvance
	require.Eventually(t, func() bool {
		head, err := elClient.InfoByLabel(ctx, eth.Unsafe)
		return err == nil && head.NumberU64() >= target
	}, 60*time.Second, time.Second,
		"chain B must keep producing blocks after the reorg (unsafe head must advance)")
}

// runDeepReorgRecovery drives a depth-block deep reorg on chain B and asserts the conductor
// cluster recovers: some node observes and commits the post-reorg head into the FSM, every
// chain-B EL converges on the replacement, AND the cluster is left with a live sequencer. The
// caller sets the health-check config via preset options, which determines whether the
// unsafe-staleness check trips during the reorg (and thus whether the FSM-commit vs
// unhealthy-driven-transfer race is exercised). Per-conductor reorg metrics are logged so a
// repeated stress run can tell which node committed (the initial leader = race won; a
// post-transfer leader = race window entered but self-healed).
func runDeepReorgRecovery(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors, depth uint64) {
	conductors := conductorsForChainB(t, sys)
	elB := sys.L2ELB.Escape().EthClient()

	leaderBefore := findLeader(t, conductors)
	t.Logger().Info("chain B leader before deep reorg", "leader", leaderBefore.String(), "depth", depth)

	// Watch leadership/health across the whole reorg window.
	stopWatch := watchClusterHealth(t, conductors, leaderBefore)

	trigger := triggerInvalidMessageReorgOnBWithDepth(t, sys, depth)

	// Some node observed the deep reorg notification and committed the post-reorg head. Sum
	// across the cluster: under a tight health check leadership can churn mid-reorg, so the
	// committer is not necessarily the node that is leader when we scrape.
	require.Eventually(t, func() bool {
		var observed, committed float64
		for _, c := range conductors {
			observed += c.ScrapeCounter(reorgObservedMetric)
			committed += c.ScrapeCounter(reorgCommittedMetric)
		}
		return observed >= 1 && committed >= 1
	}, 90*time.Second, time.Second, "some conductor should observe and commit the post-reorg head")

	// The leader's FSM head converges to the canonical post-reorg chain.
	assertLeaderHeadConverges(t, conductors, elB, trigger.invalidBlockNumber)

	// Every chain-B EL (VN + leader + candidates) converges on the replacement.
	assertAllChainBELsConverged(t, sys, trigger)

	// Liveness: the cluster must still have a healthy, active sequencer producing blocks.
	assertSequencerLive(t, conductors, elB)

	healthTrips, leaderChurned := stopWatch()
	for _, c := range conductors {
		t.Logger().Info("conductor reorg metrics",
			"conductor", c.String(),
			"was_initial_leader", c.String() == leaderBefore.String(),
			"observed", c.ScrapeCounter(reorgObservedMetric),
			"committed", c.ScrapeCounter(reorgCommittedMetric))
	}
	t.Logger().Info("deep reorg gating measurement",
		"depth", depth,
		"leader_health_tripped", healthTrips,
		"leadership_churned", leaderChurned,
		"initial_leader_committed", leaderBefore.ScrapeCounter(reorgCommittedMetric),
		"note", "with reorg-recovery ON the cluster should recover; health/leadership detail is in conductor logs")
}

// chainBELNodes returns every chain-B EL the reorg must converge across: the supernode
// validator-node EL plus every conductor-controlled sequencer EL (leader + candidates).
func chainBELNodes(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors) []*dsl.L2ELNode {
	chainB := sys.L2B.Escape().ChainID()
	vn := sys.SupernodeELs[chainB]
	t.Require().NotNil(vn, "missing supernode VN EL for chain B")
	seqELs := sys.SequencerELs[chainB]
	t.Require().NotEmpty(seqELs, "missing conductor sequencer ELs for chain B")
	names := make([]string, 0, len(seqELs))
	for name := range seqELs {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*dsl.L2ELNode, 0, len(seqELs)+1)
	out = append(out, vn)
	for _, name := range names {
		out = append(out, seqELs[name])
	}
	return out
}

// assertAllChainBELsConverged asserts every chain-B EL replaced the invalid block at the
// reorg height and converged on a single canonical hash — proving the reorg propagated to
// the supernode VN EL and every conductor-controlled sequencer EL, not just the leader's
// EL that assertLeaderHeadConverges checks. On timeout it logs each EL's block-at-height
// hash and current unsafe head so a holdout (stuck or diverged) is identified by name.
func assertAllChainBELsConverged(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors, trigger reorgTrigger) {
	ctx := t.Ctx()
	els := chainBELNodes(t, sys)
	deadline := time.Now().Add(90 * time.Second)
	var lastReport string
	nextLog := time.Now()
	for {
		converged, report := chainBELConvergence(ctx, els, trigger)
		if converged {
			t.Logger().Info("all chain-B ELs converged on the post-reorg replacement",
				"block", trigger.invalidBlockNumber, "el_count", len(els))
			return
		}
		lastReport = report
		if time.Now().After(nextLog) {
			t.Logger().Info("waiting for chain-B ELs to converge", "block", trigger.invalidBlockNumber, "state", report)
			nextLog = time.Now().Add(10 * time.Second)
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	t.Require().FailNowf("chain-B ELs did not converge",
		"every chain-B EL (VN + all conductor sequencer ELs) must converge on the replacement at block %d; last observation: %s",
		trigger.invalidBlockNumber, lastReport)
}

// chainBELConvergence reports whether every EL has replaced the invalid block at the reorg
// height with a single shared canonical hash. The returned string describes each EL's
// block-at-height hash and current unsafe head, so a failed poll names the holdout.
func chainBELConvergence(ctx context.Context, els []*dsl.L2ELNode, trigger reorgTrigger) (bool, string) {
	var canonical common.Hash
	converged := true
	var report string
	for i, el := range els {
		ref, err := el.Escape().EthClient().BlockRefByNumber(ctx, trigger.invalidBlockNumber)
		head, headErr := el.Escape().EthClient().InfoByLabel(ctx, eth.Unsafe)
		headStr := "?"
		if headErr == nil {
			headStr = head.Hash().TerminalString() + ":" + strconv.FormatUint(head.NumberU64(), 10)
		}
		switch {
		case err != nil:
			report += " " + el.String() + "(blk=err:" + err.Error() + " head=" + headStr + ")"
			converged = false
		case ref.Hash == trigger.invalidBlockHash:
			report += " " + el.String() + "(blk=INVALID head=" + headStr + ")"
			converged = false
		default:
			report += " " + el.String() + "(blk=" + ref.Hash.TerminalString() + " head=" + headStr + ")"
			if i == 0 {
				canonical = ref.Hash
			} else if ref.Hash != canonical {
				converged = false
			}
		}
	}
	return converged, report
}

// pickNonLeader returns a voter in the cluster other than the current leader.
func pickNonLeader(t devtest.T, leader *dsl.Conductor) *consensus.ServerInfo {
	leaderInfo := leader.FetchLeader()
	membership := leader.FetchClusterMembership()
	for i := range membership.Servers {
		if membership.Servers[i].ID != leaderInfo.ID && membership.Servers[i].Suffrage == consensus.Voter {
			return &membership.Servers[i]
		}
	}
	t.Require().FailNow("no non-leader voter found in cluster")
	return nil
}

// watchClusterHealth polls every conductor for sequencer health and the observed leader
// until the returned stop func is called, which reports whether any conductor went
// unhealthy and whether leadership moved away from the initial leader during the window.
// RPC errors are tolerated (not treated as trips) so the watcher never aborts the test.
func watchClusterHealth(t devtest.T, conductors dsl.ConductorSet, initialLeader *dsl.Conductor) func() (healthTripped bool, leaderChurned bool) {
	ctx, cancel := context.WithCancel(t.Ctx())
	initialLeaderID := initialLeader.FetchLeader().ID
	var healthTripped, leaderChurned atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, c := range conductors {
					callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
					healthy, err := c.Escape().RpcAPI().SequencerHealthy(callCtx)
					leaderInfo, lerr := c.Escape().RpcAPI().LeaderWithID(callCtx)
					callCancel()
					if err == nil && !healthy {
						healthTripped.Store(true)
					}
					if lerr == nil && leaderInfo != nil && leaderInfo.ID != "" && leaderInfo.ID != initialLeaderID {
						leaderChurned.Store(true)
					}
				}
			}
		}
	}()
	return func() (bool, bool) {
		cancel()
		<-done
		return healthTripped.Load(), leaderChurned.Load()
	}
}

// TestConductorReorgRecovery covers the flag-ON path on a 3-conductor-per-chain cluster:
//   - Scenario A: a real op-reth reorg notification drives the leader to commit the
//     post-reorg head, and it converges cluster-wide.
//   - Scenario C: only the leader writes the reorg commit (followers stay at 0).
//   - Scenario B: the post-reorg Raft state survives a leadership transfer.
func TestConductorReorgRecovery(gt *testing.T) {
	t := devtest.SerialT(gt)
	requireOpReth(t)

	// Use the default lenient health-check intervals. The synthetic devstack genesis sits
	// minutes behind wallclock at startup, so a tight unsafe-staleness interval would trip
	// every conductor unhealthy at bring-up and churn leadership continuously — which, with
	// reorg-recovery driving sequencing, prevents the supernode reorg rewind from
	// converging. With lenient intervals the leader stays stable and the Phase 4 gating
	// measurement below records whether a realistic reorg still trips it.
	sys := presets.NewTwoL2SupernodeInteropWithConductors(t, 0,
		presets.WithConductorReorgRecovery(),
	)
	conductors := conductorsForChainB(t, sys)
	elB := sys.L2ELB.Escape().EthClient()

	leaderBefore := findLeader(t, conductors)
	t.Logger().Info("chain B leader before reorg", "leader", leaderBefore.String())

	// --- Phase 4 gating measurement: watch for leader health trips / churn during the
	// reorg window. A transfer is not itself a failure; convergence is the success
	// criterion. The specific health error (ErrSequencerNotHealthy vs
	// ErrSequencerConnectionDown) is only visible in conductor logs.
	stopWatch := watchClusterHealth(t, conductors, leaderBefore)

	trigger := triggerInvalidMessageReorgOnB(t, sys)

	// Scenario A: the subscriber observed a real reth reorg notification, and the leader
	// committed the post-reorg head.
	leader := findLeader(t, conductors)
	require.Eventually(t, func() bool {
		return leader.ScrapeCounter(reorgObservedMetric) >= 1
	}, 60*time.Second, time.Second, "leader should observe at least one reth reorg notification")
	require.GreaterOrEqual(t, leader.ScrapeCounter(reorgCommittedMetric), float64(1),
		"leader should commit at least one post-reorg head")

	// Scenario A (cont.): the leader's FSM head converges to the canonical post-reorg
	// chain (Raft replicates the committed entry to followers; Scenario B confirms a
	// former follower has it after a transfer).
	assertLeaderHeadConverges(t, conductors, elB, trigger.invalidBlockNumber)

	// Scenario A (cont.): the reorg must propagate to every chain-B EL — the supernode
	// VN EL and every conductor-controlled sequencer EL (leader + candidates) — not just
	// the leader's EL. All converge on the same canonical replacement block.
	assertAllChainBELsConverged(t, sys, trigger)

	// Scenario C: only the leader writes — every non-leader's committed count stays 0.
	for _, c := range conductors {
		if c.IsLeader() {
			continue
		}
		require.Equal(t, float64(0), c.ScrapeCounter(reorgCommittedMetric),
			"non-leader conductor %s must not commit reorg heads", c.String())
	}

	healthTrips, leaderChurned := stopWatch()
	t.Logger().Info("Phase 4 gating measurement",
		"leader_health_tripped", healthTrips,
		"leadership_churned", leaderChurned,
		"note", "convergence succeeded regardless; specific health error (if any) is in conductor logs")

	// Scenario B: leadership transfer preserves the post-reorg Raft state.
	currentLeader := findLeader(t, conductors)
	target := pickNonLeader(t, currentLeader)
	t.Logger().Info("transferring leadership", "from", currentLeader.String(), "to", target.ID)
	currentLeader.TransferLeadershipTo(*target)

	var newLeader *dsl.Conductor
	require.Eventually(t, func() bool {
		for _, c := range conductors {
			if c.IsLeader() && c.FetchLeader().ID == target.ID {
				newLeader = c
				return true
			}
		}
		return false
	}, 30*time.Second, 500*time.Millisecond, "leadership should transfer to the target server")

	head := newLeader.FetchLatestUnsafePayload()
	num := uint64(head.ExecutionPayload.BlockNumber)
	require.GreaterOrEqual(t, num, trigger.invalidBlockNumber,
		"new leader's FSM head should be at or beyond the reorg height")
	ref, err := elB.BlockRefByNumber(t.Ctx(), num)
	require.NoError(t, err, "new leader's recorded head should exist on the canonical chain")
	require.Equal(t, head.ExecutionPayload.BlockHash, ref.Hash,
		"new leader's recorded head should be canonical — Raft state survived the transfer")
}

// TestConductorDeepReorgRecovery exercises reorg-recovery against a deep reorg: the invalid
// executing message is buried under deepReorgDepth descendants (built while interop is
// paused) before interop resumes, so the invalidation rewinds a deep span rather than a
// single same-height block. With reorg-recovery ON, the leader must observe and commit the
// post-reorg head, its FSM head must converge to the canonical post-reorg chain, every
// chain-B EL must converge on the replacement, and leadership must stay stable.
//
// Slow: L2 blocks are produced in real time, so extending the chain by deepReorgDepth
// takes minutes.
func TestConductorDeepReorgRecovery(gt *testing.T) {
	t := devtest.SerialT(gt)
	requireOpReth(t)

	// deepReorgDepth is the number of blocks the chain is extended past the invalid block
	// before interop resumes, i.e. the depth of the resulting reorg. Tune as needed.
	const deepReorgDepth = 100

	sys := presets.NewTwoL2SupernodeInteropWithConductors(t, 0,
		presets.WithConductorReorgRecovery(),
	)
	runDeepReorgRecovery(t, sys, deepReorgDepth)
}

// TestConductorDeepReorgRecoveryTightHealth runs deep-reorg recovery under an aggressive but
// valid health check: a 1s poll cadence with a 4s unsafe-staleness window (one notch tighter
// than production's 5s, still safely above the 2s block time so steady-state production never
// trips). Unlike TestConductorDeepReorgRecovery's lenient default (UnsafeInterval 3600, which
// never trips and so never transfers leadership), the deep rewind here makes the unsafe head
// stale enough to trip the staleness check and force a leadership transfer mid-reorg. That
// exercises the safety-critical race between the reorg-recovery FSM commit and the
// unhealthy-driven leadership transfer: if leadership moves away before the post-reorg head is
// committed, a new leader could be left on a stale FSM head with no pending notification to
// re-commit. The liveness assertion in runDeepReorgRecovery catches a resulting wedge.
//
// Depth is OP_DEEP_REORG_DEPTH (default 25) — small enough to keep iterations short for
// repeated stress runs, large enough that the rewind (depth*blocktime ≈ 50s) far exceeds the
// 4s window and reliably trips health.
func TestConductorDeepReorgRecoveryTightHealth(gt *testing.T) {
	t := devtest.SerialT(gt)
	requireOpReth(t)

	depth := uint64(25)
	if v := os.Getenv("OP_DEEP_REORG_DEPTH"); v != "" {
		parsed, err := strconv.ParseUint(v, 10, 64)
		require.NoError(t, err, "OP_DEEP_REORG_DEPTH must be a uint")
		require.Positive(t, parsed, "OP_DEEP_REORG_DEPTH must be > 0")
		depth = parsed
	}

	sys := presets.NewTwoL2SupernodeInteropWithConductors(t, 0,
		presets.WithConductorReorgRecovery(),
		presets.WithConductorHealthCheck(1, 4, 1200),
	)
	runDeepReorgRecovery(t, sys, depth)
}

// TestConductorReorgRecoveryDisabled is Scenario D: with reorg-recovery OFF (the
// default), the same reorg leaves the subscriber fully dormant — no reth subscription is
// opened and reorgs_observed_count stays 0, proving the flag gates the entire change.
func TestConductorReorgRecoveryDisabled(gt *testing.T) {
	t := devtest.SerialT(gt)
	requireOpReth(t)

	sys := presets.NewTwoL2SupernodeInteropWithConductors(t, 0)
	conductors := conductorsForChainB(t, sys)

	_ = findLeader(t, conductors)
	trigger := triggerInvalidMessageReorgOnB(t, sys)
	t.Logger().Info("reorg triggered with reorg-recovery disabled", "invalid_block", trigger.invalidBlockNumber)

	// The replacement is verified by interop, confirming the chain made progress; the
	// recovery machinery simply never engaged.
	sys.Supernode.AwaitValidatedTimestamp(trigger.invalidBlockTimestamp)

	// No conductor opened a reth subscription: the observed counter is dormant at 0.
	for _, c := range conductors {
		require.Equal(t, float64(0), c.ScrapeCounter(reorgObservedMetric),
			"conductor %s must not observe reorgs when reorg-recovery is disabled", c.String())
		require.Equal(t, float64(0), c.ScrapeCounter(reorgCommittedMetric),
			"conductor %s must not commit reorg heads when reorg-recovery is disabled", c.String())
	}
}
