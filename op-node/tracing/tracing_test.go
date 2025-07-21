package tracing

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// Initialize tracing for tests
	InitTracing(true, "op-node", "op-node")
}

func TestStartTraceProcessL2Payload(t *testing.T) {
	// Create a test payload
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 123,
			BlockHash:   [32]byte{1, 2, 3},
			Transactions: []eth.Data{
				eth.Data("tx1"),
				eth.Data("tx2"),
			},
		},
	}

	// Start tracing
	result := StartTraceProcessL2Payload(payload)

	// Verify the result
	assert.NotNil(t, result)
	assert.Equal(t, payload, result.ExecutionPayloadEnvelope)
	assert.NotNil(t, result.TraceContext)

	// Verify span was created
	span := getSpanByName(result.TraceContext, ProcessL2PayloadSpanName)
	assert.NotNil(t, span)
}

func TestStartSpanFromContext(t *testing.T) {
	// Create a test payload
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 456,
			BlockHash:   [32]byte{4, 5, 6},
			Transactions: []eth.Data{
				eth.Data("tx3"),
			},
		},
	}

	// Start span from context
	ctx := context.Background()
	result := startSpanFromContext(ctx, "test_span", payload)

	// Verify the result
	assert.NotNil(t, result)
	assert.Equal(t, payload, result.ExecutionPayloadEnvelope)
	assert.NotNil(t, result.TraceContext)

	// Verify span was created with correct name
	span := getSpanByName(result.TraceContext, "test_span")
	assert.NotNil(t, span)
}

func TestSpanContextManagement(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Create a mock span
	mockSpan := trace.Span(nil) // In real tests, you'd use a mock span

	// Test adding span to context
	ctx = addSpanToContext(ctx, "test_span", mockSpan)

	// Test retrieving span
	retrievedSpan := getSpanByName(ctx, "test_span")
	assert.Equal(t, mockSpan, retrievedSpan)

	// Test retrieving non-existent span
	nonExistentSpan := getSpanByName(ctx, "non_existent")
	assert.Nil(t, nonExistentSpan)
}

func TestEndSpanByName(t *testing.T) {
	// Create a test payload and start tracing
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 789,
			BlockHash:   [32]byte{7, 8, 9},
		},
	}
	envelope := StartTraceProcessL2Payload(payload)

	// End the span
	ctx := endSpanByName(envelope.TraceContext, ProcessL2PayloadSpanName)

	// Verify span was removed from context
	span := getSpanByName(ctx, ProcessL2PayloadSpanName)
	assert.Nil(t, span)
}

func TestEndProcessL2Payload(t *testing.T) {
	// Create a test payload and start tracing
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 101,
			BlockHash:   [32]byte{1, 0, 1},
		},
	}
	envelope := StartTraceProcessL2Payload(payload)

	// End the process
	result := EndProcessL2Payload(envelope)

	// Verify the result
	assert.NotNil(t, result)
	assert.Equal(t, payload, result.ExecutionPayloadEnvelope)

	// Verify span was removed from context
	span := getSpanByName(result.TraceContext, ProcessL2PayloadSpanName)
	assert.Nil(t, span)
}

func TestMultipleSpans(t *testing.T) {
	// Create a test payload
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 303,
			BlockHash:   [32]byte{3, 0, 3},
		},
	}

	// Start multiple spans
	envelope := StartTraceProcessL2Payload(payload)
	additionalEnvelope := startSpanFromContext(envelope.TraceContext, "additional_span", payload)

	// Verify both spans exist
	mainSpan := getSpanByName(additionalEnvelope.TraceContext, ProcessL2PayloadSpanName)
	additionalSpan := getSpanByName(additionalEnvelope.TraceContext, "additional_span")
	assert.NotNil(t, mainSpan)
	assert.NotNil(t, additionalSpan)

	// End one span
	ctx := endSpanByName(additionalEnvelope.TraceContext, "additional_span")

	// Verify only one span remains
	mainSpan = getSpanByName(ctx, ProcessL2PayloadSpanName)
	additionalSpan = getSpanByName(ctx, "additional_span")
	assert.NotNil(t, mainSpan)
	assert.Nil(t, additionalSpan)
}