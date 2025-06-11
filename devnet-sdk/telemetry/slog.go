package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

func WrapHandler(h slog.Handler) slog.Handler {
	return &tracingHandler{
		Handler: h,
	}
}

type tracingHandler struct {
	slog.Handler
}

var _ logmods.Handler = (*tracingHandler)(nil)

func (h *tracingHandler) Unwrap() slog.Handler {
	return h.Handler
}

func (h *tracingHandler) Handle(ctx context.Context, record slog.Record) error {
	// Send log entries as events to the tracer
	if h.Handler.Enabled(ctx, record.Level) {
		span := trace.SpanFromContext(ctx)
		if span.IsRecording() {
			attrRecorder := &attrAccumulator{}
			record.Attrs(func(a slog.Attr) bool {
				attrRecorder.register(a)
				return true
			})
			span.AddEvent(record.Message, trace.WithAttributes(attrRecorder.kv...))
		}
	}

	// Conversely add tracing data to the local logs
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String("span_id", spanCtx.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

type attrAccumulator struct {
	kv []attribute.KeyValue
}

func (ac *attrAccumulator) register(a slog.Attr) {
	switch a.Value.Kind() {
	case slog.KindAny:
		ac.kv = append(ac.kv, attribute.String(a.Key, fmt.Sprintf("%v", a.Value.Any())))
	case slog.KindBool:
		ac.kv = append(ac.kv, attribute.Bool(a.Key, a.Value.Bool()))
	case slog.KindDuration:
		ac.kv = append(ac.kv, attribute.String(a.Key, a.Value.Duration().String()))
	case slog.KindFloat64:
		ac.kv = append(ac.kv, attribute.Float64(a.Key, a.Value.Float64()))
	case slog.KindInt64:
		ac.kv = append(ac.kv, attribute.Int64(a.Key, a.Value.Int64()))
	case slog.KindString:
		ac.kv = append(ac.kv, attribute.String(a.Key, a.Value.String()))
	case slog.KindTime:
		ac.kv = append(ac.kv, attribute.String(a.Key, a.Value.Time().String()))
	case slog.KindUint64:
		val := a.Value.Uint64()
		ac.kv = append(ac.kv, attribute.Int64(a.Key, int64(val)))
		// detect overflows
		if val > uint64(1<<63-1) {
			// Value doesn't properly fit in int64
			ac.kv = append(ac.kv, attribute.Bool(a.Key+".overflow", true))
			ac.kv = append(ac.kv, attribute.String(a.Key+".actual", fmt.Sprintf("%d", val)))
		}
	case slog.KindGroup:
		for _, attr := range a.Value.Group() {
			ac.register(attr)
		}
	case slog.KindLogValuer:
		value := a.Value.LogValuer().LogValue()
		ac.register(slog.Attr{Key: a.Key, Value: value})
	}
}
