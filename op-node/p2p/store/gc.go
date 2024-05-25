package store

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
)

const (
	gcPeriod = 2 * time.Hour
)

type gcAction func() error

func startGc(ctx context.Context, logger log.Logger, clock clock.Clock, bgTasks *sync.WaitGroup, action gcAction) {
	bgTasks.Add(1)
	go func() {
		defer bgTasks.Done()

		gcTimer := clock.NewTicker(gcPeriod)
		defer gcTimer.Stop()

		for {
			select {
			case <-gcTimer.Ch():
				if err := action(); err != nil {
					logger.Warn("GC failed", "err", err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}
