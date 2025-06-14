package rwel

import (
	"context"
	"fmt"
	"sync/atomic"
)

const UnknownID ID = 0

type ID uint64

func (id ID) String() string {
	return fmt.Sprintf("RWEL-%d", uint64(id))
}

var idGen = new(atomic.Uint64)

func NextID() ID {
	return ID(idGen.Add(1))
}

type idContextKeyType struct{}

var idContextKey = idContextKeyType{}

func IDFromContext(ctx context.Context) ID {
	v := ctx.Value(idContextKey)
	if v == nil {
		return UnknownID
	}
	return v.(ID)
}

func WithID(ctx context.Context, id ID) context.Context {
	return context.WithValue(ctx, idContextKey, id)
}
