package op_e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// [Category: Initial Setup]
// In this test, we test that we can successfully setup a working cluster.
func TestSequencerFailover_SetupCluster(t *testing.T) {
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	require.Equal(t, 3, len(conductors), "Expected 3 conductors")
	for _, con := range conductors {
		require.NotNil(t, con, "Expected conductor to be non-nil")
	}
}

// [Category: Sequencer Failover]
// Test that the sequencer can successfully failover to a new sequencer once the active sequencer goes down.
func TestSequencerFailover_ActiveSequencerDown(t *testing.T) {
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	ctx := context.Background()
	leaderId, _ := findLeader(t, conductors)
	sys.RollupNodes[leaderId].Stop(ctx) // Stop the current leader sequencer

	// The leadership change should occur with no errors
	require.NoError(t, waitForLeadershipChange(t, conductors[leaderId], false))

	// Confirm the new leader is different from the old leader
	newLeaderId, _ := findLeader(t, conductors)
	require.NotEqual(t, leaderId, newLeaderId, "Expected leader to change")

	// Check that the sequencer is healthy
	require.True(t, healthy(t, ctx, conductors[newLeaderId]))

	// Check if the new leader is sequencing
	active, err := sys.RollupClient(newLeaderId).SequencerActive(ctx)
	require.NoError(t, err)
	require.True(t, active, "Expected new leader to be sequencing")
}
