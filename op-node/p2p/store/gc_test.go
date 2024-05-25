package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestScheduleGcPeriodically(t *testing.T) {
	var bgTasks sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		// Wait for the gc background process to complete after cancelling the context
		bgTasks.Wait()
	}()
	logger := testlog.Logger(t, log.LvlInfo)
	clock := clock.NewDeterministicClock(time.UnixMilli(5000))

	called := make(chan struct{}, 10)
	action := func() error {
		called <- struct{}{}
		return nil
	}
	waitForGc := func(failMsg string) {
		timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		select {
		case <-timeout.Done():
			t.Fatal(failMsg)
		case <-called:
			require.Len(t, called, 0, "should only run once after gc period")
		}
	}
	startGc(ctx, logger, clock, &bgTasks, action)
	timeout, tCancel := context.WithTimeout(ctx, 10*time.Second)
	defer tCancel()
	require.True(t, clock.WaitForNewPendingTask(timeout), "did not schedule pending GC")

	require.Len(t, called, 0, "should not run immediately")

	clock.AdvanceTime(gcPeriod)
	waitForGc("should run gc after first time period")

	clock.AdvanceTime(gcPeriod)
	waitForGc("should run gc again after second time period")
}
