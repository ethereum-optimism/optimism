package lokahi

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestMultiNodeSafeHeadsFollowLokahi checks a chain run by several nodes: with a valid cross-chain
// message, the sequencer, the source lokahi reads and the follower of each chain all make the
// message's blocks safe, take their safe label from lokahi rather than their own derivation, and
// agree on the safe chain.
func TestMultiNodeSafeHeadsFollowLokahi(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiMultiNode(t)
	nodes := slices.Concat(sys.ChainA.CLs(), sys.ChainB.CLs())

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	initMsg := alice.SendRandomInitMessage(rand.New(rand.NewSource(4321)), eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendExecMessage(initMsg)
	t.Logger().Info("Valid message sent", "init", initMsg.BlockID(), "exec", execMsg.BlockID())

	var checks []dsl.CheckFunc
	for _, node := range sys.ChainA.CLs() {
		checks = append(checks, node.ReachedRefFn(safety.CrossSafe, initMsg.BlockID(), 120))
	}
	for _, node := range sys.ChainB.CLs() {
		checks = append(checks, node.ReachedRefFn(safety.CrossSafe, execMsg.BlockID(), 120))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)

	// With lokahi down, every node keeps deriving, and no safe label moves.
	sys.Lokahi.Stop()
	checks = nil
	for _, node := range nodes {
		node.WaitForStall(safety.CrossSafe)
		checks = append(checks, node.AdvancedFn(safety.LocalSafe, 5, 60), node.NotAdvancedFn(safety.CrossSafe, 20))
	}
	dsl.CheckAll(t, checks...)
	held := logHeads(t, "lokahi stopped", nodes)

	// Once lokahi is back, every safe label passes the held one, and the nodes still agree.
	sys.Lokahi.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, safeReachesWithinLocalSafe(t, node, held[node]+3, 3*time.Minute))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)
	logHeads(t, "lokahi restarted", nodes)
}

// TestMultiNodeInvalidMessageReplacesBlock checks that a block with an invalid executing message
// is replaced on every node of its chain by the same deposits-only block, that the sequencer keeps
// building on the replacement, that no node holds the denied block after its replacement, also
// when it is handed to the node again as an unsafe payload, and that the follower follows the
// sequencer's gossip again once finality has passed the denied block.
//
// The denied block is posted to the nodes through admin_postUnsafePayload, which enters the path a
// gossiped block takes. Gossiping it again is not possible: the publisher's gossip cache refuses
// to publish the same message for two minutes, and a receiving kona-node rejects a gossiped block
// older than a minute.
func TestMultiNodeInvalidMessageReplacesBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiMultiNode(t)
	chainB := &sys.ChainB

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	initMsg := alice.SendRandomInitMessage(rand.New(rand.NewSource(8765)), eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	deniedPayload := chainB.SequencerEL.PayloadByNumber(denied.Number)
	t.Require().Equal(denied.Hash, deniedPayload.ExecutionPayload.BlockHash, "the denied block must be fetched while canonical")
	t.Logger().Info("Invalid executing message included", "block", denied, "tx", execMsg.TxHash())

	// Every node replaces the denied block with the same block, and makes it safe. From its
	// replacement on, the follower never holds the denied block again.
	replacement := awaitReplaced(t, chainB.SequencerEL, denied, 3*time.Minute)
	t.Require().Equal(replacement, awaitReplaced(t, chainB.FollowerEL, denied, 3*time.Minute),
		"the follower must replace the denied block with the sequencer's replacement")
	recipients := []*dsl.L2CLNode{chainB.Follower, chainB.Sequencer}
	heads := map[*dsl.L2CLNode]*headWatch{
		chainB.Follower:  watchHeads(t, chainB.FollowerEL),
		chainB.Sequencer: watchHeads(t, chainB.SequencerEL),
	}
	// A node that took the denied block back would fail its forkchoice update and reset, which
	// moves local-safe back for a moment, so the watch polls fast.
	localSafe := watchNoRegression(t, recipients, "local-safe", 10*time.Millisecond, func(status *eth.SyncStatus) uint64 {
		return status.LocalSafeL2.Number
	})
	var checks []dsl.CheckFunc
	for _, node := range chainB.CLs() {
		checks = append(checks, node.ReachedFn(safety.CrossSafe, denied.Number+1, 120))
	}
	dsl.CheckAll(t, checks...)
	for _, el := range chainB.ELs() {
		assertReplacement(t, el, denied, replacement, execMsg.TxHash())
	}
	t.Logger().Info("Denied block replaced on every node", "denied", denied, "replacement", replacement)

	// The sequencer keeps building on the replacement. Handed the denied block again while it is
	// above the finalized head, when no node imports unsafe blocks, neither node takes it back.
	chainB.Sequencer.Advanced(safety.LocalUnsafe, 5, 30)
	for _, node := range recipients {
		node.PostUnsafePayload(deniedPayload)
	}

	// Once finality passes the denied height, the follower follows the sequencer's gossip again,
	// and the denied block handed to it once more, now below local-safe, is still not taken back.
	chainB.Follower.Reached(safety.Finalized, denied.Number, 120)
	awaitFollowsGossip(t, chainB.Follower, chainB.Sequencer, chainB.SequencerEL, 2*time.Minute)
	for _, node := range recipients {
		node.PostUnsafePayload(deniedPayload)
	}
	dsl.CheckAll(t,
		chainB.Sequencer.AdvancedFn(safety.LocalUnsafe, 3, 30),
		chainB.Follower.AdvancedFn(safety.LocalSafe, 3, 60),
		chainB.Sequencer.AdvancedFn(safety.LocalSafe, 3, 60),
	)
	awaitFollowsGossip(t, chainB.Follower, chainB.Sequencer, chainB.SequencerEL, 2*time.Minute)

	errs := []error{localSafe.stop()}
	for node, watch := range heads {
		for _, head := range watch.seen() {
			if head == denied {
				errs = append(errs, fmt.Errorf("%s held the denied block %s after its replacement", node, denied))
			}
		}
	}
	t.Require().NoError(errors.Join(errs...))
	for _, el := range chainB.ELs() {
		assertReplacement(t, el, denied, replacement, execMsg.TxHash())
	}
	logHeads(t, "denied block posted again", chainB.CLs())
}

// awaitFollowsGossip waits until follower's unsafe head is a block of the sequencer's chain above
// follower's local-safe head, and at most two blocks behind the sequencer's unsafe head: a block
// the follower has from gossip, not from deriving it.
func awaitFollowsGossip(t devtest.T, follower, sequencer *dsl.L2CLNode, sequencerEL *dsl.L2ELNode, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
	defer cancel()
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		status, err := follower.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			return false, nil
		}
		head, err := sequencer.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			return false, nil
		}
		unsafe := status.UnsafeL2
		t.Logger().Info("Waiting for the follower to follow gossip", "follower_unsafe", unsafe.ID(),
			"follower_local_safe", status.LocalSafeL2.Number, "sequencer_unsafe", head.UnsafeL2.ID())
		if unsafe.Number <= status.LocalSafeL2.Number || unsafe.Number+2 < head.UnsafeL2.Number {
			return false, nil
		}
		block, err := sequencerEL.Escape().EthClient().BlockRefByNumber(ctx, unsafe.Number)
		return err == nil && block.Hash == unsafe.Hash, nil
	})
	t.Require().NoError(err, "%s must follow %s's unsafe blocks over gossip", follower, sequencer)
}

// TestMultiNodeSequencerRestartWhileLokahiDown checks that a sequencer restarted while lokahi is
// unreachable keeps producing unsafe blocks, which its follower imports, although it derives
// nothing until it hears from lokahi, and that no safe label moves up until lokahi is back. The
// restarted sequencer does not keep its safe label: it takes its finalized head, as op-node falls
// back to its finalized head for a verified safe label it can neither ask for nor has cached.
func TestMultiNodeSequencerRestartWhileLokahiDown(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiMultiNode(t)
	nodes := slices.Concat(sys.ChainA.CLs(), sys.ChainB.CLs())
	chainA := &sys.ChainA

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 3, 120))
	}
	dsl.CheckAll(t, checks...)

	sys.Lokahi.Stop()
	for _, node := range nodes {
		node.WaitForStall(safety.CrossSafe)
	}
	held := logHeads(t, "lokahi stopped", nodes)
	running := slices.DeleteFunc(slices.Clone(nodes), func(node *dsl.L2CLNode) bool { return node == chainA.Sequencer })
	safe := watchNoRegression(t, running, "safe", 100*time.Millisecond, func(status *eth.SyncStatus) uint64 {
		return status.SafeL2.Number
	})

	chainA.Sequencer.Stop()
	chainA.Sequencer.Start()
	dsl.CheckAll(t,
		chainA.Sequencer.AdvancedFn(safety.LocalUnsafe, 10, 60),
		chainA.Follower.AdvancedFn(safety.LocalUnsafe, 10, 60),
		sys.ChainB.Sequencer.AdvancedFn(safety.LocalUnsafe, 10, 60),
	)
	chainA.Follower.MatchedUnsafe(chainA.Sequencer, 30)
	logHeads(t, "sequencer restarted", nodes)
	for _, node := range running {
		t.Require().Equalf(held[node], node.SyncStatus().SafeL2.Number,
			"%s must hold its safe label while lokahi is down", node)
	}
	restarted := chainA.Sequencer.SyncStatus()
	t.Require().LessOrEqualf(restarted.SafeL2.Number, restarted.FinalizedL2.Number,
		"the restarted sequencer must not take a safe label above its finalized head without lokahi")

	sys.Lokahi.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, safeReachesWithinLocalSafe(t, node, held[node]+3, 3*time.Minute))
	}
	dsl.CheckAll(t, checks...)
	t.Require().NoError(safe.stop())
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)
	logHeads(t, "lokahi restarted", nodes)
}

// TestMultiNodeSourceOutage checks that with the node lokahi reads one chain from stopped, lokahi
// waits, so no safe label of either chain moves, while both chains' sequencers and followers keep
// their unsafe chains moving, and that every node recovers once the source is back.
func TestMultiNodeSourceOutage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiMultiNode(t)
	nodes := slices.Concat(sys.ChainA.CLs(), sys.ChainB.CLs())
	running := []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainA.Follower,
		sys.ChainB.Sequencer, sys.ChainB.Source, sys.ChainB.Follower}

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 3, 120))
	}
	dsl.CheckAll(t, checks...)

	sys.ChainA.Source.Stop()
	for _, node := range running {
		node.WaitForStall(safety.CrossSafe)
	}
	held := logHeads(t, "source stopped", running)
	verified := verifiedHeads(t, sys.Lokahi, []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainB.Sequencer})
	checks = nil
	for _, node := range []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainA.Follower, sys.ChainB.Sequencer, sys.ChainB.Follower} {
		checks = append(checks, node.AdvancedFn(safety.LocalUnsafe, 10, 60))
	}
	for _, node := range running {
		checks = append(checks, node.NotAdvancedFn(safety.CrossSafe, 15))
	}
	dsl.CheckAll(t, checks...)
	for node, head := range verifiedHeads(t, sys.Lokahi, []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainB.Sequencer}) {
		t.Require().Equalf(verified[node], head, "lokahi must wait for chain %s's source", node.ChainID())
	}
	for node, safeHead := range held {
		t.Require().Equalf(safeHead, node.SyncStatus().SafeL2.Number, "%s must hold its safe label", node)
	}

	sys.ChainA.Source.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 180))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)
	logHeads(t, "source restarted", nodes)
}

// assertSafeHeadsAgree checks that the safe heads of chain's nodes are on one chain, with the
// same block at the lowest of them on each node's execution layer, and that none is above lokahi's
// verified head. lokahi must run.
func assertSafeHeadsAgree(t devtest.T, lokahi *sysgo.Lokahi, chain *presets.LokahiChainNodes) {
	nodes := chain.CLs()
	safe := make([]uint64, len(nodes))
	for i, node := range nodes {
		safe[i] = node.SyncStatus().SafeL2.Number
	}
	// Read after the safe heads: lokahi's verified head only moves forward.
	verified := verifiedHeads(t, lokahi, []*dsl.L2CLNode{chain.Sequencer})[chain.Sequencer]
	lowest := slices.Min(safe)
	var blocks []eth.BlockID
	for i, el := range chain.ELs() {
		t.Require().LessOrEqualf(safe[i], verified.Block.Number,
			"%s's safe head must not be above lokahi's verified head %s", nodes[i], verified.Block)
		blocks = append(blocks, el.BlockRefByNumber(lowest).ID())
	}
	for _, block := range blocks[1:] {
		t.Require().Equalf(blocks[0], block, "chain %s's nodes disagree at their common safe height: %v",
			chain.Sequencer.ChainID(), blocks)
	}
	t.Logger().Info("Safe heads agree", "chain", chain.Sequencer.ChainID(), "common", blocks[0], "verified", verified.Block)
}

// watchNoRegression polls nodes' sync status every interval until stopped, and records the first
// time the label read by get moves back.
func watchNoRegression(t devtest.T, nodes []*dsl.L2CLNode, label string, interval time.Duration,
	get func(*eth.SyncStatus) uint64) *regressionWatch {
	ctx, cancel := context.WithCancel(t.Ctx())
	w := &regressionWatch{cancel: cancel}
	for _, node := range nodes {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			var last uint64
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				status, err := node.Escape().RollupAPI().SyncStatus(ctx)
				if err != nil {
					continue
				}
				current := get(status)
				if current < last {
					t.Logger().Warn("Label moved back", "node", node.String(), "label", label, "from", last, "to", current)
					w.fail(fmt.Errorf("%s: %s head moved back from %d to %d", node, label, last, current))
				}
				last = current
			}
		}()
	}
	return w
}

// regressionWatch is a watch started by [watchNoRegression].
type regressionWatch struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	err    error
}

func (w *regressionWatch) fail(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err == nil {
		w.err = err
	}
}

// stop ends the watch and returns the first regression seen, if any.
func (w *regressionWatch) stop() error {
	w.cancel()
	w.wg.Wait()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
