package payloads

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// Max memory used for buffering unsafe payloads, per chain
const maxUnsafePayloadsMemory = 100 * 1024 * 1024

// Payloads acts as a cache, for a single chain, of the recently received P2P blocks.
// The cache size is defined in approximate memory consumption.
// The blocks with the highest block number are retained.
type Payloads struct {
	ctx     context.Context
	chainID eth.ChainID

	log log.Logger

	queue *clsync.PayloadsQueue

	emitter event.Emitter
}

var _ event.AttachEmitter = (*Payloads)(nil)
var _ event.Deriver = (*Payloads)(nil)

func NewPayloads(ctx context.Context, logger log.Logger, chainID eth.ChainID) *Payloads {
	return &Payloads{
		ctx:     ctx,
		chainID: chainID,
		log:     logger.New("chainID", chainID),
		queue:   clsync.NewPayloadsQueue(logger, maxUnsafePayloadsMemory, clsync.PayloadMemSize),
		emitter: nil,
	}
}

func (p *Payloads) AttachEmitter(em event.Emitter) {
	p.emitter = em
}

func (p *Payloads) ChainID() eth.ChainID {
	return p.chainID
}

func (p *Payloads) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case PayloadRequestEvent:
		if p.chainID != x.ChainID {
			return false
		}
		// payload may be nil, if not found
		payload := p.queue.ByNumber(x.Num)
		p.emitter.Emit(ctx, PayloadResponseEvent{
			ChainID:  p.chainID,
			Num:      x.Num,
			Envelope: payload,
		})
	case p2p.ReceivedBlockEvent:
		if p.chainID != x.ChainID {
			return false
		}
		// this will pop the oldest payloads until there is room
		if err := p.queue.Push(x.Envelope); err != nil {
			p.log.Warn("Could not buffer payload", "err", err)
		} else {
			// Queue is non-empty now, so these are not nil
			first := p.queue.Peek()
			last := p.queue.Last()
			p.emitter.Emit(ctx, PayloadsUpdateEvent{
				ChainID: p.chainID,
				Min:     first.ID(),
				Max:     last.ID(),
				Count:   uint64(p.queue.Len()),
			})
		}
	default:
		return false
	}
	return true
}

type PayloadsUpdateEvent struct {
	ChainID eth.ChainID
	// Lowest available number
	Min eth.BlockID
	// Highest available number (there may be gaps in-between Min and Max)
	Max eth.BlockID
	// Number of blocks we have buffered for this chain
	Count uint64
}

func (ev PayloadsUpdateEvent) String() string {
	return "payloads-update"
}

type PayloadRequestEvent struct {
	ChainID eth.ChainID
	Num     uint64
}

func (ev PayloadRequestEvent) String() string {
	return "payload-request"
}

type PayloadResponseEvent struct {
	ChainID eth.ChainID
	Num     uint64

	// nil if the payload is not known
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev PayloadResponseEvent) String() string {
	return "payload-response"
}
