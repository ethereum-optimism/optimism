package interop

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBackoffAfterNoProgress verifies that no-op verification rounds back off
// only briefly while a backlog is being drained (advances reset the streak),
// and escalate to the full backoff once sustained no-ops indicate we are caught
// up to the tip — so the verifier drains backlogs fast but idles calmly.
func TestBackoffAfterNoProgress(t *testing.T) {
	require.Less(t, catchupBackoffPeriod, backoffPeriod,
		"catch-up backoff must be shorter than the steady-state backoff to drain faster")

	i := &Interop{}

	// The first maxFastNoProgress consecutive no-ops use the short catch-up
	// backoff so a backlog drains promptly.
	for n := 1; n <= maxFastNoProgress; n++ {
		require.Equal(t, catchupBackoffPeriod, i.backoffAfterNoProgress(),
			"no-op #%d while behind should use the catch-up backoff", n)
	}

	// Once the streak exceeds maxFastNoProgress (caught up to the tip, no data
	// to process) it falls back to the full backoff to avoid busy-polling.
	require.Equal(t, backoffPeriod, i.backoffAfterNoProgress(),
		"sustained no-ops should fall back to the steady-state backoff")
	require.Equal(t, backoffPeriod, i.backoffAfterNoProgress())

	// An advancing round resets the streak (progress() sets noProgressStreak=0),
	// so the next no-op is back in fast catch-up mode.
	i.noProgressStreak = 0
	require.Equal(t, catchupBackoffPeriod, i.backoffAfterNoProgress(),
		"after an advance, the next no-op should use the catch-up backoff again")
}
