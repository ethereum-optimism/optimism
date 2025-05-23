package logfilter

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type ctxTestKeyType struct{}

var ctxTestKey = ctxTestKeyType{}

func TestFilters(t *testing.T) {
	ctx := context.Background()
	t.Run("noop", func(t *testing.T) {
		require.Equal(t, log.LevelTrace, Noop()(ctx, log.LevelTrace))
		require.Equal(t, log.LevelError, Noop()(ctx, log.LevelError))
	})
	t.Run("mute", func(t *testing.T) {
		require.Equal(t, MuteLvl, Mute()(ctx, log.LevelTrace))
		require.Equal(t, MuteLvl, Mute()(ctx, log.LevelError))
	})
	t.Run("as", func(t *testing.T) {
		replacement := slog.Level(123)
		require.Equal(t, replacement, As(replacement)(ctx, log.LevelTrace))
		require.Equal(t, replacement, As(replacement)(ctx, log.LevelError))
	})
	t.Run("add", func(t *testing.T) {
		require.Equal(t, log.LevelTrace+2, Add(2)(ctx, log.LevelTrace))
		require.Equal(t, log.LevelError-2, Add(-2)(ctx, log.LevelError))
	})
	t.Run("minimum", func(t *testing.T) {
		require.Equal(t, MuteLvl, Minimum(log.LevelError)(ctx, log.LevelTrace))
		require.Equal(t, MuteLvl, Minimum(log.LevelError)(ctx, log.LevelInfo))
		require.Equal(t, log.LevelError, Minimum(log.LevelError)(ctx, log.LevelError))
		require.Equal(t, log.LevelCrit, Minimum(log.LevelError)(ctx, log.LevelCrit))
	})
	t.Run("if-match", func(t *testing.T) {
		// create a filter that sets the log level to Info if the key/value matches
		f := IfMatch(ctxTestKey, "foo", As(log.LevelInfo))
		require.Equal(t, f(ctx, log.LevelDebug), log.LevelDebug, "no match because no key/val")
		require.Equal(t, f(context.WithValue(ctx, ctxTestKey, "bar"), log.LevelDebug), log.LevelDebug, "no match because different key/val")
		require.Equal(t, f(context.WithValue(ctx, ctxTestKey, "foo"), log.LevelDebug), log.LevelInfo, "match")
	})
}
