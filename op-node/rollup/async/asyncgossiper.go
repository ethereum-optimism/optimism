package async

import (
	"context"
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
	// maxPublishAttempts bounds head-of-line blocking. A retry goes to the front of
	// the queue, so a block that keeps failing is eventually dropped rather than
	// held in front of its descendants indefinitely.
	maxPublishAttempts = 3
	// retryInterval paces retries. Topic.Publish runs op-node's own block validator
	// inline and synchronously on the publishing node, so a publish can fail in
	// about a millisecond and unpaced retries spin.
	retryInterval = time.Second
	// publishTimeout bounds a single publish. With a remote signer the publish is
	// an HTTPS round-trip that carries no deadline of its own, so without this a
	// hung connection stops gossip until the connection dies by itself.
	publishTimeout = 5 * time.Second
)

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
	// attempts counts failed publishes of the block currently at queue[0]. It is
	// reset whenever queue[0] changes.
	attempts int

	// wake coalesces signals (cap 1) for the publish goroutine.
	wake chan struct{}
	// stop is unbuffered: Stop blocks until the publish goroutine accepts.
	stop chan struct{}

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
		stop:          make(chan struct{}),
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

// Clear drops every queued block. The sequencer uses it when the chain those
// blocks extend is not the one it is building on: a reset, or a start from an
// unknown pre-state.
func (p *SimpleAsyncGossiper) Clear() {
	p.mu.Lock()
	p.queue = nil
	p.attempts = 0
	p.mu.Unlock()
	p.metrics.RecordPublishQueueLen(0)
}

// Stop stops the publish goroutine. It blocks until the goroutine accepts.
func (p *SimpleAsyncGossiper) Stop() {
	if !p.running.Load() {
		return
	}
	p.stop <- struct{}{}
}

// Start starts the publish goroutine.
func (p *SimpleAsyncGossiper) Start() {
	if !p.running.CompareAndSwap(false, true) {
		return
	}
	go p.publishLoop()
}

// publishLoop publishes the front of the queue until it is empty, then waits.
// A failed publish is retried at the front rather than skipped: peers follow the
// chain block by block, so a gap they cannot cross is worse than a delay.
func (p *SimpleAsyncGossiper) publishLoop() {
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
			if !p.waitForWork() {
				return
			}
			continue
		}

		err := p.publish(envelope)

		var gaveUp, retry bool
		p.mu.Lock()
		// Clear, or an eviction, may have removed this block while the publish was
		// in flight. Then the outcome is no longer ours to record.
		if len(p.queue) > 0 && p.queue[0].envelope == envelope {
			switch {
			case err == nil:
				p.discardHead()
			case attempts+1 >= maxPublishAttempts:
				p.discardHead()
				gaveUp = true
			default:
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

		if retry && !p.pause(p.retryInterval) {
			return
		}
	}
}

// publish publishes one block, under a deadline of its own.
func (p *SimpleAsyncGossiper) publish(envelope *eth.ExecutionPayloadEnvelope) error {
	ctx, cancel := context.WithTimeout(p.ctx, publishTimeout)
	defer cancel()
	return p.net.SignAndPublishL2Payload(ctx, envelope)
}

// waitForWork blocks until a block is queued. It reports false when the gossiper
// is stopping.
func (p *SimpleAsyncGossiper) waitForWork() bool {
	select {
	case <-p.wake:
		return true
	case <-p.stop:
		return false
	case <-p.ctx.Done():
		return false
	}
}

// pause waits out the retry interval. It reports false when the gossiper is
// stopping, so Stop does not wait for the interval to elapse.
func (p *SimpleAsyncGossiper) pause(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-p.stop:
		return false
	case <-p.ctx.Done():
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
