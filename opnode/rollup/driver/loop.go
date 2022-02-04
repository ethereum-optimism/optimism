package driver

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum/log"
)

// keep making many sync steps if we can make sync progress
const hot = time.Millisecond * 30

// check on sync regularly, but prioritize sync triggers with head updates etc.
const cold = time.Second * 8

// at least try every minute to sync, even if things are going well
const max = time.Minute

func NewDriverLoop(ctx context.Context, state StateMachine, log log.Logger, l1Heads <-chan eth.HeadSignal, driver Driver) func(quit <-chan struct{}) error {

	backoff := cold
	syncTicker := time.NewTicker(cold)
	l2HeadPoll := time.NewTicker(time.Second * 14)

	// exponential backoff, add 10% each step, up to max.
	syncBackoff := func() {
		backoff += backoff / 10
		if backoff > max {
			backoff = max
		}
		syncTicker.Reset(backoff)
	}
	syncQuickly := func() {
		syncTicker.Reset(hot)
		backoff = cold
	}
	onL2Update := func() {
		// And we want to slow down requesting the L2 engine for its head (we just changed it ourselves)
		// Request head if we don't successfully change it in the next 14 seconds.
		l2HeadPoll.Reset(time.Second * 14)
	}
	return func(quit <-chan struct{}) error {
		defer syncTicker.Stop()
		defer l2HeadPoll.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			case <-l2HeadPoll.C:
				ctx, cancel := context.WithTimeout(ctx, time.Second*4)
				if state.RequestUpdate(ctx, log, driver) {
					onL2Update()
				}
				cancel()
				continue
			case l1HeadSig := <-l1Heads:
				if state.NotifyL1Head(ctx, log, l1HeadSig, driver) {
					syncQuickly()
				}
				continue
			case <-syncTicker.C:
				// If already synced, or in case of failure, we slow down
				syncBackoff()
				if state.RequestSync(ctx, log, driver) {
					// Successfully stepped toward target. Continue quickly if we are not there yet
					syncQuickly()
					onL2Update()
				}
			}
		}
	}
}
