package event

import (
	"context"
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
	Task(handler Handler, opts ...HandlerOption)
	// Watch registers an event-handler to receive events, just to observe.
	Watch(handler Handler, opts ...HandlerOption)
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

type handlerEntry struct {
	h   Handler
	cfg *HandlerConfig
}

type handlersBundle struct {
	entries []*handlerEntry
}

func (b *handlersBundle) AddHandler(h Handler, cfg *HandlerConfig) {
	b.entries = append(b.entries, &handlerEntry{
		h:   h,
		cfg: cfg,
	})
}

func (b *handlersBundle) processEvent(ctx context.Context, ev Event) {
	for _, h := range b.entries {
		if !h.cfg.Filter(ctx, ev) {
			continue
		}
		h.h.Serve(ctx, ev)
	}
}

// basicBus is a minimal implementation of Bus,
// for simple executor implementations to build on.
type basicBus struct {
	name string

	handlers map[reflect.Type]*handlersBundle

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

func (b *basicBus) Task(handler Handler, opts ...HandlerOption) {
	cfg := &HandlerConfig{}
	cfg.ApplyOpts(opts...)
	typ := handler.EventType()
	if _, ok := b.handlers[typ]; !ok {
		b.handlers[typ] = &handlersBundle{}
	}
	b.handlers[typ].AddHandler(handler, cfg)
}

func (b *basicBus) Watch(handler Handler, opts ...HandlerOption) {
	cfg := &HandlerConfig{}
	cfg.ApplyOpts(opts...)
	typ := handler.EventType()
	if _, ok := b.handlers[typ]; !ok {
		b.handlers[typ] = &handlersBundle{}
	}
	b.handlers[typ].AddHandler(handler, cfg)
}

func (b *basicBus) Emitter() Emitter {
	return b.emitter
}

func (b *basicBus) Close() {
	b.closer()
}

func (b *basicBus) processEvent(ctx context.Context, ev Event) {
	// apply the event to all handlers of the matching type
	if b, ok := b.handlers[reflect.TypeOf(ev)]; ok {
		b.processEvent(ctx, ev)
	}
	// apply the event to all handlers that just do catch-all typing
	if b, ok := b.handlers[reflect.TypeFor[Event]()]; ok {
		b.processEvent(ctx, ev)
	}
}
