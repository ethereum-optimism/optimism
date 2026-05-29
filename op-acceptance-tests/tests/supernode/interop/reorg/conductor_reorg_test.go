package reorg

import (
	"context"
	"math/rand"
	"os"
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
)

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

// triggerInvalidMessageReorgOnB forces a same-chain reorg on chain B by including an
// invalid executing message and resuming interop, which invalidates and replaces the
// offending block. It returns once the EL has replaced the block. Mirrors the trigger in
// invalid_message_reorg_test.go.
func triggerInvalidMessageReorgOnB(t devtest.T, sys *presets.TwoL2SupernodeInteropWithConductors) reorgTrigger {
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
		"block", invalidBlockNumber, "hash", invalidBlockHash, "timestamp", invalidBlockTimestamp)

	require.Eventually(t, func() bool {
		return sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	sys.Supernode.ResumeInterop()
	require.Eventually(t, func() bool {
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			return false
		}
		return currentBlock.Hash != invalidBlockHash
	}, 60*time.Second, time.Second, "the invalid block at the reorg height should be replaced at the EL")

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
		callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
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
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return leader.Escape().RpcAPI().LatestUnsafePayload(callCtx)
}

type ethBlockByNumber interface {
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)
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
					callCtx, callCancel := context.WithTimeout(ctx, 3*time.Second)
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
