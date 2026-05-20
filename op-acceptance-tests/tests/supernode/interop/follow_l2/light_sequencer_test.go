package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestLightSequencerSupernodeDerivesSafeChain(gt *testing.T) {
	t := devtest.ParallelT(gt)

	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0)
	logger := sys.Log.With("Test", "TestLightSequencerSupernodeDerivesSafeChain")

	lightAActive, err := sys.L2ACL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A light CL sequencer status")
	t.Require().True(lightAActive, "chain A light CL should be the active sequencer")
	supernodeAActive, err := sys.L2ASupernodeCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A supernode sequencer status")
	t.Require().False(supernodeAActive, "chain A supernode route must not be sequencing")

	logger.Info("Wait for light CL sequencers to produce unsafe blocks")
	initialUnsafeA := sys.L2ACL.HeadBlockRef(types.LocalUnsafe)
	initialUnsafeB := sys.L2BCL.HeadBlockRef(types.LocalUnsafe)
	targetNumber := initialUnsafeA.Number
	if initialUnsafeB.Number > targetNumber {
		targetNumber = initialUnsafeB.Number
	}
	targetNumber += 3
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalUnsafe, targetNumber, 20),
		sys.L2BCL.ReachedFn(types.LocalUnsafe, targetNumber, 20),
	)

	targetA := sys.L2ELA.BlockRefByNumber(targetNumber)
	targetB := sys.L2ELB.BlockRefByNumber(targetNumber)
	t.Require().Equal(targetA.Time, targetB.Time, "target blocks should have matching timestamps")

	logger.Info("Wait for supernode routes to derive the light-sequenced blocks",
		"chainA_target", targetA,
		"chainB_target", targetB,
	)
	dsl.CheckAll(t,
		sys.L2ASupernodeCL.ReachedRefFn(types.LocalSafe, targetA.ID(), 60),
		sys.L2BSupernodeCL.ReachedRefFn(types.LocalSafe, targetB.ID(), 60),
		sys.L2ASupernodeCL.ReachedRefFn(types.CrossSafe, targetA.ID(), 60),
		sys.L2BSupernodeCL.ReachedRefFn(types.CrossSafe, targetB.ID(), 60),
	)

	logger.Info("Wait for supernode to validate light-sequenced timestamps",
		"timestamp", targetA.Time,
	)
	sys.Supernode.AwaitValidatedTimestamp(targetA.Time)
}
