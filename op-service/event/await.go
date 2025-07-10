package event

import (
	"context"
	"reflect"
)

type Task struct {
	taskID TaskID
	bus    Bus
}

func NewTask(bus Bus) *Task {
	return &Task{
		taskID: NewTaskID(),
		bus:    bus,
	}
}

func (p *Task) Run(ctx context.Context, ev Event) *Task {
	ctx = CtxWithTaskID(ctx, p.taskID)
	err := Emit(ctx, ev)
	_ = err // TODO
	return p
}

func (p *Task) Handle(opts ...HandlerOption) *Task {
	opts = append(opts, p.taskID)
	p.bus.Task(opts...) // register the response handler
	return p
}

func (p *Task) Close() {
	p.bus.CancelTask(p.taskID)
}

// TODO: unregister automatically when task is done

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
func Await[E Event](cap int) *AwaitCase[E] {
	return &AwaitCase[E]{result: make(chan E, cap)}
}
