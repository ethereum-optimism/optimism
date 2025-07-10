package event

import (
	"context"
	"fmt"
	"reflect"
)

// Bus represents the communication interface that
// is specific to an actor registered in the event system.
// All incoming and outgoing communication on the bus is synchronous within the scope of the bus,
// unless specified otherwise.
type Bus interface {
	Name() string
	String() string
	// Task registers an event-handler to receive events and handle as tasks.
	Task(opts ...HandlerOption)
	// Watch registers an event-handler to receive events, just to observe.
	Watch(opts ...HandlerOption)
	// Emitter returns an emitter that corresponds to the outgoing events of this bus,
	// and that is also attached to the call context that is passed into the event handler.
	Emitter() Emitter
	// Close unregisters all tasks/watches, and stops emitting events.
	// This decouples the bus from the executor.
	Close()
}

// Executor schedules and executes the events.
// The executor may run entirely synchronously,
// or support more advanced execution, like running each bus in parallel.
type Executor interface {
	NewBus(cfg *RegisterConfig) Bus
}

type handlersBundle struct {
	entries []*Handler
}

func (b *handlersBundle) AddHandler(h *Handler) {
	b.entries = append(b.entries, h)
}

func (b *handlersBundle) processEvent(ctx context.Context, ev Event) {
	for _, h := range b.entries {
		h.Fn(ctx, ev)
	}
}

// basicBus is a minimal implementation of Bus,
// for simple executor implementations to build on.
type basicBus struct {
	name string

	tasks   map[HandlerKey]*Handler
	watches map[HandlerKey][]*Handler

	emitter Emitter

	closer func()
}

var _ Bus = (*basicBus)(nil)

func (b *basicBus) Name() string {
	return b.name
}

func (b *basicBus) String() string {
	return b.name
}

func (b *basicBus) Task(opts ...HandlerOption) {
	var h Handler
	h.Apply(opts...)
	if _, ok := b.tasks[h.Key]; ok {
		panic(fmt.Errorf("already have a task handler for key %s", h.Key))
	}
	b.tasks[h.Key] = &h
}

func (b *basicBus) Watch(opts ...HandlerOption) {
	var h Handler
	h.Apply(opts...)
	b.watches[h.Key] = append(b.watches[h.Key], &h)
}

func (b *basicBus) Emitter() Emitter {
	return b.emitter
}

func (b *basicBus) Close() {
	b.closer()
}

func (b *basicBus) processEvent(ctx context.Context, ev Event) {
	domain := DomainFromCtx(ctx)
	fullKey := HandlerKey{
		EventType: reflect.TypeOf(ev),
		Domain:    domain,
	}
	for k := range fullKey.Variants {
		if h, ok := b.tasks[k]; ok {
			h.Fn(ctx, ev)
		}
		for _, h := range b.watches[k] {
			h.Fn(ctx, ev)
		}
	}
}
