package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPeerRetryState(t *testing.T) {
	t.Run("Initial state", func(t *testing.T) {
		state := &peerRetryState{
			backoff: initialStaticPeerBackoff,
		}
		require.Equal(t, initialStaticPeerBackoff, state.backoff)
		require.True(t, state.nextRetry.IsZero(), "nextRetry should be zero initially")
	})

	t.Run("Increase backoff", func(t *testing.T) {
		state := &peerRetryState{
			backoff: initialStaticPeerBackoff,
		}

		// First increase
		before := time.Now()
		state.increaseBackoff()
		after := time.Now()

		expectedBackoff := time.Duration(float64(initialStaticPeerBackoff) * 1.5)
		require.Equal(t, expectedBackoff, state.backoff)
		require.False(t, state.nextRetry.IsZero(), "nextRetry should be set after increase")
		require.True(t, state.nextRetry.After(before), "nextRetry should be after the call time")
		require.True(t, state.nextRetry.Before(after.Add(expectedBackoff+time.Second)), "nextRetry should be reasonably close to now+backoff")

		// Multiple increases should approach but not exceed max
		for i := 0; i < 10; i++ {
			state.increaseBackoff()
		}
		require.LessOrEqual(t, state.backoff, maxStaticPeerBackoff)
		require.Greater(t, state.backoff, initialStaticPeerBackoff)
	})

	t.Run("Reset backoff", func(t *testing.T) {
		state := &peerRetryState{
			backoff:   maxStaticPeerBackoff,
			nextRetry: time.Now().Add(time.Hour),
		}

		state.resetBackoff()

		require.Equal(t, initialStaticPeerBackoff, state.backoff)
		require.True(t, state.nextRetry.IsZero(), "nextRetry should be reset")
	})

	t.Run("Max backoff", func(t *testing.T) {
		state := &peerRetryState{
			backoff: maxStaticPeerBackoff,
		}

		state.increaseBackoff()

		require.Equal(t, maxStaticPeerBackoff, state.backoff, "backoff should not exceed maximum")
	})
}
