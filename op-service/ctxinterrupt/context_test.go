package ctxinterrupt

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContextKeyIsUnique(t *testing.T) {
	ass := require.New(t)
	ctx := context.Background()
	ass.Nil(ctx.Value(waiterContextKey))
	ctx = context.WithValue(ctx, waiterContextKey, 1)
	ass.Equal(ctx.Value(waiterContextKey), 1)
	ctx = context.WithValue(ctx, waiterContextKey, 2)
	ass.Equal(ctx.Value(waiterContextKey), 2)
	ass.Nil(ctx.Value(struct{}{}))
}
