package scheduler

import (
	"context"
	"sync"
)

type worker struct {
	in     <-chan job
	out    chan<- job
	cancel func()
	wg     sync.WaitGroup
}

func (w *worker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.wg.Add(1)
	go w.loop(ctx)
}

func (w *worker) Stop() {
	w.cancel()
	w.wg.Wait()
}

func (w *worker) loop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-w.in:
			j.resolved = j.player.ProgressGame(ctx)
			w.out <- j
		}
	}
}
