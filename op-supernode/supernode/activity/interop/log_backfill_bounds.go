package interop

import "time"

// LogBackfillLowerBound returns T_lo = max(T_act, T_tip - D_log) in unix seconds (L2).
// Spec: InteropELSyncBootstrap_Feature.md §3 — never ingest logs for timestamps before activation.
func LogBackfillLowerBound(tipTimestampUnix, activationTimestampUnix uint64, logBackfillDepth time.Duration) uint64 {
	if logBackfillDepth <= 0 {
		return tipTimestampUnix
	}
	sub := uint64(logBackfillDepth / time.Second)
	var raw uint64
	if tipTimestampUnix >= sub {
		raw = tipTimestampUnix - sub
	} else {
		raw = 0
	}
	if raw < activationTimestampUnix {
		return activationTimestampUnix
	}
	return raw
}
