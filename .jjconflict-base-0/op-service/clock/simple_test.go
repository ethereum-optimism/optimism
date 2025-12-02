package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleClock_Now(t *testing.T) {
	c := NewSimpleClock()
	require.Equal(t, time.Unix(0, 0), c.Now())
	expectedTime := time.Now()
	c.v.Store(&expectedTime)
	require.Equal(t, expectedTime, c.Now())
}

func TestSimpleClock_SetTime(t *testing.T) {
	tests := []struct {
		name         string
		expectedTime int64
	}{
		{
			name:         "SetZeroTime",
			expectedTime: 0,
		},
		{
			name:         "SetZeroUnixTime",
			expectedTime: time.Unix(0, 0).Unix(),
		},

		{
			name:         "SetCurrentTime",
			expectedTime: time.Now().Unix(),
		},
		{
			name:         "SetFutureTime",
			expectedTime: time.Now().Add(time.Hour).Unix(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			c := NewSimpleClock()
			c.SetTime(uint64(test.expectedTime))
			require.Equal(t, time.Unix(test.expectedTime, 0), c.Now())
		})
	}
}
