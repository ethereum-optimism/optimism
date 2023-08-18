package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExponential(t *testing.T) {
	strategy := &ExponentialStrategy{
		Min:       3000 * time.Millisecond,
		Max:       10000 * time.Millisecond,
		MaxJitter: 0,
	}

	durations := []time.Duration{4, 5, 7, 10, 10}
	for i, dur := range durations {
		require.Equal(t, dur*time.Second, strategy.Duration(i))
	}
}
