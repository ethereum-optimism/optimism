package logmods

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

type handlerA struct {
	slog.Handler
}

func wrapA(inner slog.Handler) slog.Handler {
	return &handlerA{Handler: inner}
}

type handlerB struct {
	slog.Handler
}

func wrapB(inner slog.Handler) slog.Handler {
	return &handlerB{Handler: inner}
}

func (w *handlerB) Unwrap() slog.Handler {
	return w.Handler
}

type handlerC struct {
	slog.Handler
}

func wrapC(inner slog.Handler) slog.Handler {
	return &handlerC{Handler: inner}
}

func (w *handlerC) Unwrap() slog.Handler {
	return w.Handler
}

func TestFindHandler(t *testing.T) {
	t.Run("nested", func(t *testing.T) {
		a := wrapA(nil)
		b := wrapB(a)
		c := wrapC(b)
		h := c
		got1, ok := FindHandler[*handlerA](h)
		require.True(t, ok)
		require.Equal(t, a.(*handlerA), got1)
		got2, ok := FindHandler[*handlerB](h)
		require.True(t, ok)
		require.Equal(t, b.(*handlerB), got2)
		got3, ok := FindHandler[*handlerC](h)
		require.True(t, ok)
		require.Equal(t, c.(*handlerC), got3)
	})
}
