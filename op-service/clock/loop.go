package clock

import (
	"context"
	"sync"
	"time"
)

// LoopFn is a simple ticker-loop with io.Closer support.
// Note that ticks adapt; slow function calls may result in lost ticks.
type LoopFn struct {
	ctx    context.Context
	cancel context.CancelFunc

	ticker  Ticker
	fn      func(ctx context.Context)
	onClose func() error

	wg sync.WaitGroup
}

// Close cancels the context of the ongoing function call, waits for the call to complete, and cancels further calls.
// Close is safe to call again or concurrently. The onClose callback will be called for each Close call.
func (lf *LoopFn) Close() error {
	lf.cancel()  // stop any ongoing function call, and close the main loop
	lf.wg.Wait() // wait for completion
	if lf.onClose != nil {
		return lf.onClose() // optional: user can specify function to close resources with
	}
	return nil
}

func (lf *LoopFn) work() {
	defer lf.wg.Done()
	defer lf.ticker.Stop() // clean up the timer
	for {
		select {
		case <-lf.ctx.Done():
			return
		case <-lf.ticker.Ch():
			ctx, cancel := context.WithCancel(lf.ctx)
			func() {
				defer cancel()
				lf.fn(ctx)
			}()
		}
	}
}

// NewLoopFn creates a periodic function call, which can be closed,
// with an optional onClose callback to clean up resources.
func NewLoopFn(clock Clock, fn func(ctx context.Context), onClose func() error, interval time.Duration) *LoopFn {
	ctx, cancel := context.WithCancel(context.Background())
	lf := &LoopFn{
		ctx:     ctx,
		cancel:  cancel,
		fn:      fn,
		ticker:  clock.NewTicker(interval),
		onClose: onClose,
	}
	lf.wg.Add(1)
	go lf.work()
	return lf
}
