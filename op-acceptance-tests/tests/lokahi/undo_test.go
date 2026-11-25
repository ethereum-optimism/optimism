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
