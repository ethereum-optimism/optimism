package async

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockNetwork struct {
	reqs []*eth.ExecutionPayloadEnvelope
}

func (m *mockNetwork) SignAndPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	m.reqs = append(m.reqs, payload)
	return nil
}

type mockMetrics struct {
	mu       sync.Mutex
	dropped  int
	queueLen int
	maxLen   int
}

func (m *mockMetrics) RecordPublishingError() {}

func (m *mockMetrics) RecordPublishQueueLen(length int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueLen = length
	m.maxLen = max(m.maxLen, length)
}

func (m *mockMetrics) RecordDroppedPublish() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dropped++
}

func (m *mockMetrics) counts() (dropped, queueLen, maxLen int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dropped, m.queueLen, m.maxLen
}

// TestAsyncGossiper tests the AsyncGossiper component
// because the component is small and simple, it is tested as a whole
// this test starts, runs, clears and stops the AsyncGossiper
// because the AsyncGossiper is run in an async component, it is tested with eventually
func TestAsyncGossiper(t *testing.T) {
	m := &mockNetwork{}
	// Create a new instance of AsyncGossiper
	p := NewAsyncGossiper(context.Background(), m, log.New(), &mockMetrics{})

	// Start the AsyncGossiper
	p.Start()

	// Test that the AsyncGossiper is running within a short duration
	require.Eventually(t, func() bool {
		return p.running.Load()
	}, 10*time.Second, 10*time.Millisecond)

	// send a payload
	payload := &eth.ExecutionPayload{
		BlockNumber: hexutil.Uint64(1),
	}
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: payload,
	}
	p.Gossip(envelope)
	require.Eventually(t, func() bool {
		// Test that the gossiper has content at all
		return p.Get() == envelope &&
			// Test that the payload has been sent to the (mock) network
			m.reqs[0] == envelope
	}, 10*time.Second, 10*time.Millisecond)

	p.Clear()
	require.Eventually(t, func() bool {
		// Test that the gossiper has no payload
		return p.Get() == nil
	}, 10*time.Second, 10*time.Millisecond)

	// Stop the AsyncGossiper
	p.Stop()

	// Test that the AsyncGossiper stops within a short duration
	require.Eventually(t, func() bool {
		return !p.running.Load()
	}, 10*time.Second, 10*time.Millisecond)
}

// TestAsyncGossiperLoop confirms that when called repeatedly, the AsyncGossiper holds the latest payload
// and sends all payloads to the network
func TestAsyncGossiperLoop(t *testing.T) {
	m := &mockNetwork{}
	// Create a new instance of AsyncGossiper
	p := NewAsyncGossiper(context.Background(), m, log.New(), &mockMetrics{})

	// Start the AsyncGossiper
	p.Start()

	// Test that the AsyncGossiper is running within a short duration
	require.Eventually(t, func() bool {
		return p.running.Load()
	}, 10*time.Second, 10*time.Millisecond)

	// send multiple payloads
	for i := 0; i < 10; i++ {
		payload := &eth.ExecutionPayload{
			BlockNumber: hexutil.Uint64(i),
		}
		envelope := &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: payload,
		}
		p.Gossip(envelope)
		require.Eventually(t, func() bool {
			// Test that the gossiper has content at all
			return p.Get() == envelope &&
				// Test that the payload has been sent to the (mock) network
				m.reqs[len(m.reqs)-1] == envelope
		}, 10*time.Second, 10*time.Millisecond)
	}
	require.Equal(t, 10, len(m.reqs))
	// Stop the AsyncGossiper
	p.Stop()
	// Test that the AsyncGossiper stops within a short duration
	require.Eventually(t, func() bool {
		return !p.running.Load()
	}, 10*time.Second, 10*time.Millisecond)
}

// failingNetwork is a mock network that always fails to publish
type failingNetwork struct{}

func (f *failingNetwork) SignAndPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	return errors.New("failed to publish")
}

// TestAsyncGossiperFailToPublish tests that the AsyncGossiper clears the stored payload if the network fails
func TestAsyncGossiperFailToPublish(t *testing.T) {
	m := &failingNetwork{}
	// Create a new instance of AsyncGossiper
	p := NewAsyncGossiper(context.Background(), m, log.New(), &mockMetrics{})

	// Start the AsyncGossiper
	p.Start()

	// send a payload
	payload := &eth.ExecutionPayload{
		BlockNumber: hexutil.Uint64(1),
	}
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: payload,
	}
	p.Gossip(envelope)
	// Rather than expect the payload to become available, we should never see it, due to the publish failure
	require.Never(t, func() bool {
		return p.Get() == envelope
	}, 10*time.Second, 10*time.Millisecond)
	// Stop the AsyncGossiper
	p.Stop()
	// Test that the AsyncGossiper stops within a short duration
	require.Eventually(t, func() bool {
		return !p.running.Load()
	}, 10*time.Second, 10*time.Millisecond)
}

// blockingNetwork is a mock network whose publish call blocks until released,
// standing in for a slow remote-signer round-trip.
type blockingNetwork struct {
	started     chan *eth.ExecutionPayloadEnvelope
	finished    chan *eth.ExecutionPayloadEnvelope
	release     chan struct{}
	releaseOnce sync.Once

	mu sync.Mutex
	// failuresLeft counts, per block hash, how many further publish attempts
	// must fail before one is allowed to succeed. See failNext.
	failuresLeft map[common.Hash]int
}

func newBlockingNetwork() *blockingNetwork {
	// Buffered well past the queue cap: a publish must never wait for the test to
	// observe it, or a test that drains a full queue wedges the publisher (and
	// with it Stop) instead of failing an assertion.
	const room = 4 * maxPublishQueue
	return &blockingNetwork{
		started:  make(chan *eth.ExecutionPayloadEnvelope, room),
		finished: make(chan *eth.ExecutionPayloadEnvelope, room),
		release:  make(chan struct{}),
	}
}

func (b *blockingNetwork) SignAndPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	b.started <- payload
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	b.mu.Lock()
	if left := b.failuresLeft[payload.ExecutionPayload.BlockHash]; left > 0 {
		b.failuresLeft[payload.ExecutionPayload.BlockHash] = left - 1
		b.mu.Unlock()
		return errors.New("scripted publish failure")
	}
	b.mu.Unlock()
	b.finished <- payload
	return nil
}

// Release unblocks every publish, in flight and future. Safe to call repeatedly.
func (b *blockingNetwork) Release() {
	b.releaseOnce.Do(func() { close(b.release) })
}

func testEnvelope(num uint64) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: hexutil.Uint64(num),
			BlockHash:   common.Hash{byte(num)},
		},
	}
}

// requirePayload waits for a payload to appear on c, e.g. a publish starting.
func requirePayload(t *testing.T, c chan *eth.ExecutionPayloadEnvelope, msg string) *eth.ExecutionPayloadEnvelope {
	t.Helper()
	select {
	case payload := <-c:
		return payload
	case <-time.After(30 * time.Second):
		t.Fatal(msg)
		return nil
	}
}

// TestAsyncGossiperDoesNotBlockOnPublish is the regression test for the
// sequencer stall: the sequencer calls Clear() a millisecond or two after
// Gossip(), i.e. while the publish is in flight, and computes the next block's
// build window as a residual - so every millisecond spent waiting here comes
// straight out of the next block's building time.
func TestAsyncGossiperDoesNotBlockOnPublish(t *testing.T) {
	m := newBlockingNetwork()
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), &mockMetrics{})
	p.Start()
	// deferred in this order so the publish is released before Stop waits on it
	defer p.Stop()
	defer m.Release()

	envelope := testEnvelope(1)
	p.Gossip(envelope)
	require.Equal(t, envelope, requirePayload(t, m.started, "publish never started"))

	// with the publish in flight, the sequencer's Get() and Clear() must not wait for it
	cleared := make(chan struct{})
	go func() {
		defer close(cleared)
		p.Get()
		p.Clear()
	}()
	select {
	case <-cleared:
	case <-time.After(2 * time.Second):
		m.Release() // let the publisher finish, so Stop can complete
		t.Fatal("Get/Clear blocked until the in-flight publish returned")
	}

	// clearing must not cancel the publish: peers still need the block
	m.Release()
	require.Equal(t, envelope, requirePayload(t, m.finished, "publish never finished"))

	// nor may the completing publish repopulate the buffer the sequencer just
	// cleared: reusing it would rebuild the block that is already the head
	require.Never(t, func() bool {
		return p.Get() != nil
	}, 200*time.Millisecond, 10*time.Millisecond)
}

// TestAsyncGossiperOverlappingPublishes covers a publish still being in flight
// when the next blocks are sealed: handing them over must not block, and every
// one of them is published, in the order it was sealed. A verifier cannot follow
// the chain past a block it never received, so none may be skipped.
func TestAsyncGossiperOverlappingPublishes(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	first := testEnvelope(1)
	p.Gossip(first)
	require.Equal(t, first, requirePayload(t, m.started, "first publish never started"))

	second, third := testEnvelope(2), testEnvelope(3)
	handed := make(chan struct{})
	go func() {
		defer close(handed)
		p.Gossip(second)
		p.Clear()
		p.Gossip(third)
		p.Clear()
	}()
	select {
	case <-handed:
	case <-time.After(2 * time.Second):
		m.Release()
		t.Fatal("Gossip blocked until the in-flight publish returned")
	}

	m.Release()
	require.Equal(t, first, requirePayload(t, m.finished, "first publish never finished"))
	require.Equal(t, second, requirePayload(t, m.started, "queued second never started"))
	require.Equal(t, second, requirePayload(t, m.finished, "queued second never finished"))
	require.Equal(t, third, requirePayload(t, m.started, "queued third never started"))
	require.Equal(t, third, requirePayload(t, m.finished, "queued third never finished"))

	dropped, _, maxLen := metrics.counts()
	require.Zero(t, dropped, "nothing should be dropped: every block was still publishable")
	require.Equal(t, 2, maxLen, "the backlog behind the in-flight publish should be visible in metrics")
}

// TestAsyncGossiperBoundsQueue covers a signer the sequencer cannot reach for
// long enough to fill the queue: the oldest entries are dropped rather than
// held, so the cost lands on gossip - recoverable by other means - instead of on
// block production or on memory.
func TestAsyncGossiperBoundsQueue(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	p.Gossip(testEnvelope(0))
	require.Equal(t, testEnvelope(0).ExecutionPayload.BlockNumber,
		requirePayload(t, m.started, "first publish never started").ExecutionPayload.BlockNumber)

	// pile up twice the cap behind the stuck publish
	for i := 1; i <= 2*maxPublishQueue; i++ {
		p.Gossip(testEnvelope(uint64(i)))
	}

	dropped, queueLen, _ := metrics.counts()
	require.Equal(t, maxPublishQueue, queueLen, "queue must not grow past the cap")
	require.Equal(t, maxPublishQueue, dropped, "the overflow must be counted as dropped")

	// the oldest were dropped, so the queue resumes from the newest run of blocks
	m.Release()
	next := requirePayload(t, m.started, "queued publish never started")
	require.Equal(t, hexutil.Uint64(maxPublishQueue+1), next.ExecutionPayload.BlockNumber,
		"the oldest entries are dropped, keeping the newest")
}

// countingNetwork records the last payload it published.
type countingNetwork struct {
	mu   sync.Mutex
	last uint64
}

func (c *countingNetwork) SignAndPublishL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = uint64(envelope.ExecutionPayload.BlockNumber)
	return nil
}

// TestAsyncGossiperConcurrentHandoff hammers the handoff to the publisher
// goroutine to confirm no wake-up is lost: the last payload handed over is
// always published, however the concurrent calls interleave.
func TestAsyncGossiperConcurrentHandoff(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	for round := 0; round < 50; round++ {
		m := &countingNetwork{}
		p := NewAsyncGossiper(context.Background(), m, logger, &mockMetrics{})
		p.Start()

		var wg sync.WaitGroup
		for i := 1; i <= 20; i++ {
			wg.Add(1)
			go func(i uint64) {
				defer wg.Done()
				p.Gossip(testEnvelope(i))
				p.Get()
				p.Clear()
			}(uint64(i))
		}
		wg.Wait()

		p.Gossip(testEnvelope(999))
		require.Eventually(t, func() bool {
			m.mu.Lock()
			defer m.mu.Unlock()
			return m.last == 999
		}, 10*time.Second, time.Millisecond, "round %d: the last payload was never published", round)
		p.Stop()
	}
}

// failNext scripts the next n publish attempts of a hash to fail, so a test can
// exercise a signer that rejects or times out on a block.
func (b *blockingNetwork) failNext(hash common.Hash, n int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failuresLeft == nil {
		b.failuresLeft = make(map[common.Hash]int)
	}
	b.failuresLeft[hash] = n
}

// TestAsyncGossiperDedupesResealedBlock covers the sequencer retrying a block
// whose insertion failed temporarily. Get() does not wait for the in-flight
// publish, so the sequencer re-seals the same build job and hands the identical
// block over again, once per retry. Those duplicates must not queue up behind
// the publish that is already sending that very block, and must not invalidate
// it for reuse - that would leave the sequencer re-sealing indefinitely, and
// fill the queue with copies of one block.
func TestAsyncGossiperDedupesResealedBlock(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	envelope := testEnvelope(1)
	p.Gossip(envelope)
	require.Equal(t, envelope, requirePayload(t, m.started, "publish never started"))

	for i := 0; i < 5; i++ {
		require.Nil(t, p.Get(), "the publish is still in flight, so there is nothing to reuse yet")
		// a re-seal of the same build job: an equal block, a distinct value
		p.Gossip(testEnvelope(1))
	}
	_, queueLen, maxLen := metrics.counts()
	require.Zero(t, queueLen, "a block already being published must not be queued again")
	require.Equal(t, 1, maxLen, "only the original block was ever queued, none of the duplicates")

	m.Release()
	require.Equal(t, envelope, requirePayload(t, m.finished, "publish never finished"))

	require.Eventually(t, func() bool {
		return p.Get() != nil
	}, 10*time.Second, 10*time.Millisecond,
		"the published block must be offered for reuse: the duplicates must not have invalidated it")

	require.Never(t, func() bool {
		select {
		case <-m.started:
			return true
		default:
			return false
		}
	}, 200*time.Millisecond, 20*time.Millisecond, "the block must be published only once")
}

// TestAsyncGossiperRetriesFailedPublish covers a publish that fails - a signer
// blip, or the new publish timeout. Dropping that block and carrying on with its
// descendants would leave peers a gap they cannot follow the chain past, which
// is the same reason blocks are published in seal order rather than skipping to
// the tip. So the failed block is retried ahead of the blocks behind it.
func TestAsyncGossiperRetriesFailedPublish(t *testing.T) {
	m := newBlockingNetwork()
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), &mockMetrics{})
	p.Start()
	defer p.Stop()
	defer m.Release()

	first, second, third := testEnvelope(1), testEnvelope(2), testEnvelope(3)
	m.failNext(first.ExecutionPayload.BlockHash, 1)

	p.Gossip(first)
	require.Equal(t, first, requirePayload(t, m.started, "first publish never started"))
	// queue the descendants behind the publish that is about to fail
	p.Gossip(second)
	p.Gossip(third)

	m.Release()
	require.Equal(t, first, requirePayload(t, m.started, "the failed block was never retried"))
	require.Equal(t, first, requirePayload(t, m.finished, "the retry never succeeded"))
	require.Equal(t, second, requirePayload(t, m.started, "second never started"))
	require.Equal(t, third, requirePayload(t, m.started, "third never started"))
}

// TestAsyncGossiperGivesUpOnFailedPublish bounds the retry: a signer that is
// down fails every attempt, and holding the head forever would stop every later
// block from being published too.
func TestAsyncGossiperGivesUpOnFailedPublish(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	first, second := testEnvelope(1), testEnvelope(2)
	m.failNext(first.ExecutionPayload.BlockHash, maxPublishAttempts)

	p.Gossip(first)
	require.Equal(t, first, requirePayload(t, m.started, "first publish never started"))
	p.Gossip(second)

	m.Release()
	for i := 1; i < maxPublishAttempts; i++ {
		require.Equal(t, first, requirePayload(t, m.started, "the failed block was never retried"))
	}
	require.Equal(t, second, requirePayload(t, m.started, "a block behind an unpublishable one must still go out"))
	require.Equal(t, second, requirePayload(t, m.finished, "second never finished"))

	dropped, _, _ := metrics.counts()
	require.Zero(t, dropped, "a block that failed to publish is a publishing error, not a queue-overflow drop")
}

// TestAsyncGossiperDiscardDropsQueuedBlock covers the sequencer rejecting a
// block it had already handed over - invalid, denied, or stale against a chain
// that moved on. Clear cannot express that: it is also what the sequencer calls
// when a block is successfully inserted, when the queue must keep publishing. So
// a rejected block is Discarded by hash, and only that block goes.
func TestAsyncGossiperDiscardDropsQueuedBlock(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	first, rejected, third := testEnvelope(1), testEnvelope(2), testEnvelope(3)
	p.Gossip(first)
	require.Equal(t, first, requirePayload(t, m.started, "first publish never started"))
	p.Gossip(rejected)
	p.Gossip(third)

	p.Discard(rejected.ExecutionPayload.BlockHash)
	_, queueLen, _ := metrics.counts()
	require.Equal(t, 1, queueLen, "only the rejected block leaves the queue")

	m.Release()
	require.Equal(t, first, requirePayload(t, m.finished, "first publish never finished"))
	require.Equal(t, third, requirePayload(t, m.started,
		"the rejected block must be skipped, and the good block behind it still published"))
	require.Equal(t, third, requirePayload(t, m.finished, "third never finished"))
}

// TestAsyncGossiperClearKeepsQueue is the other half: Clear means the block was
// inserted, so the blocks queued behind it are good blocks that peers still need.
func TestAsyncGossiperClearKeepsQueue(t *testing.T) {
	m := newBlockingNetwork()
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), &mockMetrics{})
	p.Start()
	defer p.Stop()
	defer m.Release()

	first, second := testEnvelope(1), testEnvelope(2)
	p.Gossip(first)
	require.Equal(t, first, requirePayload(t, m.started, "first publish never started"))
	p.Gossip(second)
	p.Clear()

	m.Release()
	require.Equal(t, first, requirePayload(t, m.finished, "first publish never finished"))
	require.Equal(t, second, requirePayload(t, m.started, "a queued block must survive Clear"))
	require.Equal(t, second, requirePayload(t, m.finished, "second never finished"))
}

// TestAsyncGossiperDiscardInFlightBlock covers a block rejected while its publish
// is already in flight. That send cannot be recalled, but it must not be retried
// when it fails, and must not come back as a payload to reuse when it succeeds.
func TestAsyncGossiperDiscardInFlightBlock(t *testing.T) {
	m := newBlockingNetwork()
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), &mockMetrics{})
	p.Start()
	defer p.Stop()
	defer m.Release()

	envelope, next := testEnvelope(1), testEnvelope(2)
	m.failNext(envelope.ExecutionPayload.BlockHash, 1)
	p.Gossip(envelope)
	require.Equal(t, envelope, requirePayload(t, m.started, "publish never started"))

	p.Discard(envelope.ExecutionPayload.BlockHash)
	p.Gossip(next)

	m.Release()
	require.Equal(t, next, requirePayload(t, m.started,
		"a discarded block must not be retried ahead of the blocks behind it"))
	require.Equal(t, next, requirePayload(t, m.finished, "next never finished"))
	require.Never(t, func() bool {
		reusable := p.Get()
		return reusable != nil &&
			reusable.ExecutionPayload.BlockHash == envelope.ExecutionPayload.BlockHash
	}, 200*time.Millisecond, 20*time.Millisecond,
		"a discarded block must never be offered back for reuse")
}

// TestAsyncGossiperDiscardAll covers the sequencer abandoning the chain the
// queued blocks extend - a derivation reset, or a start from an unknown
// pre-state. Those blocks belong to a chain that has been rewound, so none of
// them may still go out, and an in-flight publish must not come back as a
// payload to reuse on the chain the sequencer moves to.
func TestAsyncGossiperDiscardAll(t *testing.T) {
	m := newBlockingNetwork()
	metrics := &mockMetrics{}
	p := NewAsyncGossiper(context.Background(), m, testlog.Logger(t, log.LevelError), metrics)
	p.Start()
	defer p.Stop()
	defer m.Release()

	inFlight := testEnvelope(1)
	p.Gossip(inFlight)
	require.Equal(t, inFlight, requirePayload(t, m.started, "first publish never started"))
	for i := 2; i <= 5; i++ {
		p.Gossip(testEnvelope(uint64(i)))
	}
	_, queueLen, _ := metrics.counts()
	require.Equal(t, 4, queueLen, "the blocks behind the in-flight publish are queued")

	p.DiscardAll()
	_, queueLen, _ = metrics.counts()
	require.Zero(t, queueLen, "a rewound chain leaves nothing worth publishing")

	m.Release()
	require.Equal(t, inFlight, requirePayload(t, m.finished,
		"the in-flight publish cannot be recalled, and still completes"))
	require.Never(t, func() bool {
		select {
		case <-m.started:
			return true
		default:
			return p.Get() != nil
		}
	}, 300*time.Millisecond, 20*time.Millisecond,
		"no queued block may be published after the chain was abandoned, nor offered for reuse")
}
