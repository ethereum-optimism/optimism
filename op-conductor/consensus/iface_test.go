package consensus_test

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/stretchr/testify/require"
)

func TestClusterMembershipSerialization(t *testing.T) {
	expected := consensus.ClusterMembership{
		Servers: []consensus.ServerInfo{
			{
				ID:       "server1",
				Addr:     "127.0.0.1:5050",
				Suffrage: consensus.Voter,
			},
			{
				ID:       "server2",
				Addr:     "192.168.0.1:5050",
				Suffrage: consensus.Nonvoter,
			},
		}, Version: 123,
	}

	bytes, err := json.Marshal(expected)
	require.NoError(t, err)

	var actual consensus.ClusterMembership
	err = json.Unmarshal(bytes, &actual)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}

func TestClusterMembershipBackcompat(t *testing.T) {
	expected := consensus.ClusterMembership{
		Servers: []consensus.ServerInfo{
			{
				ID:       "server1",
				Addr:     "127.0.0.1:5050",
				Suffrage: consensus.Voter,
			},
			{
				ID:       "server2",
				Addr:     "192.168.0.1:5050",
				Suffrage: consensus.Nonvoter,
			},
		}, Version: 0,
	}

	legacyJson := `[{ "id": "server1", "addr": "127.0.0.1:5050", "suffrage": 0 }, { "id": "server2", "addr": "192.168.0.1:5050", "suffrage": 1 }]`

	var actual consensus.ClusterMembership
	err := json.Unmarshal([]byte(legacyJson), &actual)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}
