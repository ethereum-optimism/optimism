// Tests: InteropELSyncBootstrap_Feature.md §3 — T_lo = max(T_act, T_tip - D_log).
package interop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogBackfillLowerBound(t *testing.T) {
	t.Run("zero depth returns tip", func(t *testing.T) {
		got := LogBackfillLowerBound(1000, 100, 0)
		require.Equal(t, uint64(1000), got)
	})
	t.Run("clamps to activation when raw before activation", func(t *testing.T) {
		// tip 200, depth 500s -> raw 0, max(100,0)=100
		got := LogBackfillLowerBound(200, 100, 500*time.Second)
		require.Equal(t, uint64(100), got)
	})
	t.Run("uses raw when above activation", func(t *testing.T) {
		// tip 1000, depth 100s -> raw 900, max(100,900)=900
		got := LogBackfillLowerBound(1000, 100, 100*time.Second)
		require.Equal(t, uint64(900), got)
	})
	t.Run("tip below depth underflows to zero then clamps to activation", func(t *testing.T) {
		got := LogBackfillLowerBound(50, 40, 100*time.Second)
		require.Equal(t, uint64(40), got)
	})
}
