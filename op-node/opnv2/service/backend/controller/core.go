package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/event"
)

// TODO: consider automating events that run upon timeout/cancel of the context
type TimeoutEvent struct {
	Err
}

func (ev TimeoutEvent) String() string {
	return "timeout"
}

// Err can be embedded into events,
// to give it an error-field while implementing the error interface at the same time.
// This allows for catch-all error handling in event type-switches.
type Err error

type EventHandler func(ctx context.Context, ev event.Event)

type NextStep struct {
	// TODO: may need to nil this after the handler calls the next Emit,
	// or handle stale events somehow.
	Handler *TaskStateV2
}

type eventNextKeyType struct{}

var eventNextKey = eventNextKeyType{}

func ContextWithNextStep(ctx context.Context, n *NextStep) context.Context {
	return context.WithValue(ctx, eventNextKey, n)
}

func NextStepFromContext(ctx context.Context) *NextStep {
	v := ctx.Value(eventNextKey)
	if v == nil {
		return nil
	}
	return v.(*NextStep)
}

type TaskStateV2 struct {
	task struct {
		Inspect func()

		Default     EventHandler
		LastEmitCtx context.Context
		LastEvent   event.Event
		LastSeqNr   uint64

		// Next thing to do with an event that we previously emitted.
		Next EventHandler

		Emitter event.Emitter
	}
}

func (ts *TaskStateV2) Init(emitter event.Emitter, inspect func()) {
	ts.task.Emitter = emitter
	ts.task.Inspect = inspect
}

func (ts *TaskStateV2) Inspect() {
	if ts.IsBusy() {
		return
	}
	ts.task.Inspect()
}

func (ts *TaskStateV2) IsBusy() bool {
	return ts.task.Next != nil
}

func (ts *TaskStateV2) AssertNotBusy() {
	if ts.IsBusy() {
		panic("task is active")
	}
}

// TODO: the controller should be able to look at the event context,
// pull out the task the event originated from, and then apply the event to the task,
// on the thread of the Controller itself, so we have synchronous controller state access.
func (ts *TaskStateV2) OnEvent(ctx context.Context, ev event.Event) {
	if ts.task.Next != nil {
		ts.task.Next(ctx, ev)
	}
	ts.task.Default(ctx, ev)
}

func (ts *TaskStateV2) Emit(ctx context.Context, ev event.Event, next EventHandler) {
	ts.task.LastEvent = ev
	ts.task.LastEmitCtx = ContextWithNextStep(ctx, &NextStep{Handler: ts})
	ts.task.Next = next
	ts.task.Emitter.Emit(ts.task.LastEmitCtx, ev)
}

func (ts *TaskStateV2) Reset() {
	ts.task.Next = nil
}
