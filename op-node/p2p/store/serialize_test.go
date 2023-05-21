package store

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRoundtripScoresV0(t *testing.T) {
	scores := PeerScores{
		Gossip: 1234.52382,
	}
	data, err := serializeScoresV0(scores)
	require.NoError(t, err)

	result, err := deserializeScoresV0(data)
	require.NoError(t, err)
	require.Equal(t, scores, result)
}

// TestParseHistoricSerializations checks that existing data can still be deserialized
// Adding new fields should not require bumping the version, only removing fields
// A new entry should be added to this test each time any fields are changed to ensure it can always be deserialized
func TestParseHistoricSerializationsV0(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected PeerScores
	}{
		{
			name:     "GossipOnly",
			data:     common.Hex2Bytes("40934A18644523F6"),
			expected: PeerScores{Gossip: 1234.52382},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := deserializeScoresV0(test.data)
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		})
	}
}
