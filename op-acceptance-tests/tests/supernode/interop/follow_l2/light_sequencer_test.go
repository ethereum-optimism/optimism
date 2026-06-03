package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestLightSequencerSupernodeDerivesSafeChain(gt *testing.T) {
	t := devtest.ParallelT(gt)

	// Skipped: an ELSync follow-mode sequencer cannot bootstrap from genesis. As the chain's sole
	// block producer it has no peer payload to initial-EL-sync from, so it deadlocks in willStartEL
	// and the chain never starts. TODO #21164.
	t.Skip("follow-mode light-sequencer ELSync genesis bootstrap unsupported; TODO #21164")

	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0)
	logger := sys.Log.With("Test", "TestLightSequencerSupernodeDerivesSafeChain")

	lightAActive, err := sys.L2ACL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A light CL sequencer status")
	t.Require().True(lightAActive, "chain A light CL should be the active sequencer")
	lightBActive, err := sys.L2BCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain B light CL sequencer status")
	t.Require().True(lightBActive, "chain B light CL should be the active sequencer")
	supernodeAActive, err := sys.L2ASupernodeCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A supernode sequencer status")
	t.Require().False(supernodeAActive, "chain A supernode route must not be sequencing")
	supernodeBActive, err := sys.L2BSupernodeCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain B supernode sequencer status")
	t.Require().False(supernodeBActive, "chain B supernode route must not be sequencing")

	cfgA := sys.L2A.Escape().RollupConfig()
	cfgB := sys.L2B.Escape().RollupConfig()
	t.Require().Equal(cfgA.Genesis.L2Time, cfgB.Genesis.L2Time, "target block number assumes matching genesis timestamps")
	t.Require().Equal(cfgA.BlockTime, cfgB.BlockTime, "target block number assumes matching L2 block times")

	logger.Info("Send transactions to light CL sequencers and wait for them to be mined")
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	aliceRecipient := sys.Wallet.NewEOA(sys.L2ELA)
	bobRecipient := sys.Wallet.NewEOA(sys.L2ELB)
	txA := txplan.NewPlannedTx(alice.PlanTransfer(aliceRecipient.Address(), eth.OneWei))
	txB := txplan.NewPlannedTx(bob.PlanTransfer(bobRecipient.Address(), eth.OneWei))
	_, err = txA.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "send tx to chain A light sequencer")
	_, err = txB.Submitted.Eval(t.Ctx())
	t.Require().NoError(err, "send tx to chain B light sequencer")
	receiptA, err := txA.Included.Eval(t.Ctx())
	t.Require().NoError(err, "wait for chain A sequencer tx to be mined")
	receiptB, err := txB.Included.Eval(t.Ctx())
	t.Require().NoError(err, "wait for chain B sequencer tx to be mined")

	targetNumber := bigs.Uint64Strict(receiptA.BlockNumber)
	if bigs.Uint64Strict(receiptB.BlockNumber) > targetNumber {
		targetNumber = bigs.Uint64Strict(receiptB.BlockNumber)
	}
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.LocalUnsafe, targetNumber, 20),
		sys.L2BCL.ReachedFn(safety.LocalUnsafe, targetNumber, 20),
	)

	targetA := sys.L2ELA.BlockRefByNumber(targetNumber)
	targetB := sys.L2ELB.BlockRefByNumber(targetNumber)
	t.Require().Equal(targetA.Time, targetB.Time, "target blocks should have matching timestamps")

	logger.Info("Wait for supernode routes to derive the light-sequenced blocks",
		"chainA_target", targetA,
		"chainB_target", targetB,
	)
	dsl.CheckAll(t,
		sys.L2ASupernodeCL.ReachedRefFn(safety.LocalSafe, targetA.ID(), 60),
		sys.L2BSupernodeCL.ReachedRefFn(safety.LocalSafe, targetB.ID(), 60),
		sys.L2ASupernodeCL.ReachedRefFn(safety.CrossSafe, targetA.ID(), 60),
		sys.L2BSupernodeCL.ReachedRefFn(safety.CrossSafe, targetB.ID(), 60),
	)

	logger.Info("Wait for supernode to validate light-sequenced timestamps",
		"timestamp", targetA.Time,
	)
	sys.Supernode.AwaitValidatedTimestamp(targetA.Time)
}
