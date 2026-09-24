package async

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type publishFuncNetwork func(context.Context, *eth.ExecutionPayloadEnvelope) error

func (f publishFuncNetwork) SignAndPublishL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	return f(ctx, envelope)
}

func (publishFuncNetwork) GossipTimestampThreshold() time.Duration { return time.Minute }

func TestAsyncGossiperClearCancelsAndAllowsReenqueue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		var firstCtx context.Context
		attempts := 0
		net := publishFuncNetwork(func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
			attempts++
			if attempts == 1 {
				firstCtx = ctx
				// Model a network call that finishes after cancellation, including
				// one that reports success despite that cancellation.
				<-gate
			}
			return nil
		})
		metrics := &mockMetrics{}
		p := NewAsyncGossiper(t.Context(), net, log.New(), metrics)
		p.Start()
		defer p.Stop()

		envelope := envelopeAt(1)
		p.Gossip(envelope)
		synctest.Wait()
		require.Equal(t, 1, attempts)
		require.NoError(t, firstCtx.Err())

		p.Clear() // must return without waiting for gate
		require.ErrorIs(t, firstCtx.Err(), context.Canceled)
		require.Zero(t, metrics.reportedQueueLen())
		// Reusing the envelope must create a distinct queue entry: the old
		// attempt's successful return must not dequeue this new publication.
		p.Gossip(envelope)
		require.Equal(t, 1, metrics.reportedQueueLen())
		close(gate)
		synctest.Wait()

		require.Equal(t, 2, attempts)
		require.Zero(t, p.queueLen())
		require.Zero(t, metrics.reportedQueueLen())
		errs, dropped := metrics.counts()
		require.Zero(t, errs, "invalidation is not a publishing failure")
		require.Equal(t, 1, dropped, "the invalidated entry is counted exactly once")
		require.Len(t, metrics.reportedDelays(), 1, "only the current entry contributes a delay")
	})
}

func TestAsyncGossiperEvictionDoesNotStarvePublisher(t *testing.T) {
	for _, clearAfterEviction := range []bool{false, true} {
		name := "overflow alone"
		if clearAfterEviction {
			name = "invalidation after overflow"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				gate := make(chan struct{})
				var firstCtx context.Context
				var published []*eth.ExecutionPayloadEnvelope
				net := publishFuncNetwork(func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
					if firstCtx == nil {
						firstCtx = ctx
						select {
						case <-gate:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					published = append(published, envelope)
					return nil
				})
				metrics := &mockMetrics{}
				p := NewAsyncGossiper(t.Context(), net, log.New(), metrics)
				p.Start()
				defer p.Stop()

				p.Gossip(envelopeAt(0))
				synctest.Wait()
				for i := uint64(1); i <= maxPublishQueue; i++ {
					p.Gossip(envelopeAt(i))
				}
				// Canceling on each eviction would starve a signer that is slower
				// than block production once the queue fills, despite it working.
				require.NoError(t, firstCtx.Err())
				wantPublished, wantDropped := maxPublishQueue+1, 1
				if clearAfterEviction {
					p.Clear()
					require.ErrorIs(t, firstCtx.Err(), context.Canceled,
						"Clear must still cancel an attempt whose entry was evicted")
					wantPublished, wantDropped = 0, maxPublishQueue+1
				} else {
					close(gate)
				}
				synctest.Wait()

				require.Len(t, published, wantPublished)
				require.Zero(t, p.queueLen())
				require.Zero(t, metrics.reportedQueueLen())
				errs, dropped := metrics.counts()
				require.Zero(t, errs)
				require.Equal(t, wantDropped, dropped)
			})
		})
	}
}

func TestAsyncGossiperDoesNotStartInvalidatedEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := &mockNetwork{}
		metrics := &mockMetrics{}
		p := NewAsyncGossiper(t.Context(), net, log.New(), metrics)
		envelope := envelopeAt(1)
		p.Gossip(envelope)
		// The loop selected this entry, but Clear wins before publish registers
		// its cancellation. Even the same envelope enqueued again cannot revive it.
		selected := p.queue[0]
		p.Clear()
		p.Gossip(envelope)
		require.ErrorIs(t, p.publish(t.Context(), selected), context.Canceled)
		require.Empty(t, net.published())
		require.Equal(t, 1, p.queueLen())

		p.Start()
		defer p.Stop()
		synctest.Wait()
		require.Equal(t, []*eth.ExecutionPayloadEnvelope{envelope}, net.published())
		require.Zero(t, metrics.reportedQueueLen())
	})
}
