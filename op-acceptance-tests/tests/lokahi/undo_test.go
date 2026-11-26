package lokahi

import (
	"context"
	"math/rand"
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

// TestLokahiDenialVoidedWhileNodeIsDown checks the undo of a denial decided on L1 blocks a reorg
// removes while the denied chain's node is down. The denied block's batch, and with it the
// replacement the node made, are on an L1 block the reorg keeps, so nothing of the node's own
// makes it derive the height again: only lokahi knows the denial is gone. lokahi deletes it with
// a new epoch; once the node is back it takes the epoch on by deriving again from its finalized
// head, so the denied block comes back and lokahi decides it again on the new L1 chain. Whatever
// chain B ends up with, a replacement at the height stands only next to a denial of the block that
// holds on the canonical L1 chain.
func TestLokahiDenialVoidedWhileNodeIsDown(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	for sys.L1EL.BlockRefByLabel(eth.Unsafe).Number <= l1SafeDistance+1 {
		sys.L1Network.WaitForBlock()
	}
	epochBefore := lokahiStatusOf(t, sys.Lokahi, sys.L2BCL).Epoch
	l1 := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	buildL1Until := func(done func() bool, msg string) {
		t.Require().Eventually(func() bool {
			if err := l1.Next(t.Ctx()); err != nil {
				t.Logger().Warn("L1 block build failed, retrying", "err", err)
				return false
			}
			return done()
		}, 3*time.Minute, 2*time.Second, msg)
	}

	// Chain A's batches wait, so lokahi can check chain B's block only once they land, on a later
	// L1 block than chain B's batch.
	batcherA := dsl.NewL2Batcher(sys.L2A.Escape().L2Batchers()[0])
	batcherB := dsl.NewL2Batcher(sys.L2B.Escape().L2Batchers()[0])
	sys.L1CL.Stop()
	batcherA.Stop()
	rng := rand.New(rand.NewSource(4321))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	t.Logger().Info("Invalid executing message included", "block", denied)
	buildL1Until(func() bool { return sys.L2BCL.HeadBlockRef(safety.LocalSafe).Number >= denied.Number },
		"chain B must derive the invalid block from its batch")
	// Chain B's blocks and batches stop there, so everything chain B builds and derives is on
	// L1 blocks the reorg keeps: no reorg handling of the node's own derives the height again.
	batcherB.Stop()
	sys.L2BCL.StopSequencer()
	kept := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Chain B's batch landed", "last_l1_the_reorg_keeps", kept)

	batcherA.Start()
	var replacement eth.BlockID
	buildL1Until(func() bool {
		block, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(t.Ctx(), denied.Number)
		if err != nil || block.Hash == denied.Hash {
			return false
		}
		replacement = block.ID()
		return sys.L2BCL.HeadBlockRef(safety.CrossSafe).Number >= denied.Number
	}, "lokahi must deny the invalid block once chain A's batches land, and verify chain B's replacement")
	status := lokahiStatusOf(t, sys.Lokahi, sys.L2BCL)
	t.Require().Contains(status.Denied, denied, "lokahi denied the invalid block")
	t.Require().Equal(epochBefore, status.Epoch, "no denial was deleted yet")
	safeBefore := map[*dsl.L2CLNode]eth.L2BlockRef{}
	for _, node := range nodes {
		safeBefore[node] = node.HeadBlockRef(safety.CrossSafe)
	}
	t.Logger().Info("Denied block replaced", "denied", denied, "replacement", replacement,
		"safe_b", safeBefore[sys.L2BCL], "l1_head", sys.L1EL.BlockRefByLabel(eth.Unsafe))

	// Node B is down while L1 reorgs out the blocks the denial was decided on, but keeps the
	// block with chain B's batch.
	sys.L2BCL.Stop()
	divergence := sys.L1EL.BlockRefByNumber(kept.Number + 1)
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), divergence.ParentHash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 5)
	awaitEpochChange(t, sys.Lokahi, sys.L2ACL, epochBefore)
	sys.L2BCL.Start()
	if active, err := sys.L2BCL.Escape().RollupAPI().SequencerActive(t.Ctx()); err != nil || !active {
		sys.L2BCL.StartSequencer()
	}
	batcherB.Start()

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.ReachedFn(safety.CrossSafe, safeBefore[node].Number+1, 150))
	}
	dsl.CheckAll(t, checks...)
	final := sys.L2ELB.BlockRefByNumber(denied.Number)
	status = lokahiStatusOf(t, sys.Lokahi, sys.L2BCL)
	logHeads(t, "converged", nodes)
	t.Logger().Info("Converged after the reorg", "denied", denied, "replacement_before", replacement,
		"block_now", final.ID(), "denials_now", status.Denied, "epoch", status.Epoch)
	if final.ID() != denied {
		t.Require().Contains(status.Denied, denied,
			"chain B holds %s at the denied height, but lokahi holds no denial of %s on the canonical L1 chain",
			final.ID(), denied)
	}
}

// awaitEpochChange waits until lokahi answers an epoch other than before for node's chain.
func awaitEpochChange(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode, before common.Hash) {
	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	chainID, ok := node.ChainID().Uint64()
	t.Require().True(ok, "chain ID fits in 64 bits")
	err = wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		status, err := chainStatus(ctx, client, chainID)
		return err == nil && status.Epoch != before, nil
	})
	t.Require().NoError(err, "lokahi must delete the reorged denial with a new epoch")
}

// TestLokahiEpochChangeRestoresBlocksBuiltOnAReplacement checks the undo of a replacement that
// chain B's sequencer kept building on. lokahi loses its store, so it stands behind no denial any
// more, and its new epoch makes chain B's node derive again from its finalized head: the denied
// block comes back at its height, the unsafe blocks built on the replacement go, and the sequencer
// goes on building on the restored block. lokahi is down meanwhile, so the restored chain stands
// until lokahi is back, verifies up to the height again, denies the block again, and the chain
// converges without stalling.
func TestLokahiEpochChangeRestoresBlocksBuiltOnAReplacement(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	rng := rand.New(rand.NewSource(2468))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	replacement := awaitReplaced(t, sys.L2ELB, denied, 3*time.Minute)
	awaitBuiltOn(t, sys.L2ELB, replacement, 3, "the sequencer must build on the replacement")
	safeBefore := sys.L2BCL.HeadBlockRef(safety.CrossSafe)
	epochBefore := lokahiStatusOf(t, sys.Lokahi, sys.L2BCL).Epoch
	t.Logger().Info("The sequencer builds on the replacement", "denied", denied, "replacement", replacement,
		"safe", safeBefore, "unsafe", sys.L2BCL.HeadBlockRef(safety.LocalUnsafe))

	// lokahi loses its store, and is stopped again as soon as chain B's node took the new epoch
	// on, so nothing denies the block while the node derives it again.
	sys.Lokahi.Stop()
	sys.Lokahi.WipeData()
	sys.Lokahi.Start()
	awaitEpochChange(t, sys.Lokahi, sys.L2BCL, epochBefore)
	epoch := lokahiStatusOf(t, sys.Lokahi, sys.L2BCL).Epoch
	awaitAppliedEpoch(t, sys.L2BCL, epoch)
	sys.Lokahi.Stop()

	restored := awaitBuiltOn(t, sys.L2ELB, denied, 3, "chain B must derive the denied block again and build on it")
	t.Logger().Info("The denied block is back and the sequencer builds on it", "denied", denied,
		"unsafe", restored, "safe", sys.L2BCL.HeadBlockRef(safety.CrossSafe))

	sys.Lokahi.Start()
	final := awaitReplaced(t, sys.L2ELB, denied, 5*time.Minute)
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.CrossSafe, safeBefore.Number+1, 150),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 5, 60),
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 3, 150),
	)
	status := lokahiStatusOf(t, sys.Lokahi, sys.L2BCL)
	t.Require().Contains(status.Denied, denied, "lokahi denies the block again on its new store")
	t.Require().Equal(epoch, status.Epoch)
	logHeads(t, "converged", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})
	t.Logger().Info("Converged after the undo", "denied", denied, "replacement_before", replacement,
		"replacement_now", final)
}

// awaitBuiltOn waits until el's chain holds block and n blocks on top of it, and returns the
// highest of them.
func awaitBuiltOn(t devtest.T, el *dsl.L2ELNode, block eth.BlockID, n uint64, msg string) eth.BlockID {
	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	defer cancel()
	var top eth.BlockID
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		parent := block
		for number := block.Number; number <= block.Number+n; number++ {
			ref, err := el.Escape().EthClient().BlockRefByNumber(ctx, number)
			if err != nil {
				return false, nil
			}
			if number == block.Number && ref.ID() != block || number > block.Number && ref.ParentHash != parent.Hash {
				return false, nil
			}
			parent, top = ref.ID(), ref.ID()
		}
		return true, nil
	})
	t.Require().NoError(err, msg)
	return top
}

// awaitAppliedEpoch waits until node derives under epoch.
func awaitAppliedEpoch(t devtest.T, node *dsl.L2CLNode, epoch common.Hash) {
	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, node.Escape().UserRPC())
	t.Require().NoError(err, "dial %s", node)
	defer client.Close()
	err = wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		var applied *common.Hash
		if err := client.CallContext(ctx, &applied, "interop_appliedEpoch"); err != nil {
			return false, nil
		}
		return applied != nil && *applied == epoch, nil
	})
	t.Require().NoError(err, "%s must take lokahi's epoch %s on", node, epoch)
}
