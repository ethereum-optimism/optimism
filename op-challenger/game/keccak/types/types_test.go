package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShouldVerify(t *testing.T) {
	tests := []struct {
		name      string
		timestamp uint64
		countered bool
		now       int64
		expected  bool
	}{
		{
			name:      "IgnoreNotFinalizedAndNotCountered",
			timestamp: 0,
			countered: false,
			now:       100,
			expected:  false,
		},
		{
			name:      "VerifyFinalizedAndNotCountered",
			timestamp: 50,
			countered: false,
			now:       100,
			expected:  true,
		},
		{
			name:      "IgnoreFinalizedAndCountered",
			timestamp: 50,
			countered: true,
			now:       100,
			expected:  false,
		},
		{
			name:      "IgnoreNotFinalizedAndCountered",
			timestamp: 0,
			countered: true,
			now:       100,
			expected:  false,
		},
		{
			name:      "IgnoreFinalizedBeforeTimeWindowAndNotCountered",
			timestamp: 50,
			countered: false,
			now:       50 + int64((2 * time.Hour).Seconds()),
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := LargePreimageMetaData{
				Timestamp: test.timestamp,
				Countered: test.countered,
			}
			require.Equal(t, test.expected, metadata.ShouldVerify(time.Unix(test.now, 0), 1*time.Hour))
		})
	}
}
