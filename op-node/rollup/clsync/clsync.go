package clsync

import (
	"context"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Max memory used for buffering unsafe payloads
const maxUnsafePayloadsMemory = 500 * 1024 * 1024

type Metrics interface {
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
}

type Engine interface {
	derive.EngineState
	InsertUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error
}

// CLSync holds on to a queue of received unsafe payloads,
// and tries to apply them to the tip of the chain when requested to.
type CLSync struct {
	log            log.Logger
	cfg            *rollup.Config
	metrics        Metrics
	ec             Engine
	unsafePayloads *PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps and duplicates
}

func NewCLSync(log log.Logger, cfg *rollup.Config, metrics Metrics, ec Engine) *CLSync {
	return &CLSync{
		log:            log,
		cfg:            cfg,
		metrics:        metrics,
		ec:             ec,
		unsafePayloads: NewPayloadsQueue(log, maxUnsafePayloadsMemory, payloadMemSize),
	}
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

// AddUnsafePayload schedules an execution payload to be processed, ahead of deriving it from L1.
func (eq *CLSync) AddUnsafePayload(envelope *eth.ExecutionPayloadEnvelope) {
	if envelope == nil {
		eq.log.Warn("cannot add nil unsafe payload")
		return
	}

	if err := eq.unsafePayloads.Push(envelope); err != nil {
		eq.log.Warn("Could not add unsafe payload", "id", envelope.ExecutionPayload.ID(), "timestamp", uint64(envelope.ExecutionPayload.Timestamp), "err", err)
		return
	}
	p := eq.unsafePayloads.Peek()
	eq.metrics.RecordUnsafePayloadsBuffer(uint64(eq.unsafePayloads.Len()), eq.unsafePayloads.MemSize(), p.ExecutionPayload.ID())
	eq.log.Trace("Next unsafe payload to process", "next", p.ExecutionPayload.ID(), "timestamp", uint64(p.ExecutionPayload.Timestamp))
}

// Proceed dequeues the next applicable unsafe payload, if any, to apply to the tip of the chain.
// EOF error means we can't process the next unsafe payload. The caller should then try a different form of syncing.
func (eq *CLSync) Proceed(ctx context.Context) error {
	if eq.unsafePayloads.Len() == 0 {
		return io.EOF
	}
	firstEnvelope := eq.unsafePayloads.Peek()
	first := firstEnvelope.ExecutionPayload

	if uint64(first.BlockNumber) <= eq.ec.SafeL2Head().Number {
		eq.log.Info("skipping unsafe payload, since it is older than safe head", "safe", eq.ec.SafeL2Head().ID(), "unsafe", eq.ec.UnsafeL2Head().ID(), "unsafe_payload", first.ID())
		eq.unsafePayloads.Pop()
		return nil
	}
	if uint64(first.BlockNumber) <= eq.ec.UnsafeL2Head().Number {
		eq.log.Info("skipping unsafe payload, since it is older than unsafe head", "unsafe", eq.ec.UnsafeL2Head().ID(), "unsafe_payload", first.ID())
		eq.unsafePayloads.Pop()
		return nil
	}

	// Ensure that the unsafe payload builds upon the current unsafe head
	if first.ParentHash != eq.ec.UnsafeL2Head().Hash {
		if uint64(first.BlockNumber) == eq.ec.UnsafeL2Head().Number+1 {
			eq.log.Info("skipping unsafe payload, since it does not build onto the existing unsafe chain", "safe", eq.ec.SafeL2Head().ID(), "unsafe", eq.ec.UnsafeL2Head().ID(), "unsafe_payload", first.ID())
			eq.unsafePayloads.Pop()
		}
		return io.EOF // time to go to next stage if we cannot process the first unsafe payload
	}

	ref, err := derive.PayloadToBlockRef(eq.cfg, first)
	if err != nil {
		eq.log.Error("failed to decode L2 block ref from payload", "err", err)
		eq.unsafePayloads.Pop()
		return nil
	}

	if err := eq.ec.InsertUnsafePayload(ctx, firstEnvelope, ref); errors.Is(err, derive.ErrTemporary) {
		eq.log.Debug("Temporary error while inserting unsafe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)
		return err
	} else if err != nil {
		eq.log.Warn("Dropping invalid unsafe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)
		eq.unsafePayloads.Pop()
		return err
	}
	eq.unsafePayloads.Pop()
	eq.log.Trace("Executed unsafe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)
	return nil
}
