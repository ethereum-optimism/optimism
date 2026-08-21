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
	// publishTimeout bounds a single publish attempt: signing the block (a network
	// round-trip when a remote signer is configured) plus the p2p publish. It is
	// generous compared with any block time, so it never cuts a working signer
	// short. Its job is to stop a signer connection that hangs rather than errors
	// from wedging publishing until TCP gives up.
	publishTimeout = 10 * time.Second

	// maxPublishAge is how long a sealed block may wait for its turn to be
	// published before it is dropped. It mirrors the default of
	// --p2p.gossip.timestamp.threshold: peers REJECT a payload older than that,
	// and an invalid delivery is scored against the sender, so publishing a block
	// that has aged out spends a signing round-trip and peer score to have the
	// block refused. A node configured with a lower threshold publishes a few
	// blocks its peers refuse; the horizon is deliberately not plumbed through
	// from the p2p config for that one case.
	maxPublishAge = 60 * time.Second

	// maxPublishQueue caps the queue, so that a signer the sequencer cannot reach
	// costs gossip rather than block production: once the queue is full the
	// oldest entries are dropped instead of held, and the sequencer is never
	// asked to wait for room. A block that goes ungossiped still reaches other
	// nodes by other means - L1 and derivation, other sync mechanisms - whereas
	// a sequencer that stops building produces nothing for anyone.
	//
	// It doubles as a memory bound: an envelope can be megabytes, and
	// maxPublishAge alone bounds the queue only in time. 32 spans the whole
	// publishable window at a 2s block time.
	maxPublishQueue = 32
)

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Get() *eth.ExecutionPayloadEnvelope
	Clear()
	Stop()
	Start()
}

// SimpleAsyncGossiper is a component that stores and gossips a single payload at a time
// the payload can be accessed by the Get function to be reused when the payload was gossiped but not inserted
//
// Publishing runs on a dedicated goroutine, and the exposed functions only take
// a mutex: none of them waits for the network. The sequencer calls Gossip, Get
// and Clear on the hot path between sealing block N and opening the build window
// for block N+1, and that window is computed as a residual - so any wait here is
// subtracted 1:1 from the next block's building time.
type SimpleAsyncGossiper struct {
	running atomic.Bool
	// wake signals the publisher goroutine that there is a payload to publish.
	// Capacity 1, and sent to without blocking: a single queued signal suffices,
	// because the publisher reads the pending payload after receiving one.
	wake chan struct{}
	// channel to request stopping the publisher goroutine
	stop chan struct{}

	mu sync.Mutex
	// queue holds the payloads waiting to be published, oldest first. Blocks are
	// published in the order they were sealed, so a peer is not asked to accept a
	// block before its parent.
	queue []*queuedPayload
	// currentPayload is the last successfully published payload that has not
	// been cleared since: the payload the sequencer may reuse
	currentPayload *eth.ExecutionPayloadEnvelope
	// epoch identifies the payload the sequencer currently cares about. Both
	// Gossip and Clear advance it, so a publish that was already in flight can
	// tell that its result has since become stale.
	epoch uint64

	ctx     context.Context
	net     Network
	log     log.Logger
	metrics Metrics
	// timeNow is overridden in tests to age queue entries
	timeNow func() time.Time
}

// queuedPayload is a payload awaiting publication, tagged with the epoch it was
// handed over in and when it joined the queue.
type queuedPayload struct {
	payload    *eth.ExecutionPayloadEnvelope
	epoch      uint64
	enqueuedAt time.Time
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
	RecordPublishQueueLen(length int)
	RecordDroppedPublish()
}

func NewAsyncGossiper(ctx context.Context, net Network, log log.Logger, metrics Metrics) *SimpleAsyncGossiper {
	return &SimpleAsyncGossiper{
		wake: make(chan struct{}, 1),
		stop: make(chan struct{}),

		net:     net,
		ctx:     ctx,
		log:     log,
		metrics: metrics,
		timeNow: time.Now,
	}
}

// Gossip queues a payload for publication. It does not wait for the payload to
// be published, and never waits for room in the queue.
//
// Blocks are published in the order they were sealed rather than skipping to the
// tip: a verifier that never receives a block cannot follow the chain past it
// over gossip alone. Blocks are dropped only when the queue overflows or an
// entry ages out - see maxPublishQueue and maxPublishAge - at which point
// keeping the sequencer building is worth more than the gossip.
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	p.mu.Lock()
	p.epoch++
	p.queue = append(p.queue, &queuedPayload{
		payload:    payload,
		epoch:      p.epoch,
		enqueuedAt: p.timeNow(),
	})
	for len(p.queue) > maxPublishQueue {
		dropped := p.queue[0]
		p.queue[0] = nil // do not keep the envelope alive through the backing array
		p.queue = p.queue[1:]
		p.log.Warn("dropping unpublished block, publish queue is full",
			"dropped", dropped.payload.ExecutionPayload.ID(),
			"queued", payload.ExecutionPayload.ID(),
			"len", len(p.queue))
		p.metrics.RecordDroppedPublish()
	}
	length := len(p.queue)
	p.mu.Unlock()

	p.metrics.RecordPublishQueueLen(length)
	p.signal()
}

// signal wakes the publisher goroutine. A queued signal is enough, because the
// publisher reads the queue after receiving one.
func (p *SimpleAsyncGossiper) signal() {
	select {
	case p.wake <- struct{}{}:
	default: // the publisher is awake already, or has a signal queued
	}
}

// Get returns the payload that was published and not cleared since, if any.
//
// It does not wait for a publish that is still in flight: a nil result means
// there is nothing to reuse, and the sequencer seals (or re-seals) its building
// job instead of paying for the wait out of the block's build window.
func (p *SimpleAsyncGossiper) Get() *eth.ExecutionPayloadEnvelope {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentPayload
}

// Clear drops the payload held for reuse. A publish that is in flight still
// completes - peers need the block either way - but its result no longer
// repopulates the buffer.
func (p *SimpleAsyncGossiper) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentPayload = nil
	p.epoch++
}

// Stop is a synchronous function to stop the async routine
// it blocks until the async routine accepts the signal
func (p *SimpleAsyncGossiper) Stop() {
	// if the gossiping isn't running, nothing to do
	if !p.running.Load() {
		return
	}

	p.stop <- struct{}{}
}

// Start starts the AsyncGossiper's publisher goroutine
func (p *SimpleAsyncGossiper) Start() {
	// if the gossiping is already running, return
	if !p.running.CompareAndSwap(false, true) {
		return
	}
	// else, start the publishing loop
	go func() {
		defer p.running.Store(false)
		for {
			select {
			case <-p.wake:
				p.publish()
			case <-p.stop:
				return
			}
		}
	}()
}

// dequeue takes the oldest payload that a peer would still accept, dropping any
// that aged out while they waited. It returns nil when the queue holds nothing
// publishable, along with the number of entries left behind.
func (p *SimpleAsyncGossiper) dequeue() (*queuedPayload, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.queue) > 0 {
		next := p.queue[0]
		p.queue[0] = nil // do not keep the envelope alive through the backing array
		p.queue = p.queue[1:]

		if age := p.timeNow().Sub(next.enqueuedAt); age > maxPublishAge {
			p.log.Warn("dropping unpublished block, aged out of the gossip window",
				"id", next.payload.ExecutionPayload.ID(),
				"age", age, "len", len(p.queue))
			p.metrics.RecordDroppedPublish()
			continue
		}
		return next, len(p.queue)
	}
	return nil, 0
}

// publish publishes the next queued payload and stores it for reuse if the
// publish succeeded and the payload is still the one the sequencer cares about.
// It runs on the publisher goroutine and holds no lock across the network call.
func (p *SimpleAsyncGossiper) publish() {
	next, remaining := p.dequeue()
	p.metrics.RecordPublishQueueLen(remaining)
	if next == nil {
		return // an earlier signal already drained the queue
	}
	if remaining > 0 {
		// Come back for the rest, via the select, so a stop is still honored
		// between publishes.
		p.signal()
	}

	ctx, cancel := context.WithTimeout(p.ctx, publishTimeout)
	defer cancel()
	if err := p.net.SignAndPublishL2Payload(ctx, next.payload); err != nil {
		p.log.Warn("failed to publish newly created block",
			"id", next.payload.ExecutionPayload.ID(),
			"hash", next.payload.ExecutionPayload.BlockHash,
			"err", err)
		p.metrics.RecordPublishingError()
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	// A Clear (the block was inserted) or a newer Gossip while we were
	// publishing makes this payload unfit for reuse: the sequencer would rebuild
	// a block it has already moved past.
	if p.epoch == next.epoch {
		p.currentPayload = next.payload
	}
}

// NoOpGossiper is a no-op implementation of AsyncGossiper
// it serves as a placeholder for when the AsyncGossiper is not needed
type NoOpGossiper struct{}

func (NoOpGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {}
func (NoOpGossiper) Get() *eth.ExecutionPayloadEnvelope           { return nil }
func (NoOpGossiper) Clear()                                       {}
func (NoOpGossiper) Stop()                                        {}
func (NoOpGossiper) Start()                                       {}
