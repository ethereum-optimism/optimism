package async

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// publishTimeout bounds a single publish attempt: signing the block (a network
// round-trip when a remote signer is configured) plus the p2p publish. It is
// generous compared with any block time, so it never cuts a working signer
// short. Its job is to stop a signer connection that hangs rather than errors
// from wedging publishing until TCP gives up.
const publishTimeout = 10 * time.Second

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
	// pending is the payload waiting to be picked up by the publisher goroutine
	pending *pendingPayload
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
}

// pendingPayload is a payload handed to the publisher goroutine, tagged with the
// epoch it was handed over in.
type pendingPayload struct {
	payload *eth.ExecutionPayloadEnvelope
	epoch   uint64
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

// Gossip hands a payload to the publisher goroutine. It does not wait for the
// payload to be published.
//
// A payload that has not started publishing yet is replaced: the newer block is
// the one peers need, and they reach the older one through sync.
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	p.mu.Lock()
	if p.pending != nil {
		p.log.Warn("dropping unpublished block, superseded before it was published",
			"dropped", p.pending.payload.ExecutionPayload.ID(),
			"id", payload.ExecutionPayload.ID())
	}
	p.epoch++
	p.pending = &pendingPayload{payload: payload, epoch: p.epoch}
	p.mu.Unlock()

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

// publish publishes the pending payload, if there is one, and stores it for
// reuse if the publish succeeded and the payload is still the one the sequencer
// cares about. It runs on the publisher goroutine and holds no lock across the
// network call.
func (p *SimpleAsyncGossiper) publish() {
	p.mu.Lock()
	pending := p.pending
	p.pending = nil
	p.mu.Unlock()
	if pending == nil {
		return // an earlier signal already took it
	}

	ctx, cancel := context.WithTimeout(p.ctx, publishTimeout)
	defer cancel()
	if err := p.net.SignAndPublishL2Payload(ctx, pending.payload); err != nil {
		p.log.Warn("failed to publish newly created block",
			"id", pending.payload.ExecutionPayload.ID(),
			"hash", pending.payload.ExecutionPayload.BlockHash,
			"err", err)
		p.metrics.RecordPublishingError()
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	// A Clear (the block was inserted) or a newer Gossip while we were
	// publishing makes this payload unfit for reuse: the sequencer would rebuild
	// a block it has already moved past.
	if p.epoch == pending.epoch {
		p.currentPayload = pending.payload
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
