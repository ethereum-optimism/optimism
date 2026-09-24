package async

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// maxPublishQueue bounds the queue. Publishing only ever falls behind on a
	// signer or network hiccup, so a backlog deeper than a minute of blocks is one
	// no peer is still waiting for. The oldest block is evicted first.
	maxPublishQueue = 32
	// retryInterval paces retries. Topic.Publish runs op-node's own block validator
	// inline and synchronously on the publishing node, so a publish can fail in
	// about a millisecond and unpaced retries spin.
	retryInterval = time.Second
	// publishTimeout bounds a single publish. With a remote signer the publish is
	// an HTTPS round-trip that carries no deadline of its own, so without this a
	// hung connection stops gossip until the connection dies by itself.
	publishTimeout = 5 * time.Second
	// defaultRetryWindow is the retry budget when the network reports no gossip
	// threshold, which only happens with p2p disabled - where publishing is a
	// no-op and cannot fail. A backstop against a misconfiguration turning the
	// retry loop unbounded, not a value anything should rely on.
	defaultRetryWindow = time.Minute
)

// ErrPermanentPublish marks a publish failure that no retry can fix. Producers
// wrap it around deterministic rejections only, as a strict allowlist:
// Topic.Publish also surfaces the caller's context error, so a publish that
// merely hit publishTimeout must stay retryable, and classifying that as
// permanent would silently disable retry altogether.
var ErrPermanentPublish = errors.New("permanent publish failure")

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Clear()
	Stop()
	Start()
}

// SimpleAsyncGossiper publishes sealed blocks to peers off the sequencer's hot
// path, in seal order, on a dedicated goroutine.
//
// The sequencer hands a block over only once it has inserted it, so every queued
// block is one this node has already adopted as its canonical head. A queued
// block can therefore be late, but it can never be one the sequencer has to take
// back — which is what makes it safe to queue and retry at all.
//
// Gossip and Clear only take a mutex. Neither waits for the network.
type SimpleAsyncGossiper struct {
	running atomic.Bool

	// mu guards queue and attempts.
	mu sync.Mutex
	// queue holds blocks awaiting publication, oldest first.
	queue []pending
	// attempts counts failed publishes of the block currently at queue[0], for the
	// log field only. Retries are bounded by the block's age, not by a count.
	attempts int

	// wake coalesces signals (cap 1) for the publish goroutine.
	wake chan struct{}

	// lifecycle guards cancelLoop and loopDone, which Start replaces on each run.
	// Signalling the loop through a channel it has to receive from cannot work:
	// the loop also exits on its own when the parent context is cancelled, and a
	// Stop that had already committed to the send would then block forever with
	// no receiver left.
	lifecycle  sync.Mutex
	cancelLoop context.CancelFunc
	loopDone   chan struct{}

	// retryInterval is retryInterval, overridden in tests.
	retryInterval time.Duration

	ctx     context.Context
	net     Network
	log     log.Logger
	metrics Metrics
}

// pending is a block awaiting publication, with the instant it was queued so
// the delay peers pay can be measured.
type pending struct {
	envelope *eth.ExecutionPayloadEnvelope
	queuedAt time.Time
}

// To avoid import cycles, we define a new Network interface here
// this interface is compatible with driver.Network
type Network interface {
	SignAndPublishL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error
	// GossipTimestampThreshold is the age past which peers reject a block
	// outright. Zero means unbounded, which is what a node with p2p disabled
	// reports; publishing is a no-op there anyway.
	GossipTimestampThreshold() time.Duration
}

// To avoid import cycles, we define a new Metrics interface here
// this interface is compatible with driver.Metrics
type Metrics interface {
	RecordPublishingError()
	RecordDroppedPublish()
	RecordPublishQueueLen(length int)
	RecordPublishDelay(duration time.Duration)
}

func NewAsyncGossiper(ctx context.Context, net Network, log log.Logger, metrics Metrics) *SimpleAsyncGossiper {
	return &SimpleAsyncGossiper{
		wake:          make(chan struct{}, 1),
		retryInterval: retryInterval,
		net:           net,
		ctx:           ctx,
		log:           log,
		metrics:       metrics,
	}
}

// Gossip queues a block for publication and returns. It does not wait for the
// network, nor for the publish goroutine.
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	var dropped *eth.ExecutionPayloadEnvelope
	p.mu.Lock()
	if len(p.queue) >= maxPublishQueue {
		// queue[0] may be in flight right now. Evicting it is the point under
		// pressure, and the publish goroutine drops the outcome of a block that is
		// no longer at the front; that narrow race can over-count a drop whose
		// publish landed anyway, which is the harmless direction for this counter.
		dropped = p.queue[0].envelope
		p.discardHead()
	}
	p.queue = append(p.queue, pending{envelope: payload, queuedAt: time.Now()})
	queueLen := len(p.queue)
	p.mu.Unlock()
	p.metrics.RecordPublishQueueLen(queueLen)

	if dropped != nil {
		p.log.Warn("Publish queue is full, dropping oldest unpublished block",
			"dropped", dropped.ExecutionPayload.ID())
		p.metrics.RecordDroppedPublish()
	}
	p.signal()
}

// Clear drops every queued block. The sequencer calls it each time sequencing
// starts, since anything still queued was sealed under a previous stint and
// extends a chain it is no longer necessarily building on. It is not called on a
// reset, so blocks inserted before a reset but not yet published stay queued.
func (p *SimpleAsyncGossiper) Clear() {
	p.mu.Lock()
	p.queue = nil
	p.attempts = 0
	p.mu.Unlock()
	p.metrics.RecordPublishQueueLen(0)
}

// Stop stops the publish goroutine and waits for it to exit. It is safe to call
// on a gossiper that was never started, and on one whose loop has already
// exited because the parent context was cancelled.
func (p *SimpleAsyncGossiper) Stop() {
	p.lifecycle.Lock()
	cancel, done := p.cancelLoop, p.loopDone
	p.lifecycle.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// Start starts the publish goroutine.
func (p *SimpleAsyncGossiper) Start() {
	if !p.running.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithCancel(p.ctx)
	done := make(chan struct{})
	p.lifecycle.Lock()
	p.cancelLoop, p.loopDone = cancel, done
	p.lifecycle.Unlock()
	go p.publishLoop(ctx, done)
}

// publishLoop publishes the front of the queue until it is empty, then waits.
// A failed publish is retried at the front rather than skipped: peers follow the
// chain block by block, so a gap they cannot cross is worse than a delay.
func (p *SimpleAsyncGossiper) publishLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	defer p.running.Store(false)
	for {
		p.mu.Lock()
		var envelope *eth.ExecutionPayloadEnvelope
		var queuedAt time.Time
		if len(p.queue) > 0 {
			envelope, queuedAt = p.queue[0].envelope, p.queue[0].queuedAt
		}
		attempts := p.attempts
		p.mu.Unlock()

		if envelope == nil {
			if !p.waitForWork(ctx) {
				return
			}
			continue
		}

		// A block older than the threshold is rejected by every peer, and by our
		// own validator, which Topic.Publish runs inline. Attempting it anyway
		// costs the retry ladder's full budget at the head of the queue, and that
		// cost is what makes a transient outage permanent: the queue holds
		// maxPublishQueue blocks, so if draining one stale block takes as long as
		// a new block takes to arrive, the queue never empties and every block
		// reaching the head is stale in turn. Dropping without publishing keeps
		// the drain effectively instant, so the queue empties in one pass and
		// gossip recovers by itself.
		if stale, age := p.pastRetryWindow(envelope); stale {
			p.mu.Lock()
			dropped := len(p.queue) > 0 && p.queue[0].envelope == envelope
			if dropped {
				p.discardHead()
			}
			queueLen := len(p.queue)
			p.mu.Unlock()
			if dropped {
				p.log.Warn("Dropping block, it has aged out of the gossip window",
					"id", envelope.ExecutionPayload.ID(), "age", age,
					"threshold", p.net.GossipTimestampThreshold())
				p.metrics.RecordPublishQueueLen(queueLen)
				p.metrics.RecordDroppedPublish()
			}
			continue
		}

		err := p.publish(ctx, envelope)

		var gaveUp, retry bool
		p.mu.Lock()
		// Clear, or an eviction, may have removed this block while the publish was
		// in flight. Then the outcome is no longer ours to record.
		if len(p.queue) > 0 && p.queue[0].envelope == envelope {
			switch {
			case err == nil:
				p.discardHead()
			case errors.Is(err, ErrPermanentPublish):
				// No retry can fix this one, so do not spend the window on it.
				p.discardHead()
				gaveUp = true
			default:
				// Keep trying. The retry budget is the block's remaining life in
				// the gossip window, checked at the top of the loop, so a block is
				// abandoned exactly when peers would stop accepting it rather than
				// after an arbitrary number of tries.
				p.attempts = attempts + 1
				retry = true
			}
		}
		queueLen := len(p.queue)
		p.mu.Unlock()
		p.metrics.RecordPublishQueueLen(queueLen)
		if err == nil {
			// Queued-to-published. Added to the insert time, this is what the
			// publish-after-insert ordering actually costs peers.
			p.metrics.RecordPublishDelay(time.Since(queuedAt))
		}

		if err != nil {
			p.log.Warn("Failed to publish newly created block",
				"id", envelope.ExecutionPayload.ID(),
				"hash", envelope.ExecutionPayload.BlockHash,
				"attempt", attempts+1,
				"err", err)
			p.metrics.RecordPublishingError()
			if gaveUp {
				p.log.Error("Giving up on publishing block, peers will not receive it",
					"id", envelope.ExecutionPayload.ID())
				p.metrics.RecordDroppedPublish()
			}
		}

		if retry && !p.pause(ctx, p.retryInterval) {
			return
		}
	}
}

// pastRetryWindow reports whether a block has aged out of the window peers will
// still accept it in, along with its age. This is both the pre-publish check and
// the retry budget: a block is attempted while it is inside the window and
// abandoned once it leaves, which makes the budget independent of the block time
// and of how long each individual failure takes.
func (p *SimpleAsyncGossiper) pastRetryWindow(envelope *eth.ExecutionPayloadEnvelope) (bool, time.Duration) {
	window := p.net.GossipTimestampThreshold()
	if window <= 0 {
		window = defaultRetryWindow
	}
	age := time.Since(time.Unix(int64(envelope.ExecutionPayload.Timestamp), 0))
	return age >= window, age
}

// publish publishes one block, under a deadline of its own. It derives from the
// loop context so Stop aborts a publish already in flight rather than waiting
// out publishTimeout.
func (p *SimpleAsyncGossiper) publish(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	return p.net.SignAndPublishL2Payload(ctx, envelope)
}

// waitForWork blocks until a block is queued. It reports false when the gossiper
// is stopping.
func (p *SimpleAsyncGossiper) waitForWork(ctx context.Context) bool {
	select {
	case <-p.wake:
		return true
	case <-ctx.Done():
		return false
	}
}

// pause waits out the retry interval. It reports false when the gossiper is
// stopping, so Stop does not wait for the interval to elapse.
func (p *SimpleAsyncGossiper) pause(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// discardHead removes queue[0]. Callers must hold p.mu.
func (p *SimpleAsyncGossiper) discardHead() {
	p.queue[0] = pending{} // don't retain the block in the backing array
	p.queue = p.queue[1:]
	p.attempts = 0
}

// signal wakes the publish goroutine without blocking on it.
func (p *SimpleAsyncGossiper) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// NoOpGossiper is a no-op implementation of AsyncGossiper
// it serves as a placeholder for when the AsyncGossiper is not needed
type NoOpGossiper struct{}

func (NoOpGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {}
func (NoOpGossiper) Clear()                                       {}
func (NoOpGossiper) Stop()                                        {}
func (NoOpGossiper) Start()                                       {}
