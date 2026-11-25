package lokahi

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
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
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	els := map[*dsl.L2CLNode]*dsl.L2ELNode{sys.L2ACL: sys.L2ELA, sys.L2BCL: sys.L2ELB}
	l1ID := sys.L1Network.ChainID()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	for sys.L1EL.BlockRefByLabel(eth.Unsafe).Number <= l1SafeDistance+1 {
		sys.L1Network.WaitForBlock()
	}
	watch := watchLabels(t, sys.Lokahi, nodes, els)

	// From here L1 is built by hand, so the invalid block's batch, its denial and its
	// replacement's verification are all on hand-built L1 blocks.
	sys.L1CL.Stop()
	startL1 := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	rng := rand.New(rand.NewSource(1234))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	t.Logger().Info("Invalid executing message included", "block", denied, "start_l1", startL1)

	l1 := sys.TestSequencer.Escape().ControlAPI(l1ID)
	var replacement eth.BlockID
	t.Require().Eventually(func() bool {
		if err := l1.Next(t.Ctx()); err != nil {
			t.Logger().Warn("L1 block build failed, retrying", "err", err)
			return false
		}
		block, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(t.Ctx(), denied.Number)
		if err != nil || block.Hash == denied.Hash {
			return false
		}
		replacement = block.ID()
		return sys.L2BCL.HeadBlockRef(safety.CrossSafe).Number > denied.Number
	}, 3*time.Minute, 2*time.Second, fmt.Sprintf("chain B must replace %s and pass it", denied))
	t.Require().Contains(deniedBlocks(t, sys.Lokahi, sys.L2BCL), denied, "lokahi denied the block")
	safeBefore := sys.L2BCL.HeadBlockRef(safety.CrossSafe)
	head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Denied block replaced", "denied", denied, "replacement", replacement,
		"safe", safeBefore, "l1_head", head, "hand_built", head.Number-startL1.Number)

	divergence := sys.L1EL.BlockRefByNumber(startL1.Number + 1)
	sys.TestSequencer.SequenceBlock(t, l1ID, divergence.ParentHash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 5)

	// The chain converges: chain B's safe head passes its old height, at a block other than the
	// denied one at the denied height.
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(safety.CrossSafe, safeBefore.Number+1, 150),
		sys.L2ACL.ReachedFn(safety.CrossSafe, safeBefore.Number+1, 150))
	final := sys.L2ELB.BlockRefByNumber(denied.Number)
	t.Require().NotEqual(denied, final.ID(), "chain B must not keep the denied block")
	sys.L2ELB.AssertUserTxNotInBlock(denied.Number, execMsg.TxHash())
	deniedAfter := deniedBlocks(t, sys.Lokahi, sys.L2BCL)
	logHeads(t, "after the reorg", nodes)
	t.Logger().Info("Converged after the reorg", "denied", denied, "replacement_before", replacement,
		"block_now", final.ID(), "same_replacement", final.ID() == replacement,
		"denied_again", slices.Contains(deniedAfter, denied), "denied_now", deniedAfter)

	record := watch.stop()
	errs := record.errs
	for _, safe := range record.safe[sys.L2BCL] {
		if safe.ID() == denied {
			errs = append(errs, fmt.Errorf("chain B's safe head was the denied block %s", denied))
			break
		}
	}
	for _, answer := range record.verified[sys.L2BCL] {
		if answer.head.Source == "verified" && answer.head.Block == denied {
			errs = append(errs, fmt.Errorf("lokahi verified the denied block %s", denied))
			break
		}
	}
	t.Require().NoError(errors.Join(errs...))
}

// deniedBlocks asks lokahi for the denied blocks of node's chain.
func deniedBlocks(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode) []eth.BlockID {
	ctx, cancel := context.WithTimeout(t.Ctx(), 10*time.Second)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	chainID, ok := node.ChainID().Uint64()
	t.Require().True(ok, "chain ID fits in 64 bits")
	var denied []eth.BlockID
	t.Require().NoError(client.CallContext(ctx, &denied, "interop_deniedBlocks", chainID, 0), "interop_deniedBlocks")
	return denied
}
