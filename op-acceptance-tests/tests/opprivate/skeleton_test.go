package opprivate

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// A private chain sequences blocks with no L1 derivation, deposits, withdrawals or sequencing
// window, and users transact from test accounts funded in genesis.

// TestPrivateChainTransfersLandInBlocks checks that a transfer sent to the sequencer lands in a
// block that the RPC node also serves.
func TestPrivateChainTransfersLandInBlocks(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.Wallet.NewEOA(sys.SequencerEL)
	receipt := alice.Transfer(bob.Address(), eth.OneHundredthEther).Included.Value()

	block := bigs.Uint64Strict(receipt.BlockNumber)
	sys.RPCNodeEL.Reached(eth.Unsafe, block, 30)
	t.Require().Equal(receipt.BlockHash, sys.RPCNodeEL.BlockRefByNumber(block).Hash,
		"the RPC node has the sequencer's block with the transfer")
	bob.AsEL(sys.RPCNodeEL).VerifyBalanceExact(eth.OneHundredthEther)
}

// TestPrivateChainRunsContracts checks that a contract deploys and runs on the private chain.
func TestPrivateChainRunsContracts(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	weth := alice.DeployWETH()
	alice.Transact(alice.Plan(), txplan.WithTo(&weth), txplan.WithValue(eth.OneTenthEther)).Included.Value()
	alice.WaitForTokenBalance(weth, eth.OneTenthEther)
}

// TestPrivateChainSafeAndFinalizedHeadsStayAtGenesis checks that the nodes never derive from
// L1, so their safe and finalized heads stay at genesis while the sequencer builds blocks past
// the sequencing window, also after a sequencer restart and an engine reset, which keep the
// blocks.
func TestPrivateChainSafeAndFinalizedHeadsStayAtGenesis(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)
	rollupCfg := sys.L2Chain.Escape().RollupConfig()

	// Past this L1 origin, a public chain's sync start would mark blocks safe.
	sys.SequencerEL.WaitL1OriginReached(eth.Unsafe, rollupCfg.Genesis.L1.Number+rollupCfg.SeqWindowSize+2, 120)
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	alice.Transfer(sys.Wallet.NewEOA(sys.SequencerEL).Address(), eth.OneHundredthEther).Included.Value()
	requireSafeAndFinalizedAtGenesis(t, sys)

	head := sys.SequencerCL.SyncStatus().UnsafeL2
	restart(sys.SequencerEL, sys.SequencerCL, sys.RPCNodeEL)
	requireResumedFrom(t, sys, head)

	head = sys.SequencerCL.SyncStatus().UnsafeL2
	t.Require().NoError(sys.SequencerCL.Escape().ClientRPC().CallContext(t.Ctx(), nil, "admin_resetDerivationPipeline"))
	requireResumedFrom(t, sys, head)
}

// TestPrivateChainSequencerResumesAfterRestart checks that the sequencer restarts from its
// database with no manual step, keeps building blocks, and the RPC node keeps up.
func TestPrivateChainSequencerResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	sys.SequencerCL.AdvancedUnsafe(5, 30)
	before := sys.SequencerCL.SyncStatus().UnsafeL2

	restart(sys.SequencerEL, sys.SequencerCL, sys.RPCNodeEL)

	sys.SequencerCL.ReachedUnsafe(before.Number+5, 30)
	t.Require().Equal(before.Hash, sys.SequencerEL.BlockRefByNumber(before.Number).Hash,
		"the sequencer resumed from its own chain")
	sys.RPCNodeCL.ReachedUnsafe(before.Number+5, 30)
}

// TestPrivateChainRPCNodeResumesAfterRestart checks that the RPC node restarts from its database
// with no manual step and catches up with the sequencer.
func TestPrivateChainRPCNodeResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	sys.RPCNodeCL.AdvancedUnsafe(5, 30)
	stop(sys.RPCNodeEL, sys.RPCNodeCL)
	sys.SequencerCL.AdvancedUnsafe(5, 30)
	start(sys.RPCNodeEL, sys.RPCNodeCL, sys.SequencerEL)

	target := sys.SequencerCL.SyncStatus().UnsafeL2
	sys.RPCNodeCL.ReachedUnsafe(target.Number, 30)
	t.Require().Equal(target.Hash, sys.RPCNodeEL.BlockRefByNumber(target.Number).Hash,
		"the RPC node has the sequencer's chain")
}

// restart restarts a node from its database and re-peers its EL with peer's: op-reth does not
// reconnect to its peers after a restart.
func restart(el *dsl.L2ELNode, cl *dsl.L2CLNode, peer *dsl.L2ELNode) {
	stop(el, cl)
	start(el, cl, peer)
}

func stop(el *dsl.L2ELNode, cl *dsl.L2CLNode) {
	cl.Stop()
	el.Stop()
}

func start(el *dsl.L2ELNode, cl *dsl.L2CLNode, peer *dsl.L2ELNode) {
	el.Start()
	el.PeerWith(peer)
	cl.Start()
}

// requireResumedFrom checks that the sequencer builds on head and the RPC node follows, with the
// safe and finalized heads still at genesis.
func requireResumedFrom(t devtest.T, sys *presets.OPPrivateSkeleton, head eth.L2BlockRef) {
	sys.SequencerCL.ReachedUnsafe(head.Number+5, 30)
	sys.RPCNodeCL.ReachedUnsafe(head.Number+5, 30)
	t.Require().Equal(head.Hash, sys.SequencerEL.BlockRefByNumber(head.Number).Hash, "the sequencer kept its blocks")
	requireSafeAndFinalizedAtGenesis(t, sys)
}

func requireSafeAndFinalizedAtGenesis(t devtest.T, sys *presets.OPPrivateSkeleton) {
	genesis := sys.L2Chain.Escape().RollupConfig().Genesis.L2
	t.Require().Equal(eth.L1BlockRef{}, sys.SequencerCL.SyncStatus().CurrentL1, "the sequencer derives from no L1 block")
	for _, cl := range []*dsl.L2CLNode{sys.SequencerCL, sys.RPCNodeCL} {
		status := cl.SyncStatus()
		t.Require().Equal(genesis, status.SafeL2.ID(), "%s safe head stays at genesis", cl)
		t.Require().Equal(genesis, status.FinalizedL2.ID(), "%s finalized head stays at genesis", cl)
	}
}
