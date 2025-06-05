//go:build !ci

// use a tag prefixed with "!". Such tag ensures that the default behaviour of this test would be to be built/run even when the go toolchain (go test) doesn't specify any tag filter.
package conductor

import (
	"context"
	"fmt"
	"strings"
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
	*dsl.Conductor
	info consensus.ServerInfo
}

// TestConductorLeadershipTransfer checks if the leadership transfer works correctly on the conductors
func TestConductorLeadershipTransfer(gt *testing.T) {
	t := devtest.SerialT(gt)
	logger := testlog.Logger(t, log.LevelInfo).With("Test", "TestConductorLeadershipTransfer")

	sys := presets.NewMinimalWithConductors(t)
	tracer := t.Tracer()
	ctx := t.Ctx()
	logger.Info("Started Conductor Leadership Transfer test")

	ctx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test all L2 chains in the system
	for l2Chain, conductors := range sys.ConductorSets {
		chainId := l2Chain.String()

		_, span = tracer.Start(ctx, fmt.Sprintf("test chain %s", chainId))
		defer span.End()

		membership := conductors[0].FetchClusterMembership()
		require.Equal(t, len(membership.Servers), len(conductors), "cluster membership does not match the number of conductors", "chainId", chainId)

		idToConductor := make(map[string]conductorWithInfo)
		for _, conductor := range conductors {
			conductorId := strings.TrimPrefix(conductor.String(), stack.ConductorKind.String()+"-")
			idToConductor[conductorId] = conductorWithInfo{conductor, consensus.ServerInfo{}}
		}
		for _, memberInfo := range membership.Servers {
			conductor, ok := idToConductor[memberInfo.ID]
			require.True(t, ok, "unknown conductor in cluster membership", "unknown conductor id", memberInfo.ID, "chainId", chainId)
			conductor.info = memberInfo
			idToConductor[memberInfo.ID] = conductor
		}

		leaderInfo, err := conductors[0].Escape().RpcAPI().LeaderWithID(ctx)
		require.NoError(t, err, "failed to get current conductor info", "chainId", chainId)

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

		t.Run(fmt.Sprintf("L2_Chain_%s", chainId), func(tt devtest.T) {
			numOfLeadershipTransfers := len(voters)
			for i := 0; i < numOfLeadershipTransfers; i++ {
				// the modulo operation is used to wrap around the list of voters whenever i or i+1 becomes >= len(voters)
				oldLeaderIndex, newLeaderIndex := i%len(voters), (i+1)%len(voters)
				oldLeader, newLeader := voters[oldLeaderIndex], voters[newLeaderIndex]

				time.Sleep(3 * time.Second)

				testTransferLeadershipAndCheck(t, oldLeader, newLeader)
			}
		})
	}
}

// testTransferLeadershipAndCheck tests conductor's leadership transfer from one leader to another
func testTransferLeadershipAndCheck(t devtest.T, oldLeader, targetLeader conductorWithInfo) {

	t.Run(fmt.Sprintf("Conductor_%s_to_%s", oldLeader, targetLeader), func(tt devtest.T) {
		// ensure that the current and target leader are healthy and unpaused before transferring leadership
		require.True(tt, oldLeader.FetchSequencerHealthy(), "current leader's sequencer is not healthy, id", oldLeader)
		require.True(tt, targetLeader.FetchSequencerHealthy(), "target leader's sequencer is not healthy, id", targetLeader)
		require.False(tt, oldLeader.FetchPaused(), "current leader's sequencer is paused, id", oldLeader)
		require.False(tt, targetLeader.FetchPaused(), "target leader's sequencer is paused, id", targetLeader)

		// ensure that the current leader is the leader before transferring leadership
		require.True(tt, oldLeader.IsLeader(), "current leader was not found to be the leader")
		require.False(tt, targetLeader.IsLeader(), "target leader was already found to be the leader")

		oldLeader.TransferLeadershipTo(targetLeader.info)

		require.Eventually(
			tt,
			func() bool { return targetLeader.IsLeader() },
			5*time.Second, 1*time.Second, "target leader was not found to be the leader",
		)

		require.False(tt, oldLeader.IsLeader(), "old leader was still found to be the leader")

		// sometimes leadership transfer can cause a very brief period of unhealthiness,
		// but eventually, they should be healthy again
		require.Eventually(
			tt,
			func() bool { return oldLeader.FetchSequencerHealthy() && targetLeader.FetchSequencerHealthy() },
			3*time.Second, 1*time.Second, "at least one of the sequencers was found to be unhealthy",
		)
	})
}
