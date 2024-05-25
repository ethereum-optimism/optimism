package scheduler

import (
	"context"
	"sync"
)

// progressGames accepts jobs from in channel, calls ProgressGame on the job.player and returns the job
// with updated job.resolved via the out channel.
// The loop exits when the ctx is done.  wg.Done() is called when the function returns.
func progressGames(ctx context.Context, in <-chan job, out chan<- job, wg *sync.WaitGroup, threadActive, threadIdle func()) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-in:
			threadActive()
			j.status = j.player.ProgressGame(ctx)
			out <- j
			threadIdle()
		}
	}
}
