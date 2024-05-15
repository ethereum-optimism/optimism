package monitor

import (
	"context"
	"time"
)

func schedule(ctx context.Context, interval time.Duration, handler func(ctx context.Context)) {
	go func() {
		for {
			timer := time.NewTimer(interval)
			handler(ctx)

			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}
