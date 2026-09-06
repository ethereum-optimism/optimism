package async

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	// maxPublishQueue caps the queue, so that a signer the sequencer cannot reach
	// costs gossip rather than block production: once the queue is full the
	// oldest entries are dropped instead of held, and the sequencer is never
	// asked to wait for room. A block that goes ungossiped still reaches other
	// nodes by other means - L1 and derivation, other sync mechanisms - whereas
	// a sequencer that stops building produces nothing for anyone.
	//
	// It doubles as a memory bound: an envelope can be megabytes. 32 is ~1 minute
	// of blocks at a 2s block time.
	maxPublishQueue = 32

	// maxPublishAttempts bounds how often a single block is offered to the
	// network before it is given up on. A block that fails to publish is retried
	// ahead of the blocks behind it, because a peer cannot follow the chain past
	// a block it never received - the same reason blocks go out in seal order
	// rather than skipping to the tip. The bound keeps a signer that is simply
	// down from holding the head, and every block behind it, forever.
	maxPublishAttempts = 2
)

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Get() *eth.ExecutionPayloadEnvelope
	Clear()
	Discard(hash common.Hash)
	DiscardAll()
	Stop()
	Start()
}

// SimpleAsyncGossiper publishes sealed blocks to p2p without holding up the
// sequencer, and holds the last published block for the sequencer to reuse.
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
	// because the publisher reads the queue after receiving one.
	wake chan struct{}
	// channel to request stopping the publisher goroutine
	stop chan struct{}

	mu sync.Mutex
	// queue holds the payloads waiting to be published, oldest first. Blocks are
	// published in the order they were sealed, so a peer is not asked to accept a
	// block before its parent.
	queue []*queuedPayload
	// currentPayload is the last successfully published payload that has not
	// been cleared or discarded since: the payload the sequencer may reuse
	currentPayload *eth.ExecutionPayloadEnvelope
	// wanted is the block the sequencer is currently trying to make canonical.
	// A publish that completes for any other block must not offer its payload
	// for reuse: the sequencer has moved past it. Gossip sets it, Clear and
	// Discard zero it.
	wanted common.Hash
	// publishing is the block the publisher goroutine currently has in flight,
	// valid only while publishInFlight is set. It keeps a re-seal of that same
	// block from being queued behind the publish that is already sending it.
	// The flag is explicit rather than a zero-hash sentinel: a hash that
	// happened to be zero would otherwise read as "idle" and, worse, make every
	// block look already-in-flight and so never be published at all.
	publishing      common.Hash
	publishInFlight bool
	// publishDiscarded records that the in-flight block was discarded while it
	// was being published, so the attempt is not retried or reused on return.
	publishDiscarded bool

	ctx     context.Context
	net     Network
	log     log.Logger
	metrics Metrics
}

// queuedPayload is a payload awaiting publication.
type queuedPayload struct {
	payload *eth.ExecutionPayloadEnvelope
	hash    common.Hash
	// attempts counts how often this block has been handed to the network,
	// including the attempt in flight. See maxPublishAttempts.
	attempts int
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
	}
}

// Gossip queues a payload for publication. It does not wait for the payload to
// be published, and never waits for room in the queue.
//
// Blocks are published in the order they were sealed rather than skipping to the
// tip: a verifier that never receives a block cannot follow the chain past it
// over gossip alone. Blocks are dropped only when the queue overflows - see
// maxPublishQueue - at which point keeping the sequencer building is worth more
// than the gossip.
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	p.enqueue(payload)
	p.signal()
}

// enqueue adds a payload to the queue, dropping the oldest entries if that puts
// it over the cap. It holds the mutex across both the queue mutation and the
// gauge update, so the reported depth cannot be reordered against the queue it
// describes.
func (p *SimpleAsyncGossiper) enqueue(payload *eth.ExecutionPayloadEnvelope) {
	hash := payload.ExecutionPayload.BlockHash

	p.mu.Lock()
	defer p.mu.Unlock()

	// This is the block the sequencer is now trying to make canonical.
	p.wanted = hash
	if p.isPending(hash) {
		// A re-seal of a block already queued or in flight. The sequencer does
		// this while an insert keeps failing temporarily, once per retry.
		// Publishing the same block again is wasted work, and queueing it would
		// crowd out the blocks behind it.
		return
	}

	p.queue = append(p.queue, &queuedPayload{payload: payload, hash: hash})
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
	p.metrics.RecordPublishQueueLen(len(p.queue))
}

// isPending reports whether the block is already queued or in flight. Callers
// must hold p.mu.
func (p *SimpleAsyncGossiper) isPending(hash common.Hash) bool {
	if p.publishInFlight && p.publishing == hash {
		return true
	}
	for _, queued := range p.queue {
		if queued.hash == hash {
			return true
		}
	}
	return false
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

// Clear drops the payload held for reuse, for when the sequencer no longer needs
// it because the block was inserted. Queued blocks are left to publish: they are
// good blocks, and peers still need them.
//
// A publish that is in flight still completes - peers need the block either way
// - but its result no longer repopulates the buffer.
func (p *SimpleAsyncGossiper) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentPayload = nil
	p.wanted = common.Hash{}
}

// Discard drops a block the sequencer has rejected - invalid, denied, or stale
// against a chain that moved on - so it is neither published from the queue nor
// offered back for reuse. This is the counterpart to Clear: Clear says the block
// made it, Discard says it did not.
//
// A publish already in flight cannot be recalled, but it is not retried, and its
// result is not stored.
func (p *SimpleAsyncGossiper) Discard(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentPayload != nil && p.currentPayload.ExecutionPayload.BlockHash == hash {
		p.currentPayload = nil
	}
	if p.wanted == hash {
		p.wanted = common.Hash{}
	}
	if p.publishInFlight && p.publishing == hash {
		p.publishDiscarded = true
	}

	kept := p.queue[:0]
	for _, queued := range p.queue {
		if queued.hash == hash {
			p.log.Info("discarding unpublished block the sequencer rejected",
				"block", queued.payload.ExecutionPayload.ID())
			continue
		}
		kept = append(kept, queued)
	}
	for i := len(kept); i < len(p.queue); i++ {
		p.queue[i] = nil // do not keep discarded envelopes alive through the backing array
	}
	p.queue = kept
	p.metrics.RecordPublishQueueLen(len(p.queue))
}

// DiscardAll drops everything: the payload held for reuse, and every block still
// waiting to be published. It is for when the sequencer abandons the chain it was
// building on - a derivation reset, or a start from an unknown pre-state - after
// which the queued blocks belong to a chain that has been rewound. Publishing
// them would offer peers blocks the sequencer itself no longer stands behind.
//
// This is the bulk counterpart to Discard. Clear is deliberately not this: it is
// also what the sequencer calls when a block is successfully inserted, where the
// blocks queued behind it are good blocks that peers still need.
func (p *SimpleAsyncGossiper) DiscardAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentPayload = nil
	p.wanted = common.Hash{}
	if p.publishInFlight {
		p.publishDiscarded = true
	}
	if len(p.queue) > 0 {
		p.log.Info("discarding unpublished blocks, the sequencer abandoned the chain they extend",
			"len", len(p.queue))
	}
	for i := range p.queue {
		p.queue[i] = nil // do not keep the envelopes alive through the backing array
	}
	p.queue = nil
	p.metrics.RecordPublishQueueLen(0)
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

// dequeue takes the oldest queued payload and marks it in flight, along with the
// number of entries left behind. It returns nil when the queue is empty. Like
// enqueue, it updates the gauge under the same lock as the queue mutation.
func (p *SimpleAsyncGossiper) dequeue() (*queuedPayload, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		p.publishInFlight = false
		return nil, 0
	}
	next := p.queue[0]
	p.queue[0] = nil // do not keep the envelope alive through the backing array
	p.queue = p.queue[1:]
	next.attempts++
	p.publishing = next.hash
	p.publishInFlight = true
	p.publishDiscarded = false
	p.metrics.RecordPublishQueueLen(len(p.queue))
	return next, len(p.queue)
}

// publish publishes the next queued payload. It runs on the publisher goroutine
// and holds no lock across the network call.
func (p *SimpleAsyncGossiper) publish() {
	next, remaining := p.dequeue()
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
	err := p.net.SignAndPublishL2Payload(ctx, next.payload)
	p.finishPublish(next, err)
}

// finishPublish records the outcome of a publish attempt: on success the payload
// becomes available for reuse if the sequencer still wants it, and on failure the
// block is retried ahead of the blocks behind it, within maxPublishAttempts.
func (p *SimpleAsyncGossiper) finishPublish(next *queuedPayload, err error) {
	p.mu.Lock()
	p.publishInFlight = false
	discarded := p.publishDiscarded
	p.publishDiscarded = false

	if err == nil {
		// A Clear (the block was inserted), a Discard (it was rejected), or a
		// Gossip of a later block while we were publishing all make this payload
		// unfit for reuse: the sequencer would rebuild a block it has moved past.
		if !discarded && p.wanted == next.hash {
			p.currentPayload = next.payload
		}
		p.mu.Unlock()
		return
	}

	// Retrying at the front keeps the chain gap-free for peers, but never at the
	// cost of unbounded growth or of a block the sequencer has already rejected.
	retry := !discarded && next.attempts < maxPublishAttempts && len(p.queue) < maxPublishQueue
	if retry {
		p.queue = append([]*queuedPayload{next}, p.queue...)
		p.metrics.RecordPublishQueueLen(len(p.queue))
	}
	p.mu.Unlock()

	p.metrics.RecordPublishingError()
	p.log.Warn("failed to publish newly created block",
		"id", next.payload.ExecutionPayload.ID(),
		"hash", next.payload.ExecutionPayload.BlockHash,
		"attempts", next.attempts,
		"retrying", retry,
		"err", err)
	if retry {
		p.signal()
	}
}

// NoOpGossiper is a no-op implementation of AsyncGossiper
// it serves as a placeholder for when the AsyncGossiper is not needed
type NoOpGossiper struct{}

func (NoOpGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {}
func (NoOpGossiper) Get() *eth.ExecutionPayloadEnvelope           { return nil }
func (NoOpGossiper) Clear()                                       {}
func (NoOpGossiper) Discard(hash common.Hash)                     {}
func (NoOpGossiper) DiscardAll()                                  {}
func (NoOpGossiper) Stop()                                        {}
func (NoOpGossiper) Start()                                       {}
