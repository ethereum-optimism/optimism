package clock

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleClock_Now(t *testing.T) {
	c := NewSimpleClock()
	require.Equal(t, time.Unix(0, 0), c.Now())
	expectedTime := uint64(time.Now().Unix())
	c.unix = atomic.Uint64{}
	c.unix.Store(expectedTime)
	require.Equal(t, time.Unix(int64(expectedTime), 0), c.Now())
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

func TestSimpleClock_Add(t *testing.T) {
	t.Run("PositiveDuration", func(t *testing.T) {
		c := NewSimpleClock()
		now := c.Now()
		d := 5 * time.Minute
		expected := now.Add(d)
		actual := c.Add(d)
		require.WithinDuration(t, expected, actual, time.Second)
	})

	t.Run("NegativeDuration", func(t *testing.T) {
		c := NewSimpleClock()
		c.SetTime(uint64(10 * time.Minute))
		now := c.Now()
		d := -5 * time.Minute
		expected := now.Add(d)
		actual := c.Add(d)
		require.WithinDuration(t, expected, actual, time.Second)
	})

	t.Run("NegativeDurationSinceUnix", func(t *testing.T) {
		c := NewSimpleClock()
		c.SetTime(5)
		d := -1 * time.Duration(5*time.Second)
		actual := c.Add(d)
		require.Equal(t, time.Unix(0, 0), actual)
	})

	t.Run("NegativeDurationTooLarge", func(t *testing.T) {
		c := NewSimpleClock()
		c.SetTime(5)
		d := -2 * time.Duration(5*time.Second)
		actual := c.Add(d)
		require.Equal(t, time.Unix(0, 0), actual)
	})
}
