package clock

import (
	"time"
)

type RWClock interface {
	Now() time.Time
}

// MinCheckedTimestamp returns the minimum checked unix timestamp.
// If the duration is 0, the returned minimum timestamp is 0.
// Otherwise, the minimum timestamp is the current unix time minus the duration.
// The subtraction operation is checked and returns 0 on underflow.
func MinCheckedTimestamp(clock RWClock, duration time.Duration) uint64 {
	if duration.Seconds() == 0 {
		return 0
	}
	// To compute t-d for a duration d, use t.Add(-d).
	// See https://pkg.go.dev/time#Time.Sub
	if clock.Now().Unix() > int64(duration.Seconds()) {
		return uint64(clock.Now().Add(-duration).Unix())
	}
	return 0
}
