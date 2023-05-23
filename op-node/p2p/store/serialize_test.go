package store

import (
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRoundtripScoresV0(t *testing.T) {
	scores := scoreRecord{
		PeerScores: PeerScores{Gossip: 1234.52382},
		lastUpdate: time.UnixMilli(1923841),
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
		data     []byte
		expected scoreRecord
	}{
		{
			data: common.Hex2Bytes("00000000001D5B0140934A18644523F6"),
			expected: scoreRecord{
				PeerScores: PeerScores{Gossip: 1234.52382},
				lastUpdate: time.UnixMilli(1923841),
			},
		},
	}
	for idx, test := range tests {
		test := test
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			result, err := deserializeScoresV0(test.data)
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		})
	}
}
