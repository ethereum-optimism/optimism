package async

import (
	"context"
	"errors"
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

	// retryInterval paces retries of a failed publish. It is a spin guard, not a
	// policy: how long a block is worth retrying is decided by queue pressure,
	// above. A publish can fail in about a millisecond without touching the
	// network, because op-node runs its own block validator inline on the
	// publishing node - and a block retried past the gossip timestamp threshold
	// fails exactly that way, every time, forever. Unpaced, those retries would
	// spin this goroutine rather than wait for anything.
	retryInterval = time.Second
)

// ErrPermanentPublish marks a publish failure that will recur identically on
// every attempt: the block itself, or this node's configuration, is what was
// rejected. op-node runs its own block validator inline on the publishing node,
// so these are the common case rather than an exotic one - and a block that has
// aged past the gossip timestamp threshold joins them, permanently.
//
// Retrying such a block is not merely wasted work. It sits at the head of the
// queue holding up every block behind it, and a queue that spans more than the
// timestamp threshold would never drain again: each head is already too old, so
// each fails, and the blocks behind it age into the same state.
var ErrPermanentPublish = errors.New("permanent publish failure")

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Discard(hash common.Hash)
	DiscardAll()
	Stop()
	Start()
}

// SimpleAsyncGossiper publishes sealed blocks to p2p without holding up the
// sequencer.
//
// Publishing runs on a dedicated goroutine, and the exposed functions only take
// a mutex: none of them waits for the network. The sequencer calls Gossip on the
// hot path between sealing block N and opening the build window for block N+1,
// and that window is computed as a residual - so any wait here is subtracted 1:1
// from the next block's building time.
//
// It does not hold a block for the sequencer to reuse. The sequencer keeps the
// block it sealed on its own building state, so a failed insert is retried from
// there rather than from here.
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

	// retryInterval is the pacing between retries of a failed publish, defaulted
	// from the constant of the same name. It is a field so tests can shorten it.
	retryInterval time.Duration

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
	// including the attempt in flight. Queue pressure is what normally bounds the
	// retry; this is the backstop for when production has stopped and no eviction
	// is coming. See finishPublish.
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

		retryInterval: retryInterval,

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

	if p.isPending(hash) {
		// The same block handed over twice. The sequencer no longer does this -
		// it re-inserts the block it retained rather than re-sealing - but
		// publishing one block twice is wasted work either way, and queueing it
		// would crowd out the blocks behind it.
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

// Discard drops a block the sequencer has rejected - invalid, denied, or stale
// against a chain that moved on - so it is not published. A block that was
// inserted successfully needs no call at all: it is already on its way, and
// peers need it.
//
// A publish already in flight cannot be recalled, but it is not retried.
func (p *SimpleAsyncGossiper) Discard(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()

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

// DiscardAll drops every block still waiting to be published. It is for when the
// sequencer abandons the chain it was building on - a derivation reset, or a
// start from an unknown pre-state - after which the queued blocks belong to a
// chain that has been rewound. Publishing them would offer peers blocks the
// sequencer itself no longer stands behind.
func (p *SimpleAsyncGossiper) DiscardAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

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
//
// That means it waits for a publish in flight, and for up to retryInterval if a
// retry is being paced. In practice neither bites at shutdown: Driver.Close
// cancels the context this gossiper was built with before calling Stop, which
// both aborts the publish and skips the pacing.
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

// dequeue takes the oldest queued payload and marks it in flight. It returns nil
// when the queue is empty. Like enqueue, it updates the gauge under the same lock
// as the queue mutation.
func (p *SimpleAsyncGossiper) dequeue() *queuedPayload {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.queue) == 0 {
		p.publishInFlight = false
		return nil
	}
	next := p.queue[0]
	p.queue[0] = nil // do not keep the envelope alive through the backing array
	p.queue = p.queue[1:]
	next.attempts++
	p.publishing = next.hash
	p.publishInFlight = true
	p.publishDiscarded = false
	p.metrics.RecordPublishQueueLen(len(p.queue))
	return next
}

// publish publishes the next queued payload. It runs on the publisher goroutine
// and holds no lock across the network call.
//
// The wake-up for the rest of the queue happens at the end rather than up front,
// so that a retry cannot be picked up by a signal queued before the attempt that
// failed - which would skip the pacing exactly when it is needed.
func (p *SimpleAsyncGossiper) publish() {
	next := p.dequeue()
	if next == nil {
		return // an earlier signal already drained the queue
	}

	ctx, cancel := context.WithTimeout(p.ctx, publishTimeout)
	defer cancel()
	err := p.net.SignAndPublishL2Payload(ctx, next.payload)

	if p.finishPublish(next, err) {
		// The block went back to the front of the queue. Wait before offering it
		// again - see retryInterval.
		select {
		case <-time.After(p.retryInterval):
		case <-p.ctx.Done():
			return
		}
	}
	p.signalIfPending()
}

// signalIfPending wakes the publisher again if the queue is not empty, so it
// comes back through the select and a stop is still honored between publishes.
func (p *SimpleAsyncGossiper) signalIfPending() {
	p.mu.Lock()
	pending := len(p.queue) > 0
	p.mu.Unlock()
	if pending {
		p.signal()
	}
}

// finishPublish records the outcome of a publish attempt: on failure the block is
// put back at the front of the queue if there is room for it. It reports whether
// it requeued, so the caller can pace the retry.
func (p *SimpleAsyncGossiper) finishPublish(next *queuedPayload, err error) (requeued bool) {
	p.mu.Lock()
	p.publishInFlight = false
	discarded := p.publishDiscarded
	p.publishDiscarded = false

	if err == nil {
		p.mu.Unlock()
		return false
	}

	// A block the network will reject the same way every time is dropped now: it
	// cannot become publishable, and holding it at the head would stall the
	// blocks behind it.
	permanent := errors.Is(err, ErrPermanentPublish)

	// Retrying at the front keeps the chain gap-free for peers. Queue pressure is
	// what bounds it: the retried block sits at the head, and enqueue evicts the
	// head when the queue overflows, so a block is retried exactly as long as
	// production leaves room for it. A block the sequencer has already rejected
	// is not retried at all.
	//
	// maxPublishAttempts is a hard ceiling on top of that, for the case where
	// eviction can never come: if block production has stopped, nothing is ever
	// enqueued, so nothing evicts the head and a block that keeps failing would be
	// offered - and logged, and counted - once per retryInterval until the process
	// exits. At a 1s interval it is reached in about 32s, well before 32 blocks
	// have been produced at any normal block time, so it is the ceiling that binds
	// first in practice rather than a restatement of queue pressure.
	retry := !discarded && !permanent &&
		len(p.queue) < maxPublishQueue && next.attempts < maxPublishQueue
	if retry {
		p.queue = append([]*queuedPayload{next}, p.queue...)
		p.metrics.RecordPublishQueueLen(len(p.queue))
	}
	p.mu.Unlock()

	p.metrics.RecordPublishingError()
	if !retry {
		// Nothing else will carry this block to peers.
		p.metrics.RecordDroppedPublish()
	}
	msg := "failed to publish newly created block"
	if permanent {
		msg = "dropping block that can never be published"
	}
	p.log.Warn(msg,
		"id", next.payload.ExecutionPayload.ID(),
		"hash", next.payload.ExecutionPayload.BlockHash,
		"attempts", next.attempts,
		"retrying", retry,
		"err", err)
	return retry
}

// NoOpGossiper is a no-op implementation of AsyncGossiper
// it serves as a placeholder for when the AsyncGossiper is not needed
type NoOpGossiper struct{}

func (NoOpGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {}
func (NoOpGossiper) Discard(hash common.Hash)                     {}
func (NoOpGossiper) DiscardAll()                                  {}
func (NoOpGossiper) Stop()                                        {}
func (NoOpGossiper) Start()                                       {}
