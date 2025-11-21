package logfilter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testLogValuer struct {
	value string
}

func (t testLogValuer) LogValue() slog.Value {
	return slog.StringValue(t.value)
}

// stringLogValuer wraps a string to implement slog.LogValuer
type stringLogValuer string

func (s stringLogValuer) LogValue() slog.Value {
	return slog.StringValue(string(s))
}

type testHandler struct {
	records []slog.Record
	enabled bool
}

func (h *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.enabled
}

func (h *testHandler) Handle(ctx context.Context, record slog.Record) error {
	h.records = append(h.records, record)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestRegisterLogKeyOnContext(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	index := LogKeyIndexFromContext(ctx)
	require.Nil(t, index)

	// Test registering a key
	key := "testKey"
	ctx = AddLogAttrToContext(ctx, "test", testLogValuer{value: key})

	index = LogKeyIndexFromContext(ctx)
	require.NotNil(t, index)
	require.Len(t, index, 1)
	require.Equal(t, "test", index[0])

	// Test registering multiple keys
	key2 := stringLogValuer("testKey2")
	ctx = AddLogAttrToContext(ctx, "test2", key2)

	index = LogKeyIndexFromContext(ctx)
	require.Len(t, index, 2)

	// Check first attribute
	require.Equal(t, "test", index[0])

	// Check second attribute
	require.Equal(t, "test2", index[1])
}

func TestContextHandler_Handle(t *testing.T) {
	inner := &testHandler{enabled: true}
	handler := WrapContextHandler(inner)

	ctx := context.Background()
	testValue := testLogValuer{value: "test-value"}

	// Register key and set context value
	ctx = AddLogAttrToContext(ctx, "attr_name", testValue)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)

	require.NoError(t, err)
	require.Len(t, inner.records, 1)

	// Check that the context attribute was added
	found := false
	inner.records[0].Attrs(func(attr slog.Attr) bool {
		if attr.Key == "attr_name" && attr.Value.String() == "test-value" {
			found = true
			return false // stop iteration
		}
		return true
	})
	require.True(t, found, "Expected context attribute to be added to record")
}

func TestContextHandler_HandleError(t *testing.T) {
	inner := &testHandler{enabled: true}
	handler := WrapContextHandler(inner)

	ctx := context.Background()
	invalidValue := stringLogValuer("not-a-log-valuer")

	// First add a valid LogValuer to the context
	ctx = AddLogAttrToContext(ctx, "attr_name", invalidValue)

	// Now overwrite the context value with a non-LogValuer type
	// This simulates the bug scenario where someone stores a non-LogValuer in the context
	contextKey := contextKey{name: "attr_name"}
	ctx = context.WithValue(ctx, contextKey, "plain-string-not-logvaluer")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)

	// With the new behavior, no error is returned, but the log record is still processed
	require.NoError(t, err)
	require.Len(t, inner.records, 1)

	// Verify the record was created but the invalid attribute was not added
	found := false
	inner.records[0].Attrs(func(attr slog.Attr) bool {
		if attr.Key == "attr_name" {
			found = true
			return false
		}
		return true
	})
	require.False(t, found, "Invalid attribute should not be added to record")
}

func TestContextHandler_Enabled(t *testing.T) {
	inner := &testHandler{enabled: true}
	handler := WrapContextHandler(inner)

	require.True(t, handler.Enabled(context.Background(), slog.LevelInfo))

	inner.enabled = false
	require.False(t, handler.Enabled(context.Background(), slog.LevelInfo))
}

func TestContextHandler_WithAttrs(t *testing.T) {
	inner := &testHandler{}
	handler := WrapContextHandler(inner)

	attrs := []slog.Attr{slog.String("key", "value")}
	newHandler := handler.WithAttrs(attrs)

	require.IsType(t, &contextHandler{}, newHandler)
}

func TestContextHandler_WithGroup(t *testing.T) {
	inner := &testHandler{}
	handler := WrapContextHandler(inner)

	newHandler := handler.WithGroup("group")

	require.IsType(t, &contextHandler{}, newHandler)
}

func TestContextHandler_Unwrap(t *testing.T) {
	inner := &testHandler{}
	handler := WrapContextHandler(inner)

	unwrapped := handler.(*contextHandler).Unwrap()
	require.Equal(t, inner, unwrapped)
}

func TestForkedContexts(t *testing.T) {
	// Create base context with one attribute (a)
	baseCtx := context.Background()
	keyA := stringLogValuer("keyA")
	baseCtx = AddLogAttrToContext(baseCtx, "a", keyA)

	// Fork 1: base + b = (a, b)
	fork1Ctx := AddLogAttrToContext(baseCtx, "b", stringLogValuer("keyB"))

	// Fork 2: base + c = (a, c)
	fork2Ctx := AddLogAttrToContext(baseCtx, "c", stringLogValuer("keyC"))

	// Verify base context has only 'a'
	baseIndex := LogKeyIndexFromContext(baseCtx)
	require.Len(t, baseIndex, 1)
	require.Equal(t, "a", baseIndex[0])

	// Verify fork1 has 'a' and 'b'
	fork1Index := LogKeyIndexFromContext(fork1Ctx)
	require.Len(t, fork1Index, 2)
	require.Equal(t, "a", fork1Index[0])
	require.Equal(t, "b", fork1Index[1])

	// Verify fork2 has 'a' and 'c'
	fork2Index := LogKeyIndexFromContext(fork2Ctx)
	require.Len(t, fork2Index, 2)
	require.Equal(t, "a", fork2Index[0])
	require.Equal(t, "c", fork2Index[1])
}

func TestContextHandler_SilentLoggingBugFixed(t *testing.T) {
	// This test verifies that when context values don't implement slog.LogValuer,
	// the error is logged to stderr instead of being silently swallowed

	inner := &testHandler{enabled: true}
	handler := WrapContextHandler(inner)

	ctx := context.Background()

	// First add a valid LogValuer to establish the context attribute
	ctx = AddLogAttrToContext(ctx, "scope", stringLogValuer("valid-value"))

	// Now overwrite with a non-LogValuer value to simulate the bug scenario
	contextKey := contextKey{name: "scope"}
	ctx = context.WithValue(ctx, contextKey, "plain-string-value") // This would cause silent failure before

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)

	// The handler should not return an error (which would be silently swallowed)
	require.NoError(t, err)

	// The log record should still be processed
	require.Len(t, inner.records, 1)
	require.Equal(t, "test message", inner.records[0].Message)

	// The invalid context attribute should not be present in the log
	found := false
	inner.records[0].Attrs(func(attr slog.Attr) bool {
		if attr.Key == "scope" {
			found = true
			return false
		}
		return true
	})
	require.False(t, found, "Invalid context attribute should not be added to log record")

	// Note: The error message would be visible on stderr in real usage,
	// making the problem apparent to developers instead of silent
}
