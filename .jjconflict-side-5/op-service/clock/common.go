package clock

import (
	"context"
	"time"
)

func sleepCtx(ctx context.Context, d time.Duration, c Clock) error {
	timer := c.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.Ch():
		return nil
	}
}
