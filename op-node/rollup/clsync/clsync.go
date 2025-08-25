package clsync

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// Max memory used for buffering unsafe payloads
const maxUnsafePayloadsMemory = 500 * 1024 * 1024

type Metrics interface {
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
}

// CLSync holds on to a queue of received unsafe payloads,
// and tries to apply them to the tip of the chain when requested to.
type CLSync struct {
	log     log.Logger
	cfg     *rollup.Config
	metrics Metrics

	emitter event.Emitter

	eng attributes.EngineController

	mu sync.Mutex

	unsafePayloads *PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps and duplicates
}

func NewCLSync(log log.Logger, cfg *rollup.Config, metrics Metrics, eng attributes.EngineController) *CLSync {
	if eng == nil {
		panic("EngControllerInterface must not be nil")
	}
	return &CLSync{
		log:            log,
		cfg:            cfg,
		metrics:        metrics,
		eng:            eng,
		unsafePayloads: NewPayloadsQueue(log, maxUnsafePayloadsMemory, payloadMemSize),
	}
}

func (eq *CLSync) AttachEmitter(em event.Emitter) {
	eq.emitter = em
}

// LowestQueuedUnsafeBlock retrieves the first queued-up L2 unsafe payload, or a zeroed reference if there is none.
func (eq *CLSync) LowestQueuedUnsafeBlock() eth.L2BlockRef {
	payload := eq.unsafePayloads.Peek()
	if payload == nil {
		return eth.L2BlockRef{}
	}
	ref, err := derive.PayloadToBlockRef(eq.cfg, payload.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}
	}
	return ref
}

type ReceivedUnsafePayloadEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
}

func (ev ReceivedUnsafePayloadEvent) String() string {
	return "received-unsafe-payload"
}

func (eq *CLSync) OnEvent(ctx context.Context, ev event.Event) bool {
	// Events may be concurrent in the future. Prevent unsafe concurrent modifications to the payloads queue.
	eq.mu.Lock()
	defer eq.mu.Unlock()

	switch x := ev.(type) {
	case engine.PayloadInvalidEvent:
		eq.onInvalidPayload(x)
	case engine.ForkchoiceUpdateEvent:
		eq.onForkchoiceUpdate(ctx, x)
	case ReceivedUnsafePayloadEvent:
		eq.onUnsafePayload(ctx, x)
	default:
		return false
	}
	return true
}

// onInvalidPayload checks if the first next-up payload matches the invalid payload.
// If so, the payload is dropped, to give the next payloads a try.
func (eq *CLSync) onInvalidPayload(x engine.PayloadInvalidEvent) {
	eq.log.Debug("CL sync received invalid-payload report", "id", x.Envelope.ExecutionPayload.ID())

	block := x.Envelope.ExecutionPayload
	if peek := eq.unsafePayloads.Peek(); peek != nil &&
		block.BlockHash == peek.ExecutionPayload.BlockHash {
		eq.log.Warn("Dropping invalid unsafe payload",
			"hash", block.BlockHash, "number", uint64(block.BlockNumber),
			"timestamp", uint64(block.Timestamp))
		eq.unsafePayloads.Pop()
	}
}

// onForkchoiceUpdate refreshes unsafe payload queue and peeks at the next applicable unsafe payload, if any,
// to apply on top of the received forkchoice pre-state.
// The payload is held on to until the forkchoice changes (success case) or the payload is reported to be invalid.
func (eq *CLSync) onForkchoiceUpdate(ctx context.Context, event engine.ForkchoiceUpdateEvent) {
	eq.log.Debug("CL sync received forkchoice update",
		"unsafe", event.UnsafeL2Head, "safe", event.SafeL2Head, "finalized", event.FinalizedL2Head)

	eq.unsafePayloads.DropInapplicableUnsafePayloads(event)
	nextEnvelope := eq.unsafePayloads.Peek()
	if nextEnvelope == nil {
		eq.log.Debug("No unsafe payload to process")
		return
	}

	// Only process the next payload if it is applicable on top of the current unsafe head.
	// This avoids prematurely attempting to insert non-adjacent payloads (e.g. height gaps),
	// which could otherwise trigger EL sync behavior.
	refParentHash := nextEnvelope.ExecutionPayload.ParentHash
	refBlockNumber := uint64(nextEnvelope.ExecutionPayload.BlockNumber)
	if refParentHash != event.UnsafeL2Head.Hash || refBlockNumber != event.UnsafeL2Head.Number+1 {
		eq.log.Debug("Next unsafe payload is not applicable yet",
			"nextHash", nextEnvelope.ExecutionPayload.BlockHash, "nextNumber", refBlockNumber, "unsafe", event.UnsafeL2Head)
		return
	}

	// We don't pop from the queue. If there is a temporary error then we can retry.
	// Upon next forkchoice update or invalid-payload event we can remove it from the queue.
	eq.emitter.Emit(ctx, engine.ProcessUnsafePayloadEvent{Envelope: nextEnvelope})
}

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1.
func (eq *CLSync) onUnsafePayload(ctx context.Context, event ReceivedUnsafePayloadEvent) {
	eq.log.Debug("CL sync received payload", "payload", event.Envelope.ExecutionPayload.ID())
	if event.Envelope == nil {
		eq.log.Error("cannot add nil unsafe payload")
		return
	}

	if err := eq.unsafePayloads.Push(event.Envelope); err != nil {
		eq.log.Warn("Could not add unsafe payload", "id", event.Envelope.ExecutionPayload.ID(), "timestamp", uint64(event.Envelope.ExecutionPayload.Timestamp), "err", err)
		return
	}
	p := eq.unsafePayloads.Peek()
	eq.metrics.RecordUnsafePayloadsBuffer(uint64(eq.unsafePayloads.Len()), eq.unsafePayloads.MemSize(), p.ExecutionPayload.ID())
	eq.log.Trace("Next unsafe payload to process", "next", p.ExecutionPayload.ID(), "timestamp", uint64(p.ExecutionPayload.Timestamp))

	// request forkchoice update directly so we can process the payload
	eq.eng.RequestForkchoiceUpdate(ctx)
}
