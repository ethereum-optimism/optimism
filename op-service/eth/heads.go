package eth

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// HeadSignalFn is used as callback function to accept head-signals
type HeadSignalFn func(ctx context.Context, sig L1BlockRef)

type NewHeadSource interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

// WatchHeadChanges wraps a new-head subscription from NewHeadSource to feed the given Tracker
func WatchHeadChanges(ctx context.Context, src NewHeadSource, fn HeadSignalFn) (ethereum.Subscription, error) {
	headChanges := make(chan *types.Header, 10)
	sub, err := src.SubscribeNewHead(ctx, headChanges)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case header := <-headChanges:
				fn(ctx, L1BlockRef{
					Hash:       header.Hash(),
					Number:     header.Number.Uint64(),
					ParentHash: header.ParentHash,
					Time:       header.Time,
				})
			case err := <-sub.Err():
				return err
			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			}
		}
	}), nil
}

type L1BlockRefsSource interface {
	L1BlockRefByLabel(ctx context.Context, label BlockLabel) (L1BlockRef, error)
}

// PollBlockChanges opens a polling loop to fetch the L1 block reference with the given label,
// on provided interval and with request timeout. Results are returned with provided callback fn,
// which may block to pause/back-pressure polling.
func PollBlockChanges(ctx context.Context, log log.Logger, src L1BlockRefsSource, fn HeadSignalFn,
	label BlockLabel, interval time.Duration, timeout time.Duration) ethereum.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		if interval <= 0 {
			log.Warn("polling of block is disabled", "interval", interval, "label", label)
			<-quit
			return nil
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				reqCtx, reqCancel := context.WithTimeout(ctx, timeout)
				ref, err := src.L1BlockRefByLabel(reqCtx, label)
				reqCancel()
				if err != nil {
					log.Warn("failed to poll L1 block", "label", label, "err", err)
				} else {
					fn(ctx, ref)
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			}
		}
	})
}
