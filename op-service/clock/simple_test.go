package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleClock_Now(t *testing.T) {
	c := NewSimpleClock()
	expectedTime := time.Now()
	c.time = expectedTime
	require.Equal(t, expectedTime, c.Now())
}

func TestSimpleClock_SetTime(t *testing.T) {
	tests := []struct {
		name         string
		expectedTime time.Time
	}{
		{
			name:         "SetZeroUnixTime",
			expectedTime: time.Unix(0, 0),
		},
		{
			name:         "SetEmptyTime",
			expectedTime: time.Time{},
		},
		{
			name:         "SetCurrentTime",
			expectedTime: time.Now(),
		},
		{
			name:         "SetFutureTime",
			expectedTime: time.Now().Add(time.Hour),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			c := NewSimpleClock()
			c.SetTime(test.expectedTime)
			require.Equal(t, test.expectedTime, c.Now())
		})
	}
}
