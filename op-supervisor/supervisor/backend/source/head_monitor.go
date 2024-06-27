package source

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type HeadMonitorClient interface {
	eth.NewHeadSource
	eth.L1BlockRefsSource
}

type HeadChangeCallback interface {
	OnNewUnsafeHead(ctx context.Context, block eth.L1BlockRef)
	OnNewSafeHead(ctx context.Context, block eth.L1BlockRef)
	OnNewFinalizedHead(ctx context.Context, block eth.L1BlockRef)
}

// HeadMonitor monitors an L2 chain and sends notifications when the unsafe, safe or finalized head changes.
// Head updates may be coalesced, allowing the head block to skip forward multiple blocks.
// Reorgs are not identified.
type HeadMonitor struct {
	log               log.Logger
	epochPollInterval time.Duration
	rpc               HeadMonitorClient
	callback          HeadChangeCallback

	started      atomic.Bool
	headsSub     event.Subscription
	safeSub      ethereum.Subscription
	finalizedSub ethereum.Subscription
}

func NewHeadMonitor(logger log.Logger, epochPollInterval time.Duration, rpc HeadMonitorClient, callback HeadChangeCallback) *HeadMonitor {
	return &HeadMonitor{
		log:               logger,
		epochPollInterval: epochPollInterval,
		rpc:               rpc,
		callback:          callback,
	}
}

func (h *HeadMonitor) Start() error {
	if !h.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}

	// Keep subscribed to the unsafe head, which changes frequently.
	h.headsSub = event.ResubscribeErr(time.Second*10, func(ctx context.Context, err error) (event.Subscription, error) {
		if err != nil {
			h.log.Warn("Resubscribing after failed heads subscription", "err", err)
		}
		return eth.WatchHeadChanges(ctx, h.rpc, h.callback.OnNewUnsafeHead)
	})
	go func() {
		err, ok := <-h.headsSub.Err()
		if !ok {
			return
		}
		h.log.Error("Heads subscription error", "err", err)
	}()

	// Poll for the safe block and finalized block, which only change once per epoch at most and may be delayed.
	h.safeSub = eth.PollBlockChanges(h.log, h.rpc, h.callback.OnNewSafeHead, eth.Safe,
		h.epochPollInterval, time.Second*10)
	h.finalizedSub = eth.PollBlockChanges(h.log, h.rpc, h.callback.OnNewFinalizedHead, eth.Finalized,
		h.epochPollInterval, time.Second*10)
	h.log.Info("Chain head monitoring started")
	return nil
}

func (h *HeadMonitor) Stop() error {
	if !h.started.CompareAndSwap(true, false) {
		return errors.New("already stopped")
	}

	// stop heads feed
	if h.headsSub != nil {
		h.headsSub.Unsubscribe()
	}
	// stop polling for safe-head changes
	if h.safeSub != nil {
		h.safeSub.Unsubscribe()
	}
	// stop polling for finalized-head changes
	if h.finalizedSub != nil {
		h.finalizedSub.Unsubscribe()
	}
	return nil
}
