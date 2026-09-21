package async

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// mockNetwork records every publish, and can be scripted to fail or to block.
type mockNetwork struct {
	mu   sync.Mutex
	reqs []*eth.ExecutionPayloadEnvelope
	// err, when set, is returned by every publish.
	err error
	// gate, when set, is received from before each publish returns.
	gate chan struct{}
}

func (m *mockNetwork) SignAndPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	m.mu.Lock()
	m.reqs = append(m.reqs, payload)
	gate, err := m.gate, m.err
	m.mu.Unlock()

	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func (m *mockNetwork) published() []*eth.ExecutionPayloadEnvelope {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*eth.ExecutionPayloadEnvelope(nil), m.reqs...)
}

func (m *mockNetwork) setErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

type mockMetrics struct {
	mu      sync.Mutex
	errors  int
	dropped int
}

func (m *mockMetrics) RecordPublishingError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
}

func (m *mockMetrics) RecordDroppedPublish() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dropped++
}

func (m *mockMetrics) counts() (errs, dropped int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errors, m.dropped
}

func envelopeAt(n uint64) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{BlockNumber: hexutil.Uint64(n)},
	}
}

func (p *SimpleAsyncGossiper) queueLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.queue)
}

// newTestGossiper returns a started gossiper with a retry interval short enough
// to keep the suite fast, already registered for shutdown.
func newTestGossiper(t *testing.T, net Network) (*SimpleAsyncGossiper, *mockMetrics) {
	t.Helper()
	m := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), net, log.New(), m)
	p.retryInterval = 5 * time.Millisecond
	p.Start()
	t.Cleanup(p.Stop)
	require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)
	return p, m
}

// TestAsyncGossiperPublishesInSealOrder covers the happy path: every queued
// block reaches the network, in the order it was handed over, and the queue
// drains back to empty.
func TestAsyncGossiperPublishesInSealOrder(t *testing.T) {
	net := &mockNetwork{}
	p, metrics := newTestGossiper(t, net)

	want := make([]*eth.ExecutionPayloadEnvelope, 0, 10)
	for i := uint64(0); i < 10; i++ {
		envelope := envelopeAt(i)
		want = append(want, envelope)
		p.Gossip(envelope)
	}

	require.Eventually(t, func() bool {
		return len(net.published()) == len(want)
	}, 5*time.Second, time.Millisecond)
	require.Equal(t, want, net.published(), "published in seal order")
	require.Zero(t, p.queueLen(), "queue drains in steady state")

	errs, dropped := metrics.counts()
	require.Zero(t, errs)
	require.Zero(t, dropped)
}

// TestAsyncGossiperDoesNotBlockOnPublish is the regression test for #22554: the
// sequencer reaches Gossip on its hot path, so it must never wait for a publish
// — which, with a remote signer, is an HTTPS round-trip.
func TestAsyncGossiperDoesNotBlockOnPublish(t *testing.T) {
	gate := make(chan struct{})
	net := &mockNetwork{gate: gate}
	p, _ := newTestGossiper(t, net)

	// Occupy the publish goroutine with a publish that will not return yet.
	p.Gossip(envelopeAt(1))
	require.Eventually(t, func() bool {
		return len(net.published()) == 1
	}, 5*time.Second, time.Millisecond)

	// With a publish in flight, both exposed calls must still return promptly.
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Gossip(envelopeAt(2))
		p.Clear()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Gossip/Clear blocked while a publish was in flight")
	}

	close(gate)
}

// TestAsyncGossiperRetriesAtFront checks that a failed publish is retried rather
// than skipped, and that its descendants stay behind it: a gap peers cannot
// cross is worse for them than a delay.
func TestAsyncGossiperRetriesAtFront(t *testing.T) {
	net := &mockNetwork{err: errors.New("publish failed")}
	p, metrics := newTestGossiper(t, net)

	first, second := envelopeAt(1), envelopeAt(2)
	p.Gossip(first)
	p.Gossip(second)

	// Until the first block succeeds, only the first block is ever attempted.
	require.Eventually(t, func() bool {
		errs, _ := metrics.counts()
		return errs >= 2
	}, 5*time.Second, time.Millisecond)
	for _, published := range net.published() {
		require.Same(t, first, published, "descendants must not overtake a retrying block")
	}

	net.setErr(nil)
	require.Eventually(t, func() bool {
		return p.queueLen() == 0
	}, 5*time.Second, time.Millisecond)

	published := net.published()
	require.Same(t, first, published[len(published)-2], "first block published before its descendant")
	require.Same(t, second, published[len(published)-1])
}

// TestAsyncGossiperGivesUpAfterMaxAttempts checks the head-of-line bound. A
// block whose publish keeps failing is dropped, counted, and its descendants go
// out — otherwise one permanently unpublishable block stops gossip entirely.
func TestAsyncGossiperGivesUpAfterMaxAttempts(t *testing.T) {
	net := &mockNetwork{err: errors.New("publish failed")}
	p, metrics := newTestGossiper(t, net)

	p.Gossip(envelopeAt(1))

	require.Eventually(t, func() bool {
		_, dropped := metrics.counts()
		return dropped == 1
	}, 5*time.Second, time.Millisecond)

	errs, _ := metrics.counts()
	require.Equal(t, maxPublishAttempts, errs, "tried exactly maxPublishAttempts times")
	require.Zero(t, p.queueLen(), "the doomed block is no longer at the front")

	// Gossip is alive: a later block still publishes.
	net.setErr(nil)
	next := envelopeAt(2)
	p.Gossip(next)
	require.Eventually(t, func() bool {
		published := net.published()
		return len(published) > 0 && published[len(published)-1] == next
	}, 5*time.Second, time.Millisecond)
}

// TestAsyncGossiperEvictsOldestWhenFull checks the queue bound: under pressure
// the oldest block is evicted, since a block no peer is still waiting for is the
// right one to lose.
func TestAsyncGossiperEvictsOldestWhenFull(t *testing.T) {
	gate := make(chan struct{})
	net := &mockNetwork{gate: gate}
	p, metrics := newTestGossiper(t, net)

	// Wedge the publish goroutine so nothing drains while the queue fills.
	p.Gossip(envelopeAt(0))
	require.Eventually(t, func() bool {
		return len(net.published()) == 1
	}, 5*time.Second, time.Millisecond)

	for i := uint64(1); i <= maxPublishQueue; i++ {
		p.Gossip(envelopeAt(i))
	}

	require.Equal(t, maxPublishQueue, p.queueLen(), "queue is bounded")
	_, dropped := metrics.counts()
	require.Positive(t, dropped, "eviction is counted")

	close(gate)
}

// TestAsyncGossiperClearDropsQueue checks Clear, which the sequencer uses when
// the chain the queued blocks extend is not the one it is building on.
func TestAsyncGossiperClearDropsQueue(t *testing.T) {
	gate := make(chan struct{})
	net := &mockNetwork{gate: gate}
	p, _ := newTestGossiper(t, net)

	p.Gossip(envelopeAt(0))
	require.Eventually(t, func() bool {
		return len(net.published()) == 1
	}, 5*time.Second, time.Millisecond)
	for i := uint64(1); i < 5; i++ {
		p.Gossip(envelopeAt(i))
	}

	p.Clear()
	require.Zero(t, p.queueLen())

	// The in-flight publish resolves against an emptied queue without panicking,
	// and nothing further is published.
	close(gate)
	require.Never(t, func() bool {
		return len(net.published()) > 1
	}, 200*time.Millisecond, 10*time.Millisecond)
}

// TestAsyncGossiperStopDuringRetryPause checks that Stop does not wait out the
// retry interval, which is a second in production.
func TestAsyncGossiperStopDuringRetryPause(t *testing.T) {
	net := &mockNetwork{err: errors.New("publish failed")}
	m := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), net, log.New(), m)
	p.retryInterval = time.Hour // a pause Stop must not wait for
	p.Start()

	p.Gossip(envelopeAt(1))
	require.Eventually(t, func() bool {
		errs, _ := m.counts()
		return errs == 1
	}, 5*time.Second, time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Stop()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop waited for the retry interval")
	}
	require.Eventually(t, func() bool { return !p.running.Load() }, 5*time.Second, time.Millisecond)
}

// TestAsyncGossiperStopIsIdempotent covers Stop on a gossiper that was never
// started, and a Start/Stop cycle being repeatable.
func TestAsyncGossiperStopIsIdempotent(t *testing.T) {
	p := NewAsyncGossiper(context.Background(), &mockNetwork{}, log.New(), &mockMetrics{})
	p.Stop() // never started: must not block

	p.Start()
	require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)
	p.Stop()
	require.Eventually(t, func() bool { return !p.running.Load() }, 5*time.Second, time.Millisecond)

	p.Start()
	require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)
	p.Stop()
}

// TestAsyncGossiperConcurrentAccess hammers the exposed calls under -race.
func TestAsyncGossiperConcurrentAccess(t *testing.T) {
	net := &mockNetwork{}
	p, _ := newTestGossiper(t, net)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(base uint64) {
			defer wg.Done()
			for j := uint64(0); j < 50; j++ {
				p.Gossip(envelopeAt(base*50 + j))
				if j%10 == 0 {
					p.Clear()
				}
			}
		}(uint64(i))
	}
	wg.Wait()
}
