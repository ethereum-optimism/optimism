package scheduler

import (
	"context"
	"sync"
)

func runWorker(ctx context.Context, in <-chan job, out chan<- job, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-in:
			j.resolved = j.player.ProgressGame(ctx)
			out <- j
		}
	}
}
