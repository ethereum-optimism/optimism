package store

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundtripScoresV0(t *testing.T) {
	scores := scoreRecord{
		PeerScores: PeerScores{Gossip: GossipScores{Total: 1234.52382}},
		LastUpdate: 1923841,
	}
	data, err := serializeScoresV0(scores)
	require.NoError(t, err)

	result, err := deserializeScoresV0(data)
	require.NoError(t, err)
	require.Equal(t, scores, result)
}

// TestParseHistoricSerializations checks that existing data can still be deserialized
// Adding new fields should not require bumping the version. Removing fields may require bumping.
// Scores should always default to 0.
// A new entry should be added to this test each time any fields are changed to ensure it can always be deserialized
func TestParseHistoricSerializationsV0(t *testing.T) {
	tests := []struct {
		data     string
		expected scoreRecord
	}{
		{
			data: `{"peerScores":{"gossip":{"total":1234.52382,"blocks":{"timeInMesh":1234,"firstMessageDeliveries":12,"meshMessageDeliveries":34,"invalidMessageDeliveries":56},"IPColocationFactor":12.34,"behavioralPenalty":56.78},"reqRespSync":123456},"lastUpdate":1923841}`,
			expected: scoreRecord{
				PeerScores: PeerScores{
					Gossip: GossipScores{
						Total: 1234.52382,
						Blocks: TopicScores{
							TimeInMesh:               1234,
							FirstMessageDeliveries:   12,
							MeshMessageDeliveries:    34,
							InvalidMessageDeliveries: 56,
						},
						IPColocationFactor: 12.34,
						BehavioralPenalty:  56.78,
					},
					ReqRespSync: 123456,
				},
				LastUpdate: 1923841,
			},
		},
	}
	for idx, test := range tests {
		test := test
		out, _ := json.Marshal(&test.expected)
		t.Log(string(out))
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			result, err := deserializeScoresV0([]byte(test.data))
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		})
	}
}
