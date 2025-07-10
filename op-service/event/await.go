package event

import (
	"context"
	"errors"
	"reflect"
)

// TODO: unregister when task is done

// Then emits an event, and runs the next handler on
// any events emitted by the handlers of this event.
// This returns an error if the event failed to be emitted.
func Then(ctx context.Context, ev Event, opts ...HandlerOption) error {
	id := NewTaskID()
	bus := BusFromCtx(ctx)
	if bus == nil {
		return errors.New("cannot run task unattached from event system")
	}
	opts = append(opts, id)
	bus.Task(opts...) // register the response handler

	ctx = CtxWithTaskID(ctx, id)
	return Emit(ctx, ev)
}

// AwaitCase is a handler that waits for a value (can be typed and filtered) to arrive,
// and makes the value accessible through a channel.
type AwaitCase[E Event] struct {
	result chan E
}

var _ HandlerOption = (*AwaitCase[Event])(nil)

func (a *AwaitCase[E]) Apply(h *Handler) {
	h.Fn = a.serve
	h.Key.EventType = reflect.TypeFor[E]()
}

func (a *AwaitCase[E]) C() <-chan E {
	return a.result
}

func (a *AwaitCase[E]) serve(ctx context.Context, ev Event) {
	a.result <- ev.(E)
}

var _ HandlerOption = (*AwaitCase[Event])(nil)

// Await creates an await-case, to wait for a specific event.
// Different await cases can be created, Then can then start the work,
// and the results can be awaited with a select statement on each of the cases.
func Await[E Event]() *AwaitCase[E] {
	return &AwaitCase[E]{result: make(chan E)}
}
