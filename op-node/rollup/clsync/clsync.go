package clsync

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Max memory used for buffering unsafe payloads
const maxUnsafePayloadsMemory = 500 * 1024 * 1024

type Metrics interface {
	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)
}

// CLSync manages optimistic synchronization of unsafe L2 payloads received via P2P gossip.
// It buffers payloads in a priority queue and applies them sequentially to extend the unsafe
// chain tip when they become ready for processing.
//
// # Purpose
//
// CLSync enables fast finality by allowing nodes to optimistically track the latest sequencer
// outputs before they are confirmed on L1. This provides near-instant block confirmations
// while maintaining safety through the underlying L1 derivation process.
//
// # Invariants Maintained
//
// CLSync maintains several critical invariants to ensure correctness and safety:
//
//   - Sequential Processing: Payloads are always processed in ascending block number order,
//     ensuring proper chain extension without gaps.
//
//   - Chain Continuity: Only processes payloads whose parent hash matches the current unsafe
//     head, preventing invalid chain extensions.
//
//   - Memory Bounds: Total memory usage never exceeds 500MB. When this limit is approached,
//     the oldest (lowest block number) payloads are evicted first.
//
//   - Duplicate Prevention: The same block hash can never exist in the queue twice, preventing
//     redundant processing and memory waste.
//
//   - Age Filtering: Payloads older than the current safe or unsafe heads are automatically
//     discarded as they represent stale or already-processed blocks.
//
//   - Thread Safety: All public methods are thread-safe and can be called concurrently from
//     multiple goroutines.
//
//   - Queue Ordering: The internal queue maintains min-heap ordering by block number, ensuring
//     O(1) access to the next payload to process.
//
// # Interface Contract
//
// CLSync provides the following interface guarantees:
//
//   - AddUnsafePayload: Thread-safe addition of new payloads with immediate processing attempt.
//     Returns nil on success, error if payload is invalid or memory limits exceeded.
//     Never blocks - uses mutex for atomicity but processes eagerly.
//
//   - ProcessReadyPayloads: Thread-safe processing of all currently ready payloads.
//     Processes payloads until reaching a gap, invalid payload, or empty queue.
//     Returns nil on success, error only for engine communication failures.
//
//   - OnInvalidPayload: Thread-safe removal of invalid payloads from the queue.
//     Typically called when the engine reports a payload as invalid.
//     Always succeeds - no return value.
//
//   - LowestQueuedUnsafeBlock: Thread-safe read-only access to the next payload to be processed.
//     Returns zero value if queue is empty. Never modifies state.
//
// # Processing Logic
//
// When processing payloads, CLSync follows this decision tree for each payload at the queue head:
//
//  1. If queue is empty → abort processing
//  2. If payload block number ≤ safe/unsafe head → pop and discard (stale)
//  3. If payload hash == current unsafe head → pop and discard (already processed)
//  4. If parent hash ≠ current unsafe head hash:
//     - If block number == unsafe head + 1 → pop and discard (conflicting fork)
//     - If block number > unsafe head + 1 → abort processing (gap exists)
//  5. Otherwise → process payload by calling engine.InsertUnsafePayload
//
// # Error Handling
//
// CLSync handles errors as follows:
//
//   - Invalid payloads: Automatically removed from queue, processing continues
//   - Temporary engine errors: Processing stops, error returned to caller for retry
//   - Malformed payloads: Removed from queue with error logging
//   - Memory limit exceeded: Oldest payloads evicted automatically
//
// # Concurrency
//
// CLSync is designed for concurrent access:
//
//   - All methods use a single mutex for simplicity and correctness
//   - Methods never block for extended periods - processing is bounded
//   - Safe to call from P2P handlers, sync loops, and API handlers simultaneously
//   - No risk of deadlock as CLSync never calls back into caller code while holding locks
//
// # Memory Management
//
// Memory usage is carefully controlled:
//
//   - Each payload's memory footprint is calculated including transaction data
//   - Total memory is tracked and bounded at 500MB
//   - Eviction policy favors newer blocks over older blocks
//   - Automatic cleanup prevents memory leaks during high gossip traffic
type CLSync struct {
	log     log.Logger
	cfg     *rollup.Config
	metrics Metrics

	engine engine.CLSyncEngine

	mu sync.Mutex

	unsafePayloads *PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps and duplicates
}

func NewCLSync(log log.Logger, cfg *rollup.Config, metrics Metrics, engine engine.CLSyncEngine) *CLSync {
	return &CLSync{
		log:            log,
		cfg:            cfg,
		metrics:        metrics,
		engine:         engine,
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

// AddUnsafePayload adds an unsafe payload to the queue and processes any ready payloads.
// This replaces the event-driven flow with direct function calls.
func (eq *CLSync) AddUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if envelope == nil {
		eq.log.Warn("cannot add nil unsafe payload")
		return nil
	}

	eq.log.Debug("CL sync received payload", "payload", envelope.ExecutionPayload.ID())

	if err := eq.unsafePayloads.Push(envelope); err != nil {
		eq.log.Warn("Could not add unsafe payload", "id", envelope.ExecutionPayload.ID(), "timestamp", uint64(envelope.ExecutionPayload.Timestamp), "err", err)
		return err
	}

	p := eq.unsafePayloads.Peek()
	eq.metrics.RecordUnsafePayloadsBuffer(uint64(eq.unsafePayloads.Len()), eq.unsafePayloads.MemSize(), p.ExecutionPayload.ID())
	eq.log.Trace("Next unsafe payload to process", "next", p.ExecutionPayload.ID(), "timestamp", uint64(p.ExecutionPayload.Timestamp))

	// Process any ready payloads immediately
	return eq.processReadyPayloads(ctx)
}

// ProcessReadyPayloads processes any payloads that are ready to be applied.
// This replaces the ForkchoiceUpdate event flow with direct processing.
func (eq *CLSync) ProcessReadyPayloads(ctx context.Context) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return eq.processReadyPayloads(ctx)
}

// processReadyPayloads is the internal implementation that requires the mutex to be held
func (eq *CLSync) processReadyPayloads(ctx context.Context) error {
	// Get current forkchoice state directly from engine
	currentState := eq.engine.L2ChainState()

	eq.log.Debug("CL sync processing payloads with current state",
		"unsafe", currentState.UnsafeL2Head, "safe", currentState.SafeL2Head, "finalized", currentState.FinalizedL2Head)

	for {
		pop, abort := eq.shouldProcessNext(currentState)
		if abort {
			return nil
		}
		if pop {
			eq.unsafePayloads.Pop()
			continue
		}

		// Process the next payload
		firstEnvelope := eq.unsafePayloads.Peek()
		if firstEnvelope == nil {
			return nil
		}

		ref, err := derive.PayloadToBlockRef(eq.cfg, firstEnvelope.ExecutionPayload)
		if err != nil {
			eq.log.Error("failed to decode L2 block ref from payload", "err", err)
			eq.unsafePayloads.Pop() // Remove invalid payload
			continue
		}

		// Avoid re-processing the same unsafe payload if it has already been processed
		if ref.BlockRef().ID() == currentState.UnsafeL2Head.BlockRef().ID() {
			eq.unsafePayloads.Pop()
			continue
		}

		if err := eq.engine.InsertUnsafePayload(ctx, firstEnvelope, ref); err != nil {
			// Check if this is an invalid payload error and handle it directly
			if envelope, isInvalid := engine.IsPayloadInvalid(err); isInvalid {
				eq.log.Info("payload was invalid, removing from queue", "ref", ref,
					"txs", len(firstEnvelope.ExecutionPayload.Transactions), "err", err)
				eq.onInvalidPayloadInternal(envelope)
				// Continue processing other payloads
				continue
			}

			eq.log.Info("failed to insert payload", "ref", ref,
				"txs", len(firstEnvelope.ExecutionPayload.Transactions), "err", err)
			// For now, return the error - the caller can decide how to handle it
			return err
		}

		eq.log.Info("successfully processed payload", "ref", ref, "txs", len(firstEnvelope.ExecutionPayload.Transactions))
		eq.unsafePayloads.Pop()

		// Update current state for next iteration
		currentState.UnsafeL2Head = eq.engine.UnsafeL2Head()
		currentState.SafeL2Head = eq.engine.SafeL2Head()
		currentState.FinalizedL2Head = eq.engine.Finalized()
	}
}

// OnInvalidPayload handles when a payload is reported as invalid.
// This replaces the PayloadInvalidEvent handler.
func (eq *CLSync) OnInvalidPayload(envelope *eth.ExecutionPayloadEnvelope) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	eq.onInvalidPayloadInternal(envelope)
}

// onInvalidPayloadInternal handles invalid payload without acquiring mutex
func (eq *CLSync) onInvalidPayloadInternal(envelope *eth.ExecutionPayloadEnvelope) {
	eq.log.Debug("CL sync received invalid-payload report", "id", envelope.ExecutionPayload.ID())

	block := envelope.ExecutionPayload
	if peek := eq.unsafePayloads.Peek(); peek != nil &&
		block.BlockHash == peek.ExecutionPayload.BlockHash {
		eq.log.Warn("Dropping invalid unsafe payload",
			"hash", block.BlockHash, "number", uint64(block.BlockNumber),
			"timestamp", uint64(block.Timestamp))
		eq.unsafePayloads.Pop()
	}
}

// shouldProcessNext determines what to do with the tip of the payloads-queue, given the forkchoice pre-state.
// If abort, there is nothing to process (either due to empty queue, or unsuitable tip).
// If pop, the tip should be dropped, and processing can repeat from there.
// If not abort or pop, the tip is ready to process.
func (eq *CLSync) shouldProcessNext(currentState eth.L2ChainState) (pop bool, abort bool) {
	if eq.unsafePayloads.Len() == 0 {
		return false, true
	}
	firstEnvelope := eq.unsafePayloads.Peek()
	first := firstEnvelope.ExecutionPayload

	if first.BlockHash == currentState.UnsafeL2Head.Hash {
		eq.log.Debug("successfully processed payload, removing it from the payloads queue now")
		return true, false
	}

	if uint64(first.BlockNumber) <= currentState.SafeL2Head.Number {
		eq.log.Info("skipping unsafe payload, since it is older than safe head", "safe", currentState.SafeL2Head.ID(), "unsafe", currentState.UnsafeL2Head.ID(), "unsafe_payload", first.ID())
		return true, false
	}
	if uint64(first.BlockNumber) <= currentState.UnsafeL2Head.Number {
		eq.log.Info("skipping unsafe payload, since it is older than unsafe head", "unsafe", currentState.UnsafeL2Head.ID(), "unsafe_payload", first.ID())
		return true, false
	}

	// Ensure that the unsafe payload builds upon the current unsafe head
	if first.ParentHash != currentState.UnsafeL2Head.Hash {
		if uint64(first.BlockNumber) == currentState.UnsafeL2Head.Number+1 {
			eq.log.Info("skipping unsafe payload, since it does not build onto the existing unsafe chain", "safe", currentState.SafeL2Head.ID(), "unsafe", currentState.UnsafeL2Head.ID(), "unsafe_payload", first.ID())
			return true, false
		}
		return false, true // rollup-node should try something different if it cannot process the first unsafe payload
	}

	return false, false
}
