package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestLimiter(t *testing.T) {
	count := uint64(0)
	em := EmitterFunc(func(ev Event) {
		count += 1
	})
	hitRateLimitAt := uint64(0)
	// Test that we are able to hit the specified rate limit, and no earlier
	lim := NewLimiter(context.Background(), em, rate.Limit(10), 10, func() {
		if hitRateLimitAt != 0 {
			return
		}
		hitRateLimitAt = count
	})
	for i := 0; i < 30; i++ {
		lim.Emit(TestEvent{})
	}
	require.LessOrEqual(t, uint64(10), hitRateLimitAt)
}
