package event

import (
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type LogTracer struct {
	log log.Logger
	lvl slog.Level
}

var _ Tracer = (*LogTracer)(nil)

func NewLogTracer(log log.Logger, lvl slog.Level) *LogTracer {
	return &LogTracer{
		log: log,
		lvl: lvl,
	}
}

func (lt *LogTracer) OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time) {
	lt.log.Log(lt.lvl, "Processing event", "deriver", name, "event", ev.Event,
		"emit_context", ev.EmitContext, "deriv_context", derivContext)
}

func (lt *LogTracer) OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	lt.log.Log(lt.lvl, "Processed event", "deriver", name, "duration", duration,
		"event", ev.Event, "emit_context", ev.EmitContext, "deriv_context", derivContext, "effect", effect)
}

func (lt *LogTracer) OnRateLimited(name string, derivContext uint64) {
	lt.log.Log(lt.lvl, "Rate-limited event-emission", "emitter", name, "context", derivContext)
}

func (lt *LogTracer) OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	lt.log.Log(lt.lvl, "Emitting event", "emitter", name, "event", ev.Event, "emit_context", ev.EmitContext, "deriv_context", derivContext)
}
