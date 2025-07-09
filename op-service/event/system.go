package event

import (
	"fmt"
	"sync"

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

type sysActor struct {
	actor Actor
	bus   Bus
}

// Sys is the canonical implementation of System.
type Sys struct {
	regs     map[string]*sysActor
	regsLock sync.Mutex

	log log.Logger

	executor Executor
}

func NewSystem(log log.Logger, ex Executor) *Sys {
	return &Sys{
		regs:     make(map[string]*sysActor),
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
	cfg.Name = name
	for _, opt := range opts {
		opt(cfg)
	}

	bus := s.executor.NewBus(cfg)
	a := &sysActor{
		actor: actor,
		bus:   bus,
	}
	s.regs[name] = a

	// run setup function of the actor, to make it register handlers etc.
	actor.Events(bus)

	return bus
}

func (s *Sys) Unregister(name string) {
	s.regsLock.Lock()
	defer s.regsLock.Unlock()
	s.unregister(name)
}

func (s *Sys) unregister(name string) {
	r, ok := s.regs[name]
	if !ok {
		return
	}
	// if this was registered as deriver with the executor, then leave the executor
	if r.bus != nil {
		r.bus.Close()
	}
	delete(s.regs, name)
	if cl, ok := r.actor.(Unattacher); ok {
		cl.Unattach()
	}
}

// Stop shuts down the system
// by unregistering all emitters/derivers,
// freeing up executor resources.
func (s *Sys) Stop() {
	s.regsLock.Lock()
	defer s.regsLock.Unlock()
	for _, r := range s.regs {
		s.unregister(r.bus.Name())
	}
}
