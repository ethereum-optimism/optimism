package lokahi

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestInvalidMessageReplacesBlock checks that a block with an invalid executing message is
// replaced: lokahi denies it, the chain's kona-node replaces it with the deposits-only block at
// the same height, the safe heads of both chains pass that height, and both chains keep producing
// blocks. A restart of the node afterwards does not bring the denied block back, and while the
// restarted node cannot ask lokahi for the denied blocks it derives nothing.
func TestInvalidMessageReplacesBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	ctx := t.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	rng := rand.New(rand.NewSource(1234))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	deniedTime := sys.L2B.TimestampForBlockNum(denied.Number)
	t.Logger().Info("Invalid executing message included", "block", denied, "timestamp", deniedTime, "tx", execMsg.TxHash())

	replacement := awaitReplaced(t, sys.L2ELB, denied, 3*time.Minute)
	t.Logger().Info("Denied block replaced", "denied", denied, "replacement", replacement)

	// Both chains' safe heads pass the denied block's timestamp, which lokahi only verifies with
	// the replacement in place.
	chainAAtDenied, err := sys.L2A.Escape().RollupConfig().TargetBlockNumber(deniedTime)
	t.Require().NoError(err)
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.CrossSafe, denied.Number+1, 120),
		sys.L2ACL.ReachedFn(safety.CrossSafe, chainAAtDenied+1, 120),
	)
	assertReplacement(t, sys.L2ELB, denied, replacement, execMsg.TxHash())

	// Both chains keep producing blocks on top.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 5, 30),
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 3, 120),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 3, 120),
	)
	logHeads(t, "replaced", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})

	// Restart chain B's node while lokahi is down, so the restarted node does not know the denied
	// blocks. It re-derives from its sync start, which is at or below finality, but imports nothing
	// derived until it can ask lokahi again. lokahi restarts too, and still denies the block: the
	// denial is in its store.
	sys.Lokahi.Stop()
	heads := watchHeads(t, sys.L2ELB)
	statusBefore := sys.L2BCL.SyncStatus()
	t.Logger().Info("Restarting chain B's node", "denied", denied.Number,
		"finalized", statusBefore.FinalizedL2.Number, "local_safe", statusBefore.LocalSafeL2.Number,
		"rederives_denied_height", statusBefore.FinalizedL2.Number < denied.Number)
	sys.L2BCL.Stop()
	sys.L2BCL.Start()
	restarted := awaitReset(t, sys.L2BCL, statusBefore.UnsafeL2.Number, time.Minute)
	t.Logger().Info("Chain B's node restarted", "unsafe", restarted.UnsafeL2.Number,
		"local_safe", restarted.LocalSafeL2.Number, "finalized", restarted.FinalizedL2.Number)
	sys.L2BCL.NotAdvanced(safety.LocalSafe, 20)
	assertReplacement(t, sys.L2ELB, denied, replacement, execMsg.TxHash())
	// Restarting the node restarts its execution layer too when kona runs inside op-reth. The
	// head watch subscribes to the running execution layer before re-derivation can start, so it
	// sees every head re-derivation leads to. Nothing moves the head before: the stalled node
	// does not sequence either.
	heads.resubscribe(t)

	// Once lokahi is back the node derives again, past the replaced block, and keeps it.
	sys.Lokahi.Start()
	reachedKeepingReplacement(t, sys.L2BCL, sys.L2ELB, statusBefore.LocalSafeL2.Number+1, denied, 3*time.Minute)
	sys.L2BCL.Reached(safety.CrossSafe, statusBefore.LocalSafeL2.Number, 120)
	assertReplacement(t, sys.L2ELB, denied, replacement, execMsg.TxHash())
	// Re-deriving the chain did not reorg it, not even for a moment: the execution layer's head
	// only moved forward, and never was the denied block.
	seen := heads.seen()
	t.Require().NotEmpty(seen, "chain B must have announced heads")
	for i, head := range seen {
		t.Require().NotEqual(denied, head, "the restarted node must not bring the denied block back")
		if i > 0 {
			t.Require().GreaterOrEqual(head.Number, seen[i-1].Number,
				"the restarted node must not reorg chain B back from %s to %s", seen[i-1], head)
		}
	}
	t.Logger().Info("Chain B's head only moved forward", "heads", len(seen), "subscriptions", heads.subscribed(), "first", seen[0], "last", seen[len(seen)-1])
	logHeads(t, "restarted", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})

	receipt, err := sys.L2ELB.Escape().EthClient().TransactionReceipt(ctx, execMsg.TxHash())
	if err == nil {
		t.Logger().Info("The invalid executing message was included again later", "block", receipt.BlockNumber)
	} else {
		t.Logger().Info("The invalid executing message is not included anymore", "err", err)
	}
}

// awaitReplaced waits until el's block at denied's height is another block, and returns it.
func awaitReplaced(t devtest.T, el *dsl.L2ELNode, denied eth.BlockID, timeout time.Duration) eth.BlockID {
	ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
	defer cancel()
	var replacement eth.BlockID
	err := wait.For(ctx, time.Second, func() (bool, error) {
		block, err := el.Escape().EthClient().BlockRefByNumber(ctx, denied.Number)
		if err != nil {
			t.Logger().Warn("Block lookup failed; retrying", "number", denied.Number, "err", err)
			return false, nil
		}
		replacement = block.ID()
		return block.Hash != denied.Hash, nil
	})
	t.Require().NoError(err, "the denied block %s must be replaced", denied)
	return replacement
}

// assertReplacement checks that el's block at denied's height is replacement, a block with no
// user transactions.
func assertReplacement(t devtest.T, el *dsl.L2ELNode, denied, replacement eth.BlockID, deniedTx common.Hash) {
	block := el.BlockRefByNumber(denied.Number)
	t.Require().Equal(replacement, block.ID(), "the replacement of %s must stay canonical", denied)
	_, userTxs, err := el.Escape().L2EthClient().InfoAndUserTxsByNumber(t.Ctx(), denied.Number)
	t.Require().NoError(err)
	t.Require().Emptyf(userTxs, "the replacement of %s must have only deposits", denied)
	el.AssertUserTxNotInBlock(denied.Number, deniedTx)
}

// reachedKeepingReplacement waits until node's local-safe head reaches target, failing as soon as
// el's block at denied's height is the denied block again.
func reachedKeepingReplacement(t devtest.T, node *dsl.L2CLNode, el *dsl.L2ELNode, target uint64, denied eth.BlockID, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		block, err := el.Escape().EthClient().BlockRefByNumber(ctx, denied.Number)
		if err == nil && block.Hash == denied.Hash {
			return false, fmt.Errorf("%s brought back the denied block %s", node, denied)
		}
		status, err := node.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			return false, nil
		}
		t.Logger().Info("Waiting for the local-safe head", "local_safe", status.LocalSafeL2.Number, "target", target)
		return status.LocalSafeL2.Number >= target, nil
	})
	t.Require().NoError(err)
}

// headWatch records the heads an execution layer announces, across restarts of the execution
// layer: when the subscription ends it subscribes again.
type headWatch struct {
	mu    sync.Mutex
	heads []eth.BlockID
	// subscriptions counts the subscriptions made so far.
	subscriptions int
	// unsubscribe ends the current subscription.
	unsubscribe context.CancelFunc
}

// watchHeads records every head el announces until the test ends. It returns once the first
// subscription is up.
func watchHeads(t devtest.T, el *dsl.L2ELNode) *headWatch {
	ctx, cancel := context.WithCancel(t.Ctx())
	t.Cleanup(cancel)
	watch := &headWatch{unsubscribe: func() {}}
	go func() {
		for ctx.Err() == nil {
			subCtx, unsubscribe := context.WithCancel(ctx)
			watch.mu.Lock()
			watch.unsubscribe = unsubscribe
			watch.mu.Unlock()
			err := watch.follow(subCtx, el.Escape().UserRPC())
			unsubscribe()
			if err != nil && ctx.Err() == nil {
				t.Logger().Info("Head subscription ended; subscribing again", "el", el, "err", err)
			}
			select {
			case <-ctx.Done():
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	t.Require().Eventuallyf(func() bool { return watch.subscribed() > 0 }, time.Minute, 100*time.Millisecond,
		"subscribe to the heads of %s", el)
	return watch
}

// follow subscribes to the heads at url and records them until the subscription ends or ctx is
// done.
func (w *headWatch) follow(ctx context.Context, url string) error {
	client, err := rpc.DialContext(ctx, url)
	if err != nil {
		return err
	}
	defer client.Close()
	type header struct {
		Number hexutil.Uint64 `json:"number"`
		Hash   common.Hash    `json:"hash"`
	}
	announced := make(chan header, 1024)
	sub, err := client.EthSubscribe(ctx, announced, "newHeads")
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	w.mu.Lock()
	w.subscriptions++
	w.mu.Unlock()
	for {
		select {
		case head := <-announced:
			w.mu.Lock()
			w.heads = append(w.heads, eth.BlockID{Number: uint64(head.Number), Hash: head.Hash})
			w.mu.Unlock()
		case err := <-sub.Err():
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

// resubscribe ends the current subscription and waits for the next one, so that from then on the
// watch follows the execution layer as it runs now, also after a restart that the current
// subscription may not have noticed yet.
func (w *headWatch) resubscribe(t devtest.T) {
	w.mu.Lock()
	before, unsubscribe := w.subscriptions, w.unsubscribe
	w.mu.Unlock()
	unsubscribe()
	t.Require().Eventuallyf(func() bool { return w.subscribed() > before }, time.Minute, 100*time.Millisecond,
		"subscribe to the heads again")
}

// subscribed returns how many subscriptions the watch has made.
func (w *headWatch) subscribed() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.subscriptions
}

// seen returns the heads announced so far.
func (w *headWatch) seen() []eth.BlockID {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.heads)
}

// awaitReset waits until a restarted node has recovered its labels from the execution layer,
// which reports an unsafe head of at least unsafe.
func awaitReset(t devtest.T, node *dsl.L2CLNode, unsafe uint64, timeout time.Duration) *eth.SyncStatus {
	ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
	defer cancel()
	var status *eth.SyncStatus
	err := wait.For(ctx, time.Second, func() (bool, error) {
		var err error
		status, err = node.Escape().RollupAPI().SyncStatus(ctx)
		return err == nil && status.UnsafeL2.Number >= unsafe, nil
	})
	t.Require().NoError(err, fmt.Sprintf("%s must recover its labels after a restart", node))
	return status
}

// TestReplacementInvalidatesDependentBlock checks the cascade of a replacement: a block on chain A
// executes a message that a block on chain B initiated, and that block of chain B has an invalid
// executing message. Once lokahi denies chain B's block and chain B replaces it, the initiating log
// is gone, so lokahi denies chain A's block too, chain A replaces it, and both safe heads pass it.
func TestReplacementInvalidatesDependentBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	ctx := t.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	// With lokahi down nothing is denied, so chain A can execute a message from chain B's block
	// before that block is replaced.
	sys.Lokahi.Stop()
	rng := rand.New(rand.NewSource(5678))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	invalidMsg := bob.SendInvalidExecMessage(initMsg)
	deniedB := invalidMsg.BlockID()

	// The CrossL2Inbox log of chain B's invalid executing message is a message chain A can execute.
	dependent := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](alice.Plan())
	dependent.Content.DependOn(&invalidMsg.Tx.Result)
	dependent.Content.Fn(txintent.ExecuteIndexed(predeploys.CrossL2InboxAddr, &invalidMsg.Tx.Result, 0))
	dependentReceipt, err := dependent.PlannedTx.Included.Eval(ctx)
	t.Require().NoError(err, "chain A must include the dependent executing message")
	deniedA := eth.BlockID{Number: bigs.Uint64Strict(dependentReceipt.BlockNumber), Hash: dependentReceipt.BlockHash}
	t.Logger().Info("Included the invalid message and its dependent", "chain_b_block", deniedB, "chain_a_block", deniedA)

	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.LocalSafe, deniedB.Number, 120),
		sys.L2ACL.ReachedFn(safety.LocalSafe, deniedA.Number, 120),
	)
	sys.Lokahi.Start()

	replacementB := awaitReplaced(t, sys.L2ELB, deniedB, 3*time.Minute)
	replacementA := awaitReplaced(t, sys.L2ELA, deniedA, 3*time.Minute)
	t.Logger().Info("Both blocks replaced", "chain_b", replacementB, "chain_a", replacementA)
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.CrossSafe, deniedB.Number+1, 120),
		sys.L2ACL.ReachedFn(safety.CrossSafe, deniedA.Number+1, 120),
	)
	assertReplacement(t, sys.L2ELB, deniedB, replacementB, invalidMsg.TxHash())
	assertReplacement(t, sys.L2ELA, deniedA, replacementA, dependentReceipt.TxHash)
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 3, 120),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 3, 120),
	)
	logHeads(t, "cascade replaced", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})
}
