package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIConfig_Check_logBackfillRequiresActivation(t *testing.T) {
	ptr := func(u uint64) *uint64 { return &u }
	t.Run("ok with both", func(t *testing.T) {
		c := &CLIConfig{L1NodeAddr: "http://x", InteropActivationTimestamp: ptr(1), InteropLogBackfillDepth: time.Hour}
		require.NoError(t, c.Check())
	})
	t.Run("backfill without activation", func(t *testing.T) {
		c := &CLIConfig{L1NodeAddr: "http://x", InteropLogBackfillDepth: time.Hour}
		require.ErrorContains(t, c.Check(), "interop.log-backfill-depth requires interop.activation-timestamp")
	})
	t.Run("negative depth", func(t *testing.T) {
		c := &CLIConfig{L1NodeAddr: "http://x", InteropActivationTimestamp: ptr(1), InteropLogBackfillDepth: -time.Second}
		require.ErrorContains(t, c.Check(), "interop.log-backfill-depth must be >= 0")
	})
}
