package event

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Registry interface {
	// Register registers a named actor, optionally processing events itself:
	// the actor is not required to register events, not all registrants have to process events.
	// A non-nil actor may implement Actor to setup event handlers,
	// before the actor itself becomes executable.
	// A non-nil actor may implement Unattacher to close resources upon being unregistered.
	Register(name string, deriver Actor, opts ...RegisterOption) Bus
	// Unregister removes a named emitter,
	// also removing it from the set of events-receiving derivers (if registered with non-nil deriver).
	// If the originally attached Deriver implements Unattacher it will be notified.
	Unregister(name string) (old Emitter)
}

type System interface {
	Registry
	// AddTracer registers a tracer to capture all event deriver/emitter work. It runs until RemoveTracer is called.
	// Duplicate tracers are allowed.
	AddTracer(t Tracer)
	// RemoveTracer removes a tracer. This is a no-op if the tracer was not previously added.
	// It will remove all added duplicates of the tracer.
	RemoveTracer(t Tracer)
	// Stop shuts down the System by un-registering all derivers/emitters.
	Stop()
}

// Unattacher is called when a deriver/emitter is unregistered from the system.
type Unattacher interface {
	Unattach()
}

type AnnotatedEvent struct {
	Ctx                 context.Context // Ctx passed in via Emit, and provided via executor to OnEvent handlers
	Event               Event
	EmitContext         uint64   // uniquely identifies the emission of the event, useful for debugging and creating diagrams
	EmitPriority        Priority // how important the emitter is, higher is more important
	PostProcessCallback func()   // callback to be called after the event is processed by all derivers
}

func (e AnnotatedEvent) Equals(other AnnotatedEvent) bool {
	return e.Event == other.Event && e.EmitContext == other.EmitContext && e.EmitPriority == other.EmitPriority
}

// systemActor is a deriver and/or emitter, registered in System with a name.
// If deriving, the actor is added as Executable to the Executor of the System.
type systemActor struct {
	name string
	sys  *Sys

	// To manage the execution peripherals, like rate-limiting, of this deriver
	ctx    context.Context
	cancel context.CancelFunc

	limiter *rate.Limiter

	actor         Actor
	leaveExecutor func()

	// 0 if event does not originate from Deriver-handling of another event
	currentEvent uint64

	// How important this actor is as emitter. Higher is more important.
	// Emitted events from actors with a higher emit priority
	// will be prioritized over other queued up events.
	emitPriority Priority

	// handlers to run
	handlers map[reflect.Type]*handlersBundle
}

func newSystemActor(name string, actor Actor, sys *Sys, cfg *RegisterConfig) *systemActor {
	ctx, cancel := context.WithCancel(context.Background())
	r := &systemActor{
		name:   name,
		actor:  actor,
		sys:    sys,
		ctx:    ctx,
		cancel: cancel,
		// prioritize the outgoing messages
		emitPriority: cfg.Emitter.Priority,
	}
	if cfg.Emitter.Limiting {
		r.limiter = rate.NewLimiter(cfg.Emitter.Rate, cfg.Emitter.Burst)
	}
	return r
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
func (b *handlersBundle) Serve(ctx context.Context, ev Event) {
	for _, h := range b.entries {
		if !h.cfg.Filter(ctx, ev) {
			continue
		}
		h.h.Serve(ctx, ev)
	}
}

type handlerEntry struct {
	h   Handler
	cfg *HandlerConfig
}

func (r *systemActor) Handle(handler Handler, opts ...HandlerOption) {
	cfg := &HandlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	k := handler.EventType()
	if _, ok := r.handlers[k]; !ok {
		r.handlers[k] = &handlersBundle{}
	}
	handlers := r.handlers[k]
	handlers.AddHandler(handler, cfg)
}

func (r *systemActor) Emitter() Emitter {
	return r
}

var _ Bus = (*systemActor)(nil)

// Emit is called by the end-user
func (r *systemActor) Emit(ctx context.Context, ev Event) error {
	if ctx == nil {
		return fmt.Errorf("emitter %s must provide a context with the emitted event %s", r.name, ev.String())
	}
	if e := r.ctx.Err(); e != nil {
		return e
	}
	if r.limiter != nil {
		r.sys.recordRateLimited(r.name, r.currentEvent)
		if err := r.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return r.sys.emit(r.name, r.currentEvent, ctx, ev, r.emitPriority)
}

// TODO: refactor this to yield every handler instead,
// so that the executor can match and run each handler individually.

// RunEvent is called by the events executor.
// While different things may execute in parallel, only one event is executed per entry at a time.
func (r *systemActor) RunEvent(ev AnnotatedEvent) {
	if r.actor == nil {
		return
	}
	if r.ctx.Err() != nil {
		return
	}
	if r.sys.abort.Load() && !Is[CriticalErrorEvent](ev.Event) {
		// if aborting, and not the CriticalErrorEvent itself, then do not process the event
		return
	}

	prev := r.currentEvent
	start := time.Now()
	r.currentEvent = r.sys.recordDerivStart(r.name, ev, start)
	typ := reflect.TypeOf(ev.Event)
	// TODO fix legacy metric
	effect := false
	// apply the event to all handlers of the matching type
	if b, ok := r.handlers[typ]; ok {
		b.Serve(ev.Ctx, ev.Event)
		effect = true
	}
	// apply the event to all handlers that just do catch-all typing
	if b, ok := r.handlers[reflect.TypeFor[Event]()]; ok {
		b.Serve(ev.Ctx, ev.Event)
		effect = true
	}
	elapsed := time.Since(start)
	r.sys.recordDerivEnd(r.name, ev, r.currentEvent, start, elapsed, effect)
	r.currentEvent = prev
}

// Sys is the canonical implementation of System.
type Sys struct {
	regs     map[string]*systemActor
	regsLock sync.Mutex

	log log.Logger

	executor Executor

	// used to generate a unique id for each event deriver processing call.
	derivContext atomic.Uint64
	// used to generate a unique id for each event-emission.
	emitContext atomic.Uint64

	tracers     []Tracer
	tracersLock sync.RWMutex

	// if true, no events may be processed, except CriticalError itself
	abort atomic.Bool
}

func NewSystem(log log.Logger, ex Executor) *Sys {
	return &Sys{
		regs:     make(map[string]*systemActor),
		executor: ex,
		log:      log,
	}
}

func (s *Sys) Register(name string, actor Actor, opts ...RegisterOption) Bus {
	s.regsLock.Lock()
	defer s.regsLock.Unlock()

	if _, ok := s.regs[name]; ok {
		panic(fmt.Errorf("a deriver/emitter with name %q already exists", name))
	}

	cfg := defaultRegisterConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	r := newSystemActor(name, actor, s, cfg)

	s.regs[name] = r

	// run setup function of the actor, to make it register handlers etc.
	actor.Events(r)

	return r
}

func (s *Sys) Unregister(name string) (previous Emitter) {
	s.regsLock.Lock()
	defer s.regsLock.Unlock()
	return s.unregister(name)
}

func (s *Sys) unregister(name string) (previous Emitter) {
	r, ok := s.regs[name]
	if !ok {
		return nil
	}
	r.cancel()
	// if this was registered as deriver with the executor, then leave the executor
	if r.leaveExecutor != nil {
		r.leaveExecutor()
	}
	delete(s.regs, name)
	if cl, ok := r.actor.(Unattacher); ok {
		cl.Unattach()
	}
	return r
}

// Stop shuts down the system
// by unregistering all emitters/derivers,
// freeing up executor resources.
func (s *Sys) Stop() {
	s.regsLock.Lock()
	defer s.regsLock.Unlock()
	for _, r := range s.regs {
		s.unregister(r.name)
	}
}

func (s *Sys) AddTracer(t Tracer) {
	s.tracersLock.Lock()
	defer s.tracersLock.Unlock()
	s.tracers = append(s.tracers, t)
}

func (s *Sys) RemoveTracer(t Tracer) {
	s.tracersLock.Lock()
	defer s.tracersLock.Unlock()
	// We are not removing tracers often enough to optimize the deletion;
	// instead we prefer fast and simple tracer iteration during regular operation.
	s.tracers = slices.DeleteFunc(s.tracers, func(v Tracer) bool {
		return v == t
	})
}

// recordDeriv records that the deriver by name [deriv] is processing event [ev].
// This returns a unique integer (during lifetime of Sys), usable as ID to reference processing.
func (s *Sys) recordDerivStart(name string, ev AnnotatedEvent, startTime time.Time) uint64 {
	derivContext := s.derivContext.Add(1)

	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnDeriveStart(name, ev, derivContext, startTime)
	}

	return derivContext
}

func (s *Sys) recordDerivEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnDeriveEnd(name, ev, derivContext, startTime, duration, effect)
	}
}

func (s *Sys) recordRateLimited(name string, derivContext uint64) {
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	s.log.Warn("Event-system emitter component was rate-limited", "emitter", name)
	for _, t := range s.tracers {
		t.OnRateLimited(name, derivContext)
	}
}

func (s *Sys) recordAfterProcessed(evtype string) {
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnAfterProcessed(evtype)
	}
}

func (s *Sys) recordEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnEmit(name, ev, derivContext, emitTime)
	}
}

// emit an event [ev] during the derivation of another event, referenced by derivContext.
// If the event was emitted not as part of deriver event execution, then the derivContext is 0.
// The name of the emitter is provided to further contextualize the event.
func (s *Sys) emit(name string, derivContext uint64, ctx context.Context, ev Event, emitPriority Priority) error {
	emitContext := s.emitContext.Add(1)
	annotated := AnnotatedEvent{
		Ctx:          ctx,
		Event:        ev,
		EmitContext:  emitContext,
		EmitPriority: emitPriority,
		PostProcessCallback: func() {
			s.recordAfterProcessed(ev.String())
		},
	}

	// As soon as anything emits a critical event,
	// make the system aware, before the executor event schedules it for processing.
	if Is[CriticalErrorEvent](ev) {
		s.abort.Store(true)
	}

	emitTime := time.Now()
	s.recordEmit(name, annotated, derivContext, emitTime)
	return s.executor.Enqueue(annotated)
}
