package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSystemClock_SleepCtx(t *testing.T) {
	t.Run("ReturnWhenContextDone", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		err := SystemClock.SleepCtx(ctx, 5*time.Minute)
		end := time.Now()
		require.ErrorIs(t, err, context.Canceled)
		// The call shouldn't block for the 5 minutes, but use a high tolerance as test servers can be slow
		// and clocks are inaccurate.
		require.Less(t, end.Sub(start), time.Minute)
	})

	t.Run("ReturnAfterDuration", func(t *testing.T) {
		start := time.Now()
		err := SystemClock.SleepCtx(context.Background(), 100*time.Millisecond)
		end := time.Now()
		require.NoError(t, err)
		// Require the call to sleep for at least a little. Use a high tolerance since clocks can be quite inaccurate.
		require.Greater(t, end.Sub(start), 5*time.Millisecond, "should sleep at least a bit")
	})
}
