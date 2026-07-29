package retry

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExponential(t *testing.T) {
	strategy := &ExponentialStrategy{
		Min:       3 * time.Second,
		Max:       10 * time.Second,
		MaxJitter: 0,
	}

	require.Equal(t, 3*time.Second, strategy.Duration(-1))
	durations := []time.Duration{4, 5, 7, 10, 10}
	for i, dur := range durations {
		require.Equal(t, dur*time.Second, strategy.Duration(i), "attempt %d", i)
	}
	require.Equal(t, 10*time.Second, strategy.Duration(100))
	require.Equal(t, 10*time.Second, strategy.Duration(16000))
	require.Equal(t, 10*time.Second, strategy.Duration(math.MaxInt))
}
