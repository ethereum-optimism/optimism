package event

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
)

type ParallelExec struct {
	workers []*worker
	mu      sync.RWMutex
}

var _ Executor = (*ParallelExec)(nil)

func NewParallelExec() *ParallelExec {
	return &ParallelExec{}
}

func (p *ParallelExec) Add(d Executable, opts *ExecutorOpts) (leaveExecutor func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	w := newWorker(p, d, opts)
	p.workers = append(p.workers, w)
	return w.leave
}

func (p *ParallelExec) remove(w *worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Linear search to delete is fine,
	// since we delete much less frequently than we process events with these.
	for i, v := range p.workers {
		if v == w {
			p.workers = slices.Delete(p.workers, i, i+1)
			return
		}
	}
}

func (p *ParallelExec) Enqueue(ev AnnotatedEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, w := range p.workers {
		w.enqueue(ev) // this will block if its capacity is full, providing back-pressure to the Enqueue caller
	}
	return nil
}

type worker struct {
	// ctx signals when the worker is exiting.
	// No additional events will be accepted after cancellation.
	ctx    context.Context
	cancel context.CancelFunc

	// closed as channel is closed upon exit of the run loop
	closed chan struct{}

	// ingress is the buffered channel of events to process
	ingress chan AnnotatedEvent

	// d is the underlying executable to process events on
	d Executable

	// p is a reference to the ParallelExec that owns this worker.
	// The worker removes itself from this upon leaving.
	p atomic.Pointer[ParallelExec]
}

func newWorker(p *ParallelExec, d Executable, opts *ExecutorOpts) *worker {
	ctx, cancel := context.WithCancel(context.Background())
	w := &worker{
		ctx:     ctx,
		cancel:  cancel,
		closed:  make(chan struct{}),
		ingress: make(chan AnnotatedEvent, opts.Capacity),
		d:       d,
	}
	w.p.Store(p)
	go w.run()
	return w
}

func (w *worker) enqueue(ev AnnotatedEvent) {
	select {
	case <-w.ctx.Done():
	case w.ingress <- ev:
	}
}

func (w *worker) leave() {
	w.cancel()
	if old := w.p.Swap(nil); old != nil {
		// remove from worker pool
		old.remove(w)
	}
	// wait for run loop to exit
	<-w.closed
}

func (w *worker) run() {
	for {
		select {
		case <-w.ctx.Done():
			close(w.closed)
			return
		case ev := <-w.ingress:
			w.d.RunEvent(ev)
		}
	}
}
