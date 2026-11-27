package lokahi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// l1SafeDistance is how far behind the head the test sequencer moves L1's safe head.
const l1SafeDistance = 10

// TestLokahiL1Reorg checks that lokahi and the nodes follow an L1 reorg that removes the L1
// blocks both chains derived their verified blocks from. lokahi's verified head moves back and
// reports no block of the old fork afterwards, no node's safe head points at a block off its
// canonical chain, finality never moves back, and the safe heads pass their old height again on
// the new fork.
func TestLokahiL1Reorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	els := map[*dsl.L2CLNode]*dsl.L2ELNode{sys.L2ACL: sys.L2ELA, sys.L2BCL: sys.L2ELB}
	l1ID := sys.L1Network.ChainID()

	// Verified history and finality first, so finality has something to lose.
	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 120), node.ReachedFn(safety.Finalized, 1, 120))
	}
	dsl.CheckAll(t, checks...)
	watch := watchLabels(t, sys.Lokahi, nodes, els)

	// Build L1 by hand until both chains' safe heads have an L1 origin on the hand-built blocks:
	// their batches are then in those blocks, and so are the L1 inclusions of lokahi's latest
	// results. The test sequencer can build on a parent only once it has moved L1's safe head,
	// ten blocks behind, below that parent.
	for sys.L1EL.BlockRefByLabel(eth.Unsafe).Number <= l1SafeDistance+1 {
		sys.L1Network.WaitForBlock()
	}
	sys.L1CL.Stop()
	startL1 := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	for _, node := range nodes {
		sys.TestSequencer.BuildL1UntilSafeOriginPast(t, l1ID, els[node], startL1.Number)
	}
	before := verifiedHeads(t, sys.Lokahi, nodes)
	safeBefore := make(map[*dsl.L2CLNode]eth.L2BlockRef, len(nodes))
	for _, node := range nodes {
		safeBefore[node] = node.HeadBlockRef(safety.CrossSafe)
		t.Logger().Info("Before the reorg", "chain", node.ChainID(), "safe", safeBefore[node],
			"safe_l1_origin", safeBefore[node].L1Origin, "verified", before[node], "start_l1", startL1)
	}

	divergence := sys.L1EL.BlockRefByNumber(startL1.Number + 1)
	t.Logger().Info("Reorging L1 from the first hand-built block", "divergence", divergence)
	reorgAt := time.Now()
	sys.TestSequencer.SequenceBlock(t, l1ID, divergence.ParentHash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 5)

	// lokahi's verified head moves back below where it was, then both safe heads pass their old
	// height on the new fork.
	for _, node := range nodes {
		awaitVerifiedBelow(t, sys.Lokahi, node, before[node].Block.Number, 5*time.Minute)
	}
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.ReachedFn(safety.CrossSafe, safeBefore[node].Number+1, 150))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "after the reorg", nodes)
	// Finality moves on, on the new fork.
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.ReachedFn(safety.Finalized, safeBefore[node].Number+1, 150))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "finalized past the reorg", nodes)

	record := watch.stop()
	errs := record.errs
	for _, node := range nodes {
		el := els[node]
		// The negative control: the reorg replaced the safe head's block, so a check that the
		// labels stay canonical has something to catch.
		if el.IsCanonical(safeBefore[node].ID()) {
			errs = append(errs, fmt.Errorf("chain %s: safe head %s before the reorg is still canonical; the reorg did not reach it",
				node.ChainID(), safeBefore[node]))
		}
		// After lokahi first answered below its pre-reorg head, it answers no block of the old
		// fork: every block it reports then is canonical on the new one.
		movedBack := false
		for _, answer := range record.verified[node] {
			if answer.at.Before(reorgAt) || answer.head.Source != "verified" {
				continue
			}
			if !movedBack && answer.head.Block.Number < before[node].Block.Number {
				movedBack = true
				t.Logger().Info("lokahi's verified head moved back", "chain", node.ChainID(),
					"from", before[node].Block, "to", answer.head.Block, "after", answer.at.Sub(reorgAt))
			}
			if movedBack && !el.IsCanonical(answer.head.Block) {
				errs = append(errs, fmt.Errorf("chain %s: lokahi answered %s of the old fork after moving back",
					node.ChainID(), answer.head.Block))
				break
			}
		}
		if !movedBack {
			errs = append(errs, fmt.Errorf("chain %s: lokahi's verified head never moved back below %s",
				node.ChainID(), before[node].Block))
		}
	}
	t.Require().NoError(errors.Join(errs...))
}

// awaitVerifiedBelow waits until lokahi answers a verified head for node's chain below number.
func awaitVerifiedBelow(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode, number uint64, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(t.Ctx(), timeout)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	chainID, ok := node.ChainID().Uint64()
	t.Require().True(ok, "chain ID fits in 64 bits")
	err = wait.For(ctx, 100*time.Millisecond, func() (bool, error) {
		head, err := verifiedHead(ctx, client, chainID)
		if err != nil {
			return false, nil
		}
		return head.Source == "verified" && head.Block.Number < number, nil
	})
	t.Require().NoError(err, "chain %s: lokahi's verified head did not move back below %d", node.ChainID(), number)
}

// labelWatch polls the nodes' labels and lokahi's verified heads until stopped.
type labelWatch struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	record labelRecord
}

// labelRecord is what a [labelWatch] saw.
type labelRecord struct {
	// errs are the violations: a safe head off the node's canonical chain, or finality moving
	// back.
	errs []error
	// verified is every head lokahi answered for each node's chain, in order.
	verified map[*dsl.L2CLNode][]timedHead
	// safe is every safe head each node reported, in order.
	safe map[*dsl.L2CLNode][]eth.L2BlockRef
}

type timedHead struct {
	at   time.Time
	head verifierHead
}

// watchLabels starts a [labelWatch]. A safe head that is not the block of its height on the
// node's execution layer is a violation only if the node still reports it a second later, so a
// sync status read just before a reorg's forkchoice update is not mistaken for one.
func watchLabels(t devtest.T, lokahi *sysgo.Lokahi, nodes []*dsl.L2CLNode, els map[*dsl.L2CLNode]*dsl.L2ELNode) *labelWatch {
	ctx, cancel := context.WithCancel(t.Ctx())
	w := &labelWatch{cancel: cancel, record: labelRecord{
		verified: make(map[*dsl.L2CLNode][]timedHead, len(nodes)),
		safe:     make(map[*dsl.L2CLNode][]eth.L2BlockRef, len(nodes)),
	}}
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	for _, node := range nodes {
		el := els[node]
		chainID, ok := node.ChainID().Uint64()
		t.Require().True(ok, "chain ID fits in 64 bits")
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			var finalized eth.L2BlockRef
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				if head, err := verifiedHead(ctx, client, chainID); err == nil {
					w.add(func(r *labelRecord) { r.verified[node] = append(r.verified[node], timedHead{time.Now(), head}) })
				}
				status, err := node.Escape().RollupAPI().SyncStatus(ctx)
				if err != nil {
					continue
				}
				w.add(func(r *labelRecord) { r.safe[node] = append(r.safe[node], status.SafeL2) })
				// The label is empty, with a zero hash, until the node's first reset.
				f := status.FinalizedL2
				if finalized.Hash != (common.Hash{}) && (f.Number < finalized.Number || (f.Number == finalized.Number && f.Hash != finalized.Hash)) {
					w.fail(fmt.Errorf("%s: finalized head moved from %s to %s", node, finalized, f))
				} else {
					finalized = f
				}
				if err := offCanonical(ctx, node, el, status.SafeL2); err != nil {
					t.Logger().Error("Safe head off the canonical chain", "node", node.String(), "err", err)
					w.fail(err)
				}
			}
		}()
	}
	return w
}

// offCanonical reports a safe head that is not the canonical block at its height on el and that
// node still reports a second later.
func offCanonical(ctx context.Context, node *dsl.L2CLNode, el *dsl.L2ELNode, safe eth.L2BlockRef) error {
	if safe.Hash == (common.Hash{}) {
		return nil // the empty label before the node's first reset
	}
	canonical := func() (bool, error) {
		ref, err := el.Escape().EthClient().BlockRefByNumber(ctx, safe.Number)
		if err != nil {
			return false, err
		}
		return ref.Hash == safe.Hash, nil
	}
	if ok, err := canonical(); err != nil || ok {
		return nil
	}
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(time.Second):
	}
	status, err := node.Escape().RollupAPI().SyncStatus(ctx)
	if err != nil || status.SafeL2 != safe {
		return nil
	}
	if ok, err := canonical(); err != nil || ok {
		return nil
	}
	return fmt.Errorf("%s: safe head %s is not the canonical block at its height", node, safe)
}

func (w *labelWatch) add(update func(*labelRecord)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	update(&w.record)
}

func (w *labelWatch) fail(err error) {
	w.add(func(r *labelRecord) {
		if len(r.errs) < 10 {
			r.errs = append(r.errs, err)
		}
	})
}

// stop ends the watch and returns what it saw.
func (w *labelWatch) stop() labelRecord {
	w.cancel()
	w.wg.Wait()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.record
}
