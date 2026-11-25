package lokahi

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
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

// TestLokahiL1ReorgOfADenial checks a chain that replaced a denied block when an L1 reorg removes
// the L1 blocks the denial and the replacement were derived from. lokahi rewinds the denial with
// its results, and the chain converges again on a block other than the denied one at its height:
// the safe heads pass it, and no safe head or verified head is ever the denied block or off the
// canonical chain. The test logs whether the chain derived the denied block again and lokahi
// denied it again, or the chain replaced it right away.
func TestLokahiL1ReorgOfADenial(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	d := denyOnHandBuiltL1(t, sys)
	d.reorg(t, sys)
	d.converge(t, sys)
}

// TestLokahiStaleDenialAcrossL1Reorg checks that a node does not act on a denial decided on L1
// blocks a reorg removed, although the denial is still in the last answer it has: lokahi is down
// across the reorg, so the node cannot hear that lokahi deleted it. The node derives the denied
// block again rather than keep the replacement, and once lokahi is back lokahi decides on the new
// L1 chain, denies the block again, and the chain converges.
func TestLokahiStaleDenialAcrossL1Reorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	d := denyOnHandBuiltL1(t, sys)
	sys.Lokahi.Stop()
	d.reorg(t, sys)

	// The batch of the denied block lands again on the new L1 chain, and chain B's node derives
	// the block from it instead of replacing it on the strength of the stale denial.
	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		block, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, d.denied.Number)
		if err != nil {
			return false, nil
		}
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			return false, nil
		}
		t.Logger().Info("Waiting for chain B to derive past the denied height", "block", block.ID(),
			"local_safe", status.LocalSafeL2.Number)
		return block.ID() == d.denied && status.LocalSafeL2.Number > d.denied.Number, nil
	})
	t.Require().NoError(err, "chain B's node must derive the denied block %s again while lokahi is down", d.denied)
	logHeads(t, "lokahi down after the reorg", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})

	sys.Lokahi.Start()
	d.converge(t, sys)
}

// denialOnHandBuiltL1 is a block chain B replaced after lokahi denied it, all on L1 blocks built
// by hand, and the watch over the labels since before.
type denialOnHandBuiltL1 struct {
	startL1     eth.L1BlockRef
	denied      eth.BlockID
	replacement eth.BlockID
	deniedTx    common.Hash
	safeBefore  eth.L2BlockRef
	watch       *labelWatch
}

// denyOnHandBuiltL1 stops L1's own block production, has chain B include an invalid executing
// message, and builds L1 by hand until lokahi denied the block and chain B replaced it and its
// safe head passed it. The invalid block's batch, its denial and its replacement's verification
// are then all on the hand-built L1 blocks.
func denyOnHandBuiltL1(t devtest.T, sys *presets.TwoL2Lokahi) *denialOnHandBuiltL1 {
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	els := map[*dsl.L2CLNode]*dsl.L2ELNode{sys.L2ACL: sys.L2ELA, sys.L2BCL: sys.L2ELB}
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	for sys.L1EL.BlockRefByLabel(eth.Unsafe).Number <= l1SafeDistance+1 {
		sys.L1Network.WaitForBlock()
	}
	d := &denialOnHandBuiltL1{watch: watchLabels(t, sys.Lokahi, nodes, els)}

	sys.L1CL.Stop()
	d.startL1 = sys.L1EL.BlockRefByLabel(eth.Unsafe)
	rng := rand.New(rand.NewSource(1234))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	d.denied, d.deniedTx = execMsg.BlockID(), execMsg.TxHash()
	t.Logger().Info("Invalid executing message included", "block", d.denied, "start_l1", d.startL1)

	l1 := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	t.Require().Eventually(func() bool {
		if err := l1.Next(t.Ctx()); err != nil {
			t.Logger().Warn("L1 block build failed, retrying", "err", err)
			return false
		}
		block, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(t.Ctx(), d.denied.Number)
		if err != nil || block.Hash == d.denied.Hash {
			return false
		}
		d.replacement = block.ID()
		return sys.L2BCL.HeadBlockRef(safety.CrossSafe).Number > d.denied.Number
	}, 3*time.Minute, 2*time.Second, fmt.Sprintf("chain B must replace %s and pass it", d.denied))
	// The replacement shows lokahi denied the block, so its answer is only logged.
	denials := deniedBlocks(t, sys.Lokahi, sys.L2BCL)
	d.safeBefore = sys.L2BCL.HeadBlockRef(safety.CrossSafe)
	head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Denied block replaced", "denied", d.denied, "replacement", d.replacement,
		"denials", denials, "safe", d.safeBefore, "l1_head", head, "hand_built", head.Number-d.startL1.Number)
	return d
}

// reorg reorgs L1 from the first hand-built block and lets L1 produce blocks again.
func (d *denialOnHandBuiltL1) reorg(t devtest.T, sys *presets.TwoL2Lokahi) {
	divergence := sys.L1EL.BlockRefByNumber(d.startL1.Number + 1)
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), divergence.ParentHash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 5)
}

// converge checks that both chains' safe heads pass their old height with a block other than
// the denied one at the denied height, and that no safe head or verified head of chain B was
// ever the denied block, nor any label off its chain.
func (d *denialOnHandBuiltL1) converge(t devtest.T, sys *presets.TwoL2Lokahi) {
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.CrossSafe, d.safeBefore.Number+1, 150),
		sys.L2ACL.ReachedFn(safety.CrossSafe, d.safeBefore.Number+1, 150))
	final := sys.L2ELB.BlockRefByNumber(d.denied.Number)
	t.Require().NotEqual(d.denied, final.ID(), "chain B must not keep the denied block")
	sys.L2ELB.AssertUserTxNotInBlock(d.denied.Number, d.deniedTx)
	denials := deniedBlocks(t, sys.Lokahi, sys.L2BCL)
	logHeads(t, "converged", []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL})
	t.Logger().Info("Converged after the reorg", "denied", d.denied, "replacement_before", d.replacement,
		"block_now", final.ID(), "same_replacement", final.ID() == d.replacement,
		"denied_again", slices.ContainsFunc(denials, func(r deniedBlock) bool { return r.Block == d.denied }),
		"denials_now", denials)

	record := d.watch.stop()
	errs := record.errs
	for _, safe := range record.safe[sys.L2BCL] {
		if safe.ID() == d.denied {
			errs = append(errs, fmt.Errorf("chain B's safe head was the denied block %s", d.denied))
			break
		}
	}
	for _, answer := range record.verified[sys.L2BCL] {
		if answer.head.Source == "verified" && answer.head.Block == d.denied {
			errs = append(errs, fmt.Errorf("lokahi verified the denied block %s", d.denied))
			break
		}
	}
	t.Require().NoError(errors.Join(errs...))
}

// deniedBlock is one entry of lokahi's answer to interop_deniedBlocks.
type deniedBlock struct {
	Block eth.BlockID `json:"block"`
	L1    eth.BlockID `json:"l1"`
}

// deniedBlocks asks lokahi for the denied blocks of node's chain.
func deniedBlocks(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode) []deniedBlock {
	ctx, cancel := context.WithTimeout(t.Ctx(), 30*time.Second)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	chainID, ok := node.ChainID().Uint64()
	t.Require().True(ok, "chain ID fits in 64 bits")
	var denied []deniedBlock
	t.Require().NoError(client.CallContext(ctx, &denied, "interop_deniedBlocks", chainID, 0), "interop_deniedBlocks")
	return denied
}
