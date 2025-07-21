package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/telemetry"
	"github.com/honeycombio/otel-config-go/otelconfig"
)

const (
	// SpanTimeout is the maximum duration a span can be active before it is automatically ended
	SpanTimeout = 30 * time.Second
	// ProcessL2PayloadSpanName is the name of the span for processing L2 payloads
	ProcessL2PayloadSpanName = "process_l2_payload"

	TraceTimeoutError = "timeout"
)

var (
	tracer trace.Tracer = noop.NewTracerProvider().Tracer("noop")
)

type contextKey int

const (
	// spansContextKey is the context key for storing spans by name. We need to store spans in the
	// context because it may need to carry multiple spans at the same time.
	spansContextKey contextKey = iota
)

// InitializeTracerFromConfig initializes the tracer from a tracing configuration
func InitTracing(tracingEnabled bool, serviceName string, tracerName string) error {
	if !tracingEnabled {
		return nil
	}

	if tracerName == "" {
		tracerName = "op-node"
	}

	if serviceName == "" {
		serviceName = "op-node"
	}

	_, _, err := telemetry.SetupOpenTelemetry(context.Background(),
		otelconfig.WithServiceName(serviceName),
	)
	if err != nil {
		return fmt.Errorf("failed to setup OpenTelemetry: %w", err)
	}
	// print out all configs of the tracer
	tracer = otel.Tracer(tracerName)
	return nil

}

func StartTraceProcessL2Payload(envelope *eth.ExecutionPayloadEnvelope) *eth.ExecutionPayloadEnvelopeWithContext {
	return startSpanFromPayload(envelope, ProcessL2PayloadSpanName)
}

func EndProcessL2Payload(envelope *eth.ExecutionPayloadEnvelopeWithContext) *eth.ExecutionPayloadEnvelopeWithContext {
	envelope.TraceContext = endSpanByName(envelope.TraceContext, ProcessL2PayloadSpanName)
	return envelope
}

func startSpanFromPayload(envelope *eth.ExecutionPayloadEnvelope, spanName string) *eth.ExecutionPayloadEnvelopeWithContext {
	return startSpanFromContext(context.Background(), spanName, envelope)
}

// getSpanByName retrieves a span by its name from the context
func getSpanByName(ctx context.Context, name string) trace.Span {
	if spans, ok := ctx.Value(spansContextKey).(map[string]trace.Span); ok {
		span := spans[name]
		return span
	}
	return nil
}

// addSpanToContext adds a span to the context with its name
func addSpanToContext(ctx context.Context, name string, span trace.Span) context.Context {
	spans := make(map[string]trace.Span)
	if existingSpans, ok := ctx.Value(spansContextKey).(map[string]trace.Span); ok {
		spans = existingSpans
	}
	spans[name] = span
	return context.WithValue(ctx, spansContextKey, spans)
}

func startSpanFromContext(ctx context.Context, spanName string, envelope *eth.ExecutionPayloadEnvelope) *eth.ExecutionPayloadEnvelopeWithContext {
	// Start a new span
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.String("payload_id", envelope.ExecutionPayload.ID().String()),
			attribute.Int64("payload_size", int64(len(envelope.ExecutionPayload.Transactions))),
			attribute.String("block_hash", envelope.ExecutionPayload.BlockHash.String()),
			attribute.Int64("block_number", int64(envelope.ExecutionPayload.BlockNumber)),
		),
	)

	// Add the span to context with its name
	ctx = addSpanToContext(ctx, spanName, span)

	// set a timeout for the span and end the span if it's not finished by then
	timeoutCtx, cancel := context.WithTimeout(ctx, SpanTimeout)
	go func() {
		<-timeoutCtx.Done()
		span.SetStatus(codes.Error, TraceTimeoutError)
		span.End()
		cancel()
	}()
	fmt.Println("span started", span.SpanContext().IsValid())

	// Return the context with the span and the enhanced envelope
	return &eth.ExecutionPayloadEnvelopeWithContext{
		ExecutionPayloadEnvelope: envelope,
		TraceContext:             ctx,
	}
}

// endSpanByName ends a span by its name and removes it from the context
func endSpanByName(ctx context.Context, spanName string) context.Context {
	if span := getSpanByName(ctx, spanName); span != nil {
		fmt.Println("ending span", span.SpanContext().IsValid())
		span.End()
		// Remove the span from the context
		if spans, ok := ctx.Value(spansContextKey).(map[string]trace.Span); ok {
			delete(spans, spanName)
			ctx = context.WithValue(ctx, spansContextKey, spans)
		}
	}
	return ctx
}