package event

import (
	"context"
	"fmt"
	"sync/atomic"
)

var taskIDGen = new(atomic.Uint64)

const UndefinedTask = TaskID(0)

// TaskID is a unique identifier to attach to an event context,
// so that short-lived task-handlers can pick up this event.
type TaskID uint64

var _ HandlerOption = TaskID(0)

func (id TaskID) Apply(h *Handler) {
	h.Key.Task = id
}

func (id TaskID) String() string {
	return fmt.Sprintf("task-%d", uint64(id))
}

func NewTaskID() TaskID {
	// The first returned task-ID is non-zero,
	// to not conflict with UndefinedTask constant.
	return TaskID(taskIDGen.Add(1))
}

type taskIDCtxKeyType struct{}

var taskIDCtxKey = taskIDCtxKeyType{}

func CtxWithTaskID(ctx context.Context, id TaskID) context.Context {
	return context.WithValue(ctx, taskIDCtxKey, id)
}

func TaskIDFromCtx(ctx context.Context) TaskID {
	v := ctx.Value(taskIDCtxKey)
	if v == nil {
		return UndefinedTask
	}
	return v.(TaskID)
}
