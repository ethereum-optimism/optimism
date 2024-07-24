package event

import (
	"time"
)

type Tracer interface {
	OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time)
	OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool)
	OnRateLimited(name string, derivContext uint64)
	OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time)
}
