package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

type testHandler struct {
	entries []log.Record
}

func (t *testHandler) Log(r *log.Record) error {
	t.entries = append(t.entries, *r)
	return nil
}

// TestLogBuilder_With tests that the [LogBuilder.With] method works as expected.
func TestLogBuilder_With(t *testing.T) {
	t.Run("SinglePair_Succeeds", func(t *testing.T) {
		builder := NewLogBuilder()
		builder.With("key", "value")
		logger := builder.Build()
		handler := &testHandler{}
		logger.SetHandler(handler)
		logger.Info("test")
		assert.Equal(t, 1, len(handler.entries))
		assert.Equal(t, "key", handler.entries[0].Ctx[0])
		assert.Equal(t, "value", handler.entries[0].Ctx[1])
	})

	t.Run("MultiplePairs_Succeed", func(t *testing.T) {
		builder := NewLogBuilder()
		builder.With("key", "value")
		builder.With("key2", "value2")
		logger := builder.Build()
		handler := &testHandler{}
		logger.SetHandler(handler)
		logger.Info("test")
		assert.Equal(t, 1, len(handler.entries))
		assert.Equal(t, "key", handler.entries[0].Ctx[0])
		assert.Equal(t, "value", handler.entries[0].Ctx[1])
		assert.Equal(t, "key2", handler.entries[0].Ctx[2])
		assert.Equal(t, "value2", handler.entries[0].Ctx[3])
	})

	t.Run("NoContext_Succeeds", func(t *testing.T) {
		builder := NewLogBuilder()
		logger := builder.Build()
		handler := &testHandler{}
		logger.SetHandler(handler)
		logger.Info("test")
		assert.Equal(t, 1, len(handler.entries))
		assert.Equal(t, 0, len(handler.entries[0].Ctx))
	})
}
