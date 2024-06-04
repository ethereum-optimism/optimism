package async

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Get() *eth.ExecutionPayloadEnvelope
	Clear()
	Stop()
	Start()
}

// SimpleAsyncGossiper is a component that stores and gossips a single payload at a time
// it uses a separate goroutine to handle gossiping the payload asynchronously
// the payload can be accessed by the Get function to be reused when the payload was gossiped but not inserted
// exposed functions are synchronous, and block until the async routine is able to start handling the request
type SimpleAsyncGossiper struct {
	running atomic.Bool
	// channel to add new payloads to gossip
	set chan *eth.ExecutionPayloadEnvelope
	// channel to request getting the currently gossiping payload
	get chan chan *eth.ExecutionPayloadEnvelope
	// channel to request clearing the currently gossiping payload
	clear chan struct{}
	// channel to request stopping the handling loop
	stop chan struct{}

	currentPayload *eth.ExecutionPayloadEnvelope
	ctx            context.Context
	net            Network
	log            log.Logger
	metrics        Metrics
}

// To avoid import cycles, we define a new Network interface here
// this interface is compatible with driver.Network
type Network interface {
	PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
}

// To avoid import cycles, we define a new Metrics interface here
// this interface is compatible with driver.Metrics
type Metrics interface {
	RecordPublishingError()
}

func NewAsyncGossiper(ctx context.Context, net Network, log log.Logger, metrics Metrics) *SimpleAsyncGossiper {
	return &SimpleAsyncGossiper{
		running: atomic.Bool{},
		set:     make(chan *eth.ExecutionPayloadEnvelope),
		get:     make(chan chan *eth.ExecutionPayloadEnvelope),
		clear:   make(chan struct{}),
		stop:    make(chan struct{}),

		currentPayload: nil,
		net:            net,
		ctx:            ctx,
		log:            log,
		metrics:        metrics,
	}
}

// Gossip is a synchronous function to store and gossip a payload
// it blocks until the payload can be taken by the async routine
func (p *SimpleAsyncGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	p.set <- payload
}

// Get is a synchronous function to get the currently held payload
// it blocks until the async routine is able to return the payload
func (p *SimpleAsyncGossiper) Get() *eth.ExecutionPayloadEnvelope {
	c := make(chan *eth.ExecutionPayloadEnvelope)
	p.get <- c
	return <-c
}

// Clear is a synchronous function to clear the currently gossiping payload
// it blocks until the signal to clear is picked up by the async routine
func (p *SimpleAsyncGossiper) Clear() {
	p.clear <- struct{}{}
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

// Start starts the AsyncGossiper's gossiping loop on a separate goroutine
// each behavior of the loop is handled by a select case on a channel, plus an internal handler function call
func (p *SimpleAsyncGossiper) Start() {
	// if the gossiping is already running, return
	if !p.running.CompareAndSwap(false, true) {
		return
	}
	// else, start the handling loop
	go func() {
		defer p.running.Store(false)
		for {
			select {
			// new payloads to be gossiped are found in the `set` channel
			case payload := <-p.set:
				p.gossip(p.ctx, payload)
			// requests to get the current payload are found in the `get` channel
			case c := <-p.get:
				p.getPayload(c)
			// requests to clear the current payload are found in the `clear` channel
			case <-p.clear:
				p.clearPayload()
			// if the context is done, return
			case <-p.stop:
				return
			}
		}
	}()
}

// gossip is the internal handler function for gossiping the current payload
// and storing the payload in the async AsyncGossiper's state
// it is called by the Start loop when a new payload is set
// the payload is only stored if the publish is successful
func (p *SimpleAsyncGossiper) gossip(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
	if err := p.net.PublishL2Payload(ctx, payload); err == nil {
		p.currentPayload = payload
	} else {
		p.log.Warn("failed to publish newly created block",
			"id", payload.ExecutionPayload.ID(),
			"hash", payload.ExecutionPayload.BlockHash,
			"err", err)
		p.metrics.RecordPublishingError()
	}
}

// getPayload is the internal handler function for getting the current payload
// c is the channel the caller expects to receive the payload on
func (p *SimpleAsyncGossiper) getPayload(c chan *eth.ExecutionPayloadEnvelope) {
	c <- p.currentPayload
}

// clearPayload is the internal handler function for clearing the current payload
func (p *SimpleAsyncGossiper) clearPayload() {
	p.currentPayload = nil
}

// NoOpGossiper is a no-op implementation of AsyncGossiper
// it serves as a placeholder for when the AsyncGossiper is not needed
type NoOpGossiper struct{}

func (NoOpGossiper) Gossip(payload *eth.ExecutionPayloadEnvelope) {}
func (NoOpGossiper) Get() *eth.ExecutionPayloadEnvelope           { return nil }
func (NoOpGossiper) Clear()                                       {}
func (NoOpGossiper) Stop()                                        {}
func (NoOpGossiper) Start()                                       {}
