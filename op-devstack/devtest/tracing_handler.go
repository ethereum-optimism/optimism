package devtest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func wrapTracingHandler(h slog.Handler) slog.Handler {
	return &tracingHandler{Handler: h}
}

type tracingHandler struct {
	slog.Handler
}

var _ logmods.Handler = (*tracingHandler)(nil)

func (h *tracingHandler) Unwrap() slog.Handler {
	return h.Handler
}

func (h *tracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &tracingHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *tracingHandler) WithGroup(name string) slog.Handler {
	return &tracingHandler{Handler: h.Handler.WithGroup(name)}
}

func (h *tracingHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		recorder := &traceAttrAccumulator{}
		record.Attrs(func(attr slog.Attr) bool {
			recorder.register(attr)
			return true
		})
		span.AddEvent(record.Message, trace.WithAttributes(recorder.kv...))
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String("span_id", spanCtx.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

type traceAttrAccumulator struct {
	kv []attribute.KeyValue
}

func (ac *traceAttrAccumulator) register(attr slog.Attr) {
	switch attr.Value.Kind() {
	case slog.KindAny:
		ac.kv = append(ac.kv, attribute.String(attr.Key, fmt.Sprintf("%v", attr.Value.Any())))
	case slog.KindBool:
		ac.kv = append(ac.kv, attribute.Bool(attr.Key, attr.Value.Bool()))
	case slog.KindDuration:
		ac.kv = append(ac.kv, attribute.String(attr.Key, attr.Value.Duration().String()))
	case slog.KindFloat64:
		ac.kv = append(ac.kv, attribute.Float64(attr.Key, attr.Value.Float64()))
	case slog.KindInt64:
		ac.kv = append(ac.kv, attribute.Int64(attr.Key, attr.Value.Int64()))
	case slog.KindString:
		ac.kv = append(ac.kv, attribute.String(attr.Key, attr.Value.String()))
	case slog.KindTime:
		ac.kv = append(ac.kv, attribute.String(attr.Key, attr.Value.Time().String()))
	case slog.KindUint64:
		val := attr.Value.Uint64()
		ac.kv = append(ac.kv, attribute.Int64(attr.Key, int64(val)))
		if val > uint64(1<<63-1) {
			ac.kv = append(ac.kv, attribute.Bool(attr.Key+".overflow", true))
			ac.kv = append(ac.kv, attribute.String(attr.Key+".actual", fmt.Sprintf("%d", val)))
		}
	case slog.KindGroup:
		for _, groupAttr := range attr.Value.Group() {
			ac.register(groupAttr)
		}
	case slog.KindLogValuer:
		ac.register(slog.Attr{Key: attr.Key, Value: attr.Value.LogValuer().LogValue()})
	}
}
