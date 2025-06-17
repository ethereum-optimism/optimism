package indexing

import "time"

// eventTimestamp helps tracking an event's last reference value with its last update time.
// It is used to avoid sending the same event multiple times within a specified ttl duration window.
type eventTimestamp[T comparable] struct {
	ttl time.Duration

	lastValue T
	lastTime  time.Time
}

func newEventTimestamp[T comparable](resendTTL time.Duration) eventTimestamp[T] {
	return eventTimestamp[T]{
		ttl: resendTTL,
	}
}

func (et *eventTimestamp[T]) Update(value T) bool {
	now := time.Now()
	if et.lastValue != value {
		et.lastValue = value
		et.lastTime = now
		return true
	} else if now.Sub(et.lastTime) > et.ttl {
		et.lastTime = now
		return true
	}
	return false
}
