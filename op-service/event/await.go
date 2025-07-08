package event

import (
	"context"
	"iter"
	"reflect"
	"slices"
)

type awaitCtxKeyType struct{}

var awaitCtxKey = awaitCtxKeyType{}

// awaitData caries the follow-up await cases to run.
// Any events that are emitted with
// this attached to the context will be matched against the applicable handlers.
type awaitData struct {
	next []Handler
	// TODO completion state
}

func (d *awaitData) Servers() iter.Seq[Handler] {
	return slices.Values(d.next)
}

// Then emits an event, and runs the next handler on
// any events emitted by the handlers of this event.
// This returns an error if the event failed to be emitted.
func Then(ctx context.Context, ev Event, next ...Handler) error {
	// annotate to reference the next handler
	// (override any previous annotated value, since that is of the parent event, not relevant).
	ctx = context.WithValue(ctx, awaitCtxKey, &awaitData{next: next})
	return Emit(ctx, ev)
}

// AwaitCase is a handler that waits for a value (can be typed and filtered) to arrive,
// and makes the value accessible through a channel.
type AwaitCase[E Event] struct {
	result chan E
	opts   []HandlerOption
}

func (a AwaitCase[E]) C() <-chan E {
	return a.result
}

func (a AwaitCase[E]) Serve(ctx context.Context, ev Event) {
	a.result <- ev.(E)
}

func (a AwaitCase[E]) EventType() reflect.Type {
	return reflect.TypeFor[E]()
}

var _ Handler = (*AwaitCase[Event])(nil)

// Await creates an await-case, to wait for a specific event, optionally with filters.
// Different await cases can be created, Then can then start the work,
// and the results can be awaited with a select statement on each of the cases.
func Await[E Event](opts ...HandlerOption) *AwaitCase[E] {
	return &AwaitCase[E]{result: make(chan E), opts: opts}
}
