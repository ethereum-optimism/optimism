package async

import (
	"context"
	"errors"
	"fmt"
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
	// threshold is reported as the gossip timestamp threshold.
	threshold time.Duration
}

func (m *mockNetwork) GossipTimestampThreshold() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.threshold
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
	mu       sync.Mutex
	errors   int
	dropped  int
	queueLen int
	maxQueue int
	delays   []time.Duration
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

func (m *mockMetrics) RecordPublishQueueLen(length int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueLen = length
	if length > m.maxQueue {
		m.maxQueue = length
	}
}

func (m *mockMetrics) RecordPublishDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays = append(m.delays, d)
}

func (m *mockMetrics) counts() (errs, dropped int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errors, m.dropped
}

func (m *mockMetrics) reportedQueueLen() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queueLen
}

func (m *mockMetrics) reportedMaxQueueLen() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxQueue
}

func (m *mockMetrics) reportedDelays() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]time.Duration(nil), m.delays...)
}

func envelopeAt(n uint64) *eth.ExecutionPayloadEnvelope {
	return envelopeAged(n, 0)
}

// envelopeAged builds a block whose timestamp is `age` in the past.
func envelopeAged(n uint64, age time.Duration) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: hexutil.Uint64(n),
			Timestamp:   hexutil.Uint64(time.Now().Add(-age).Unix()),
		},
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

	require.Zero(t, metrics.reportedQueueLen(), "queue length is reported back to zero")
	require.Len(t, metrics.reportedDelays(), len(want), "one publish delay per block")
	for _, d := range metrics.reportedDelays() {
		require.Positive(t, d, "a publish delay is a real elapsed duration")
	}
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
	require.Equal(t, maxPublishQueue, metrics.reportedMaxQueueLen(),
		"the reported depth never exceeds the bound")
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

// TestAsyncGossiperDropsBlocksPastThreshold pins the fix for the fixed point
// observed on a live devnet: a block already older than the gossip timestamp
// threshold is rejected by every peer, so it must be dropped without spending a
// publish attempt at the head of the queue.
func TestAsyncGossiperDropsBlocksPastThreshold(t *testing.T) {
	net := &mockNetwork{threshold: 60 * time.Second}
	p, metrics := newTestGossiper(t, net)

	stale := envelopeAged(1, 90*time.Second)
	fresh := envelopeAt(2)
	p.Gossip(stale)
	p.Gossip(fresh)

	require.Eventually(t, func() bool {
		return p.queueLen() == 0
	}, 5*time.Second, time.Millisecond)

	published := net.published()
	require.Len(t, published, 1, "the stale block must not be offered to peers at all")
	require.Same(t, fresh, published[0], "the fresh block still goes out")

	errs, dropped := metrics.counts()
	require.Zero(t, errs, "a block dropped before publishing is not a publish error")
	require.Equal(t, 1, dropped, "the stale block is counted as never reaching peers")
}

// TestAsyncGossiperRecoversFromSustainedOutage reproduces the devnet failure
// directly. A queue full of blocks that all aged past the threshold during an
// outage must drain in one pass once publishing works again, rather than
// settling into a state where every block reaching the head is already stale.
func TestAsyncGossiperRecoversFromSustainedOutage(t *testing.T) {
	net := &mockNetwork{threshold: 60 * time.Second}
	p, metrics := newTestGossiper(t, net)

	// Fill the queue with blocks sealed during a two-minute outage.
	for i := uint64(0); i < maxPublishQueue; i++ {
		p.Gossip(envelopeAged(i, 120*time.Second))
	}
	// Then one fresh block, as the sequencer would seal after recovery.
	fresh := envelopeAt(maxPublishQueue)
	p.Gossip(fresh)

	require.Eventually(t, func() bool {
		return p.queueLen() == 0
	}, 5*time.Second, 2*time.Millisecond)

	published := net.published()
	require.Len(t, published, 1, "no stale block is published")
	require.Same(t, fresh, published[0], "gossip recovers for fresh blocks")

	_, dropped := metrics.counts()
	require.Positive(t, dropped, "the stale backlog is counted as dropped")
}

// TestAsyncGossiperDropsPermanentFailureImmediately checks the classification:
// a failure no retry can fix costs one attempt, not maxPublishAttempts.
func TestAsyncGossiperDropsPermanentFailureImmediately(t *testing.T) {
	net := &mockNetwork{err: fmt.Errorf("%w: topic closed", ErrPermanentPublish)}
	p, metrics := newTestGossiper(t, net)

	p.Gossip(envelopeAt(1))
	require.Eventually(t, func() bool {
		_, dropped := metrics.counts()
		return dropped == 1
	}, 5*time.Second, time.Millisecond)

	errs, _ := metrics.counts()
	require.Equal(t, 1, errs, "one attempt, not maxPublishAttempts")
	require.Len(t, net.published(), 1, "publish was tried exactly once")
}

// TestAsyncGossiperKeepsTimeoutsRetryable is the guard against classifying too
// broadly. Topic.Publish surfaces the caller's context error, so a publish that
// merely hit publishTimeout must still be retried.
func TestAsyncGossiperKeepsTimeoutsRetryable(t *testing.T) {
	net := &mockNetwork{err: context.DeadlineExceeded}
	p, metrics := newTestGossiper(t, net)

	p.Gossip(envelopeAt(1))
	require.Eventually(t, func() bool {
		errs, _ := metrics.counts()
		return errs >= maxPublishAttempts
	}, 5*time.Second, time.Millisecond)

	// The distinguishing fact: a timeout spends the whole retry ladder, where a
	// permanent failure would have been dropped after a single attempt.
	errs, _ := metrics.counts()
	require.Equal(t, maxPublishAttempts, errs, "a timed-out publish is retried, not classified permanent")
	require.Len(t, net.published(), maxPublishAttempts, "every attempt reached the network")
}

// TestAsyncGossiperUnboundedThresholdPublishesAnyAge covers p2p being disabled,
// where the threshold is reported as zero.
func TestAsyncGossiperUnboundedThresholdPublishesAnyAge(t *testing.T) {
	net := &mockNetwork{threshold: 0}
	p, _ := newTestGossiper(t, net)

	old := envelopeAged(1, time.Hour)
	p.Gossip(old)
	require.Eventually(t, func() bool {
		return len(net.published()) == 1
	}, 5*time.Second, time.Millisecond)
	require.Same(t, old, net.published()[0], "no age bound when the threshold is zero")
}

// TestAsyncGossiperStopAfterContextCancel pins the invariant the CI deadlock in
// TestSupernodeVerifierELSyncsFromCold violated: OpNode.Stop cancels the driver
// context and then calls Stop, so the publish loop may already have exited, and
// Stop must still return.
//
// It does not reproduce that failure on the old code, because it waits for the
// loop to exit before calling Stop and so steps past the window where Stop saw
// running still true. The deterministic reproduction is
// TestAsyncGossiperStopCancelsInFlightPublish, which fails on the old
// signalling; TestAsyncGossiperStopRacesContextCancel covers the window itself,
// probabilistically.
func TestAsyncGossiperStopAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := NewAsyncGossiper(ctx, &mockNetwork{}, log.New(), &mockMetrics{})
	p.Start()
	require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)

	// The loop exits by itself, exactly as it does on driver shutdown.
	cancel()
	require.Eventually(t, func() bool { return !p.running.Load() }, 5*time.Second, time.Millisecond)

	done := make(chan struct{})
	go func() { defer close(done); p.Stop() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop deadlocked after the publish loop had already exited")
	}
}

// TestAsyncGossiperStopRacesContextCancel covers the narrow window the CI run
// actually lost: the loop exits and Stop is called concurrently, so neither
// ordering may block.
func TestAsyncGossiperStopRacesContextCancel(t *testing.T) {
	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		p := NewAsyncGossiper(ctx, &mockNetwork{}, log.New(), &mockMetrics{})
		p.Start()
		require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)

		done := make(chan struct{})
		go func() { defer close(done); p.Stop() }()
		cancel() // races the Stop above

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Stop deadlocked on iteration %d", i)
		}
	}
}

// TestAsyncGossiperStopCancelsInFlightPublish checks that Stop does not wait out
// publishTimeout on a publish that is already hanging.
func TestAsyncGossiperStopCancelsInFlightPublish(t *testing.T) {
	gate := make(chan struct{})
	defer close(gate)
	net := &mockNetwork{gate: gate}
	p := NewAsyncGossiper(context.Background(), net, log.New(), &mockMetrics{})
	p.Start()
	require.Eventually(t, p.running.Load, 5*time.Second, time.Millisecond)

	p.Gossip(envelopeAt(1))
	require.Eventually(t, func() bool { return len(net.published()) == 1 }, 5*time.Second, time.Millisecond)

	start := time.Now()
	done := make(chan struct{})
	go func() { defer close(done); p.Stop() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop blocked on an in-flight publish")
	}
	require.Less(t, time.Since(start), publishTimeout,
		"Stop must cancel the publish rather than wait out publishTimeout")
}
