package base

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type conductorWithInfo struct {
	stack.Conductor
	info consensus.ServerInfo
}

// TestLeadershipTransfer checks if the leadership transfer works correctly on the conductors
func TestLeadershipTransfer(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	logger := testlog.Logger(t, log.LevelInfo).With("Test", "TestLeadershipTransfer")
	tracer := t.Tracer()
	ctx := t.Ctx()
	logger.Info("Started L2 RPC connectivity test")

	ctx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test all L2 chains in the system
	for _, l2Chain := range sys.L2Networks() {
		_, span = tracer.Start(ctx, "test chain")
		defer span.End()

		require.NotEmpty(t, l2Chain.Escape().Conductors(), "no conductors found in the L2 chain")

		membership := dsl.NewConductor(l2Chain.Escape().Conductors()[0]).FetchClusterMembership()
		require.Equal(t, len(membership.Servers), len(l2Chain.Escape().Conductors()), "cluster membership does not match the number of conductors")

		idToConductor := make(map[string]conductorWithInfo)
		for _, conductor := range l2Chain.Escape().Conductors() {
			idToConductor[string(conductor.ID())] = conductorWithInfo{conductor, consensus.ServerInfo{}}
		}
		for _, memberInfo := range membership.Servers {
			conductor, ok := idToConductor[memberInfo.ID]
			require.True(t, ok, "unknown conductor in cluster membership", memberInfo.ID)
			conductor.info = memberInfo
			idToConductor[memberInfo.ID] = conductor
		}

		leaderInfo, err := l2Chain.Escape().Conductors()[0].RpcAPI().LeaderWithID(ctx)
		require.NoError(t, err, "failed to get current conductor info")

		leaderConductor := idToConductor[leaderInfo.ID]

		voters := []conductorWithInfo{leaderConductor}
		for _, member := range membership.Servers {
			if member.ID == leaderInfo.ID || member.Suffrage == consensus.Nonvoter {
				continue
			}

			voters = append(voters, idToConductor[member.ID])
		}

		if len(voters) == 1 {
			t.Skip("only one voter found in the cluster, skipping leadership transfer test")
			continue
		}

		t.Run(fmt.Sprintf("L2_Chain_%s", l2Chain.String()), func(tt devtest.T) {
			numOfLeadershipTransfers := len(voters)
			for i := 0; i < numOfLeadershipTransfers; i++ {
				oldLeader := voters[i]                 // no need for i%numOfLeadershipTransfers, since i is always less than numOfLeadershipTransfers
				newLeader := voters[(i+1)%len(voters)] // same as i+1, except it becomes 0 when i+1 is numOfLeadershipTransfers

				testTransferLeadershipAndCheck(t, voters, oldLeader, newLeader)
			}
		})
	}
}

// testTransferLeadershipAndCheck tests from one leader to another
func testTransferLeadershipAndCheck(t devtest.T, voters []conductorWithInfo, oldLeader, targetLeader conductorWithInfo) {
	if len(voters) == 0 {
		t.Skip("no voters found in the cluster, skipping leadership transfer test")
		return
	}

	if oldLeader.Conductor == nil || targetLeader.Conductor == nil {
		t.Skip("current or target leader is nil, skipping leadership transfer test")
		return
	}

	oldLeaderDsl := dsl.NewConductor(oldLeader.Conductor)
	targetLeaderDsl := dsl.NewConductor(targetLeader.Conductor)

	t.Run(fmt.Sprintf("Conductor_%s_to_%s", oldLeader.ID(), targetLeader.ID()), func(tt devtest.T) {
		// ensure that the current and target leader are healthy and unpaused before transferring leadership
		require.True(tt, oldLeaderDsl.FetchSequencerHealthy(), "current leader's sequencer is not healthy, id", oldLeader.ID())
		require.True(tt, targetLeaderDsl.FetchSequencerHealthy(), "target leader's sequencer is not healthy, id", targetLeader.ID())
		require.False(tt, oldLeaderDsl.FetchPaused(), "current leader's sequencer is paused, id", oldLeader.ID())
		require.False(tt, targetLeaderDsl.FetchPaused(), "target leader's sequencer is paused, id", targetLeader.ID())

		// ensure that the current leader is the leader before transferring leadership
		require.True(tt, oldLeaderDsl.IsLeader(), "current leader was not found to be the leader")
		require.False(tt, targetLeaderDsl.IsLeader(), "target leader was already found to be the leader")

		oldLeaderDsl.TransferLeadershipTo(targetLeader.info.ID, targetLeader.info.Addr)

		require.Eventually(
			tt,
			func() bool { return targetLeaderDsl.IsLeader() },
			5*time.Second, 1*time.Second, "target leader was not found to be the leader",
		)

		require.False(tt, oldLeaderDsl.IsLeader(), "old leader was still found to be the leader")

		// sometimes leadership transfer can cause a very brief period of unhealthiness,
		// but eventually, they should be healthy again
		require.Eventually(
			tt,
			func() bool { return oldLeaderDsl.FetchSequencerHealthy() && targetLeaderDsl.FetchSequencerHealthy() },
			3*time.Second, 1*time.Second, "at least one of the sequencers was found to be unhealthy",
		)
	})
}
