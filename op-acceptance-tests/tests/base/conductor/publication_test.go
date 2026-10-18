package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestRejectedConductorCommitDoesNotPublish verifies that conductor acceptance
// precedes both CL head publication and EL canonicalization.
func TestRejectedConductorCommitDoesNotPublish(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalWithConductors(t)
	leader := sys.Conductors.AwaitOneActiveSequencer()
	follower := sys.Conductors.Without(leader)[0]
	leader.Sequencer().AdvancedUnsafe(2, 30)

	// Pause automation without stopping Raft or the commit RPC. Otherwise the
	// controllers would stop our manually activated follower during the check.
	for _, conductor := range sys.Conductors {
		t.Require().NoError(conductor.Escape().RpcAPI().Pause(t.Ctx()))
	}
	t.Require().Same(leader, sys.Conductors.AwaitLeader())
	follower.AwaitNotLeader()

	stoppedAt := leader.Sequencer().StopSequencer()
	settled := leader.Sequencer().HeadBlockRef(safety.LocalUnsafe)
	t.Require().Equal(stoppedAt, settled.Hash)
	followerCL := follower.Sequencer()
	followerEL := dsl.NewL2ELNode(followerCL.Escape().ELs()[0])
	followerCL.ReachedRef(safety.LocalUnsafe, settled.ID(), 30)
	followerEL.Reached(eth.Unsafe, settled.Number, 30)
	t.Require().Equal(settled.ID(), followerCL.HeadBlockRef(safety.LocalUnsafe).ID())
	t.Require().Equal(settled.ID(), followerEL.BlockRefByLabel(eth.Unsafe).ID())

	// This overrides the external leader check only. Unlike the node's local
	// override, it cannot bypass the actual follower's Raft commit rejection.
	follower.OverrideLeader(true)
	t.Require().True(follower.LeaderOverridden())
	err := follower.Escape().RpcAPI().CommitUnsafePayload(t.Ctx(), followerEL.PayloadByNumber(settled.Number))
	t.Require().ErrorContains(err, "not the leader")
	t.Require().NoError(followerCL.StartSequencerAt(stoppedAt))
	followerCL.AwaitSequencerActive()
	t.Require().Equal(settled.ID(), followerCL.HeadBlockRef(safety.LocalUnsafe).ID())
	t.Require().Equal(settled.ID(), followerEL.BlockRefByLabel(eth.Unsafe).ID())

	// Each helper samples for ten seconds, spanning multiple one-second commit
	// retry windows. Checking the attached EL also catches a hidden import that
	// leaves the published CL head unchanged.
	dsl.CheckAll(t,
		followerCL.NotAdvancedFn(safety.LocalUnsafe, 5),
		followerEL.NotAdvancedFn(eth.Unsafe, 5),
	)
	t.Require().True(followerCL.SequencerActive())
	t.Require().Equal(settled.ID(), followerCL.HeadBlockRef(safety.LocalUnsafe).ID())
	t.Require().Equal(settled.ID(), followerEL.BlockRefByLabel(eth.Unsafe).ID())
	t.Require().Equal(stoppedAt, followerCL.StopSequencer())

	follower.OverrideLeader(false)
	for _, conductor := range sys.Conductors {
		t.Require().NoError(conductor.Escape().RpcAPI().Resume(t.Ctx()))
	}
	recovered := sys.Conductors.AwaitOneActiveSequencer()
	recovered.Sequencer().AdvancedUnsafe(2, 30)
	followerCL.InSync(recovered.Sequencer(), safety.LocalUnsafe, 30)
}
