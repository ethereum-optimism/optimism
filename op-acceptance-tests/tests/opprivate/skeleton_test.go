package opprivate

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
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

// TestPrivateChainSafeAndFinalizedHeadsStayAtGenesis checks that the sequencer never derives
// from L1, so its safe and finalized heads stay at genesis while it builds blocks, also after a
// restart.
func TestPrivateChainSafeAndFinalizedHeadsStayAtGenesis(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	sys.SequencerCL.AdvancedUnsafe(5, 30)
	requireSafeAndFinalizedAtGenesis(t, sys)

	restart(sys.SequencerEL, sys.SequencerCL)
	sys.SequencerCL.AdvancedUnsafe(5, 30)
	requireSafeAndFinalizedAtGenesis(t, sys)
}

// TestPrivateChainSequencerResumesAfterRestart checks that the sequencer restarts from its
// database with no manual step, keeps building blocks, and the RPC node keeps up.
func TestPrivateChainSequencerResumesAfterRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	sys.SequencerCL.AdvancedUnsafe(5, 30)
	before := sys.SequencerCL.SyncStatus().UnsafeL2

	restart(sys.SequencerEL, sys.SequencerCL)

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
	sys.RPCNodeCL.Stop()
	sys.RPCNodeEL.Stop()
	sys.SequencerCL.AdvancedUnsafe(5, 30)
	sys.RPCNodeEL.Start()
	sys.RPCNodeCL.Start()

	target := sys.SequencerCL.SyncStatus().UnsafeL2
	sys.RPCNodeCL.ReachedUnsafe(target.Number, 30)
	t.Require().Equal(target.Hash, sys.RPCNodeEL.BlockRefByNumber(target.Number).Hash,
		"the RPC node has the sequencer's chain")
}

// TestPrivateChainConfigHoldsSettlementChainID checks that op-deployer wrote the same settlement
// chain ID into the ChainConfig predeploy and the rollup config.
func TestPrivateChainConfigHoldsSettlementChainID(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewOPPrivateSkeleton(t)

	chainConfig := bindings.NewBindings[chainConfig](
		bindings.WithClient(sys.SequencerEL.Escape().EthClient()),
		bindings.WithTo(predeploys.ChainConfigAddr),
		bindings.WithTest(t),
	)
	want := uint64(sysgo.OPPrivateSettlementChainID)
	t.Require().Equal(want, bigs.Uint64Strict(contract.Read(chainConfig.SettlementChainID())), "ChainConfig.settlementChainId()")
	t.Require().Equal(want, sys.L2Chain.Escape().RollupConfig().SettlementChainID, "rollup config settlement_chain_id")
}

type chainConfig struct {
	SettlementChainID func() bindings.TypedCall[*big.Int] `sol:"settlementChainId"`
}

func restart(el *dsl.L2ELNode, cl *dsl.L2CLNode) {
	cl.Stop()
	el.Stop()
	el.Start()
	cl.Start()
}

func requireSafeAndFinalizedAtGenesis(t devtest.T, sys *presets.OPPrivateSkeleton) {
	genesis := sys.L2Chain.Escape().RollupConfig().Genesis.L2
	status := sys.SequencerCL.SyncStatus()
	t.Require().Equal(eth.L1BlockRef{}, status.CurrentL1, "the sequencer derives from no L1 block")
	t.Require().Equal(genesis, status.SafeL2.ID(), "safe head stays at genesis")
	t.Require().Equal(genesis, status.FinalizedL2.ID(), "finalized head stays at genesis")
}
