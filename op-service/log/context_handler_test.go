package log

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testKeyType struct{}

var testKey testKeyType = testKeyType{}

type testLogValuer struct {
	value string
}

func (t testLogValuer) LogValue() slog.Value {
	return slog.StringValue(t.value)
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
	ctx = RegisterLogAttrOnContext(ctx, "test", key)

	index = LogKeyIndexFromContext(ctx)
	require.NotNil(t, index)
	require.Len(t, index, 1)
	require.Equal(t, "test", index[0].name)
	require.Equal(t, key, index[0].key)

	// Test registering multiple keys
	key2 := "testKey2"
	ctx = RegisterLogAttrOnContext(ctx, "test2", key2)

	index = LogKeyIndexFromContext(ctx)
	require.Len(t, index, 2)

	// Check first attribute
	require.Equal(t, "test", index[0].name)
	require.Equal(t, key, index[0].key)

	// Check second attribute
	require.Equal(t, "test2", index[1].name)
	require.Equal(t, key2, index[1].key)
}

func TestContextHandler_Handle(t *testing.T) {
	inner := &testHandler{enabled: true}
	handler := WrapContextHandler(inner)

	ctx := context.Background()
	testValue := testLogValuer{value: "test-value"}

	// Register key and set context value
	ctx = RegisterLogAttrOnContext(ctx, "attr_name", testKey)
	ctx = context.WithValue(ctx, testKey, testValue)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)

	require.NoError(t, err)
	require.Len(t, inner.records, 1)

	// Check that the context attribute was added
	found := false
	inner.records[0].Attrs(func(attr slog.Attr) bool {
		if attr.Key == "attr_name" && attr.Value.String() == "{test-value}" {
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
	invalidValue := "not-a-log-valuer"

	// Register key with invalid value type
	ctx = RegisterLogAttrOnContext(ctx, "attr_name", testKey)
	ctx = context.WithValue(ctx, testKey, invalidValue)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)

	require.Error(t, err)
	require.Contains(t, err.Error(), "expected value to implement slog.LogValuer")
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
	keyA := "keyA"
	baseCtx = RegisterLogAttrOnContext(baseCtx, "a", keyA)

	// Fork 1: base + b = (a, b)
	fork1Ctx := RegisterLogAttrOnContext(baseCtx, "b", "keyB")

	// Fork 2: base + c = (a, c)
	fork2Ctx := RegisterLogAttrOnContext(baseCtx, "c", "keyC")

	// Verify base context has only 'a'
	baseIndex := LogKeyIndexFromContext(baseCtx)
	require.Len(t, baseIndex, 1)
	require.Equal(t, "a", baseIndex[0].name)
	require.Equal(t, keyA, baseIndex[0].key)

	// Verify fork1 has 'a' and 'b'
	fork1Index := LogKeyIndexFromContext(fork1Ctx)
	require.Len(t, fork1Index, 2)
	require.Equal(t, "a", fork1Index[0].name)
	require.Equal(t, keyA, fork1Index[0].key)
	require.Equal(t, "b", fork1Index[1].name)
	require.Equal(t, "keyB", fork1Index[1].key)

	// Verify fork2 has 'a' and 'c'
	fork2Index := LogKeyIndexFromContext(fork2Ctx)
	require.Len(t, fork2Index, 2)
	require.Equal(t, "a", fork2Index[0].name)
	require.Equal(t, keyA, fork2Index[0].key)
	require.Equal(t, "c", fork2Index[1].name)
	require.Equal(t, "keyC", fork2Index[1].key)
}
