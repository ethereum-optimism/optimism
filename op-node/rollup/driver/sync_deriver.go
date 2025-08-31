package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/log"
)

type SyncDeriver struct {
	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2Safe.
	Derivation DerivationPipeline

	SafeHeadNotifs rollup.SafeHeadListener // notified when safe head is updated

	CLSync CLSync

	// The engine controller is used by the sequencer & Derivation components.
	// We will also use it for EL sync in a future PR.
	Engine *engine.EngineController

	// Sync Mod Config
	SyncCfg *sync.Config

	Config *rollup.Config

	L1 L1Chain
	// Track L1 view when new unsafe L1 block is observed
	L1Tracker *status.L1Tracker
	L2        L2Chain

	Emitter event.Emitter

	Log log.Logger

	Ctx context.Context

	// When in interop, and managed by an op-supervisor,
	// the node performs a reset based on the instructions of the op-supervisor.
	ManagedBySupervisor bool

	StepDeriver StepDeriver
}

func (s *SyncDeriver) AttachEmitter(em event.Emitter) {
	s.Emitter = em
}

func (s *SyncDeriver) OnL1Unsafe(ctx context.Context) {
	// a new L1 head may mean we have the data to not get an EOF again.
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) OnL1Finalized(ctx context.Context) {
	// On "safe" L1 blocks: no step, justified L1 information does not do anything for L2 derivation or status.
	// On "finalized" L1 blocks: we may be able to mark more L2 data as finalized now.
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO(#16917) Remove Event System Refactor Comments
	//  ELSyncStartedEvent is removed and OnELSyncStarted is synchronously called at EngineController
	//  ReceivedBlockEvent is removed and OnUnsafeL2Payload is synchronously called at NewBlockReceiver
	//  L1UnsafeEvent is removed and OnL1Unsafe is synchronously called at L1Handler
	//  FinalizeL1Event is removed and OnL1Finalized is synchronously called at L1Handler
	switch x := ev.(type) {
	case StepEvent:
		s.SyncStep()
	case rollup.ResetEvent:
		s.onResetEvent(ctx, x)
	case rollup.L1TemporaryErrorEvent:
		s.Log.Warn("L1 temporary error", "err", x.Err)
		s.StepDeriver.RequestStep(ctx, false)
	case rollup.EngineTemporaryErrorEvent:
		s.Log.Warn("Engine temporary error", "err", x.Err)
		// Make sure that for any temporarily failed attributes we retry processing.
		// This will be triggered by a step. After appropriate backoff.
		s.StepDeriver.RequestStep(ctx, false)
	case engine.EngineResetConfirmedEvent:
		s.onEngineConfirmedReset(ctx, x)
	case derive.DeriverIdleEvent:
		// Once derivation is idle the system is healthy
		// and we can wait for new inputs. No backoff necessary.
		s.StepDeriver.ResetStepBackoff(ctx)
	case derive.DeriverMoreEvent:
		// If there is more data to process,
		// continue derivation quickly
		s.StepDeriver.RequestStep(ctx, true)
	case engine.SafeDerivedEvent:
		s.onSafeDerivedBlock(ctx, x)
	case derive.ProvideL1Traversal:
		s.StepDeriver.RequestStep(ctx, false)
	default:
		return false
	}
	return true
}

func (s *SyncDeriver) OnUnsafeL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) {
	// If we are doing CL sync or done with engine syncing, fallback to the unsafe payload queue & CL P2P sync.
	if s.SyncCfg.SyncMode == sync.CLSync || !s.Engine.IsEngineSyncing() {
		s.Log.Info("Optimistically queueing unsafe L2 execution payload", "id", envelope.ExecutionPayload.ID())
		s.Emitter.Emit(ctx, clsync.ReceivedUnsafePayloadEvent{Envelope: envelope})
	} else if s.SyncCfg.SyncMode == sync.ELSync {
		ref, err := derive.PayloadToBlockRef(s.Config, envelope.ExecutionPayload)
		if err != nil {
			s.Log.Info("Failed to turn execution payload into a block ref", "id", envelope.ExecutionPayload.ID(), "err", err)
			return
		}
		if ref.Number <= s.Engine.UnsafeL2Head().Number {
			return
		}
		s.Log.Info("Optimistically inserting unsafe L2 execution payload to drive EL sync", "id", envelope.ExecutionPayload.ID())
		if err := s.Engine.InsertUnsafePayload(s.Ctx, envelope, ref); err != nil {
			s.Log.Warn("Failed to insert unsafe payload for EL sync", "id", envelope.ExecutionPayload.ID(), "err", err)
		}
	}
}

func (s *SyncDeriver) onSafeDerivedBlock(ctx context.Context, x engine.SafeDerivedEvent) {
	if s.SafeHeadNotifs != nil && s.SafeHeadNotifs.Enabled() {
		if err := s.SafeHeadNotifs.SafeHeadUpdated(x.Safe, x.Source.ID()); err != nil {
			// At this point our state is in a potentially inconsistent state as we've updated the safe head
			// in the execution client but failed to post process it. Reset the pipeline so the safe head rolls back
			// a little (it always rolls back at least 1 block) and then it will retry storing the entry
			s.Emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("safe head notifications failed: %w", err),
			})
		}
	}
}

func (s *SyncDeriver) OnELSyncStarted() {
	// The EL sync may progress the safe head in the EL without deriving those blocks from L1
	// which means the safe head db will miss entries so we need to remove all entries to avoid returning bad data
	s.Log.Warn("Clearing safe head db because EL sync started")
	if s.SafeHeadNotifs != nil {
		if err := s.SafeHeadNotifs.SafeHeadReset(eth.L2BlockRef{}); err != nil {
			s.Log.Error("Failed to notify safe-head reset when optimistically syncing")
		}
	}
}

func (s *SyncDeriver) onEngineConfirmedReset(ctx context.Context, x engine.EngineResetConfirmedEvent) {
	// If the listener update fails, we return,
	// and don't confirm the engine-reset with the derivation pipeline.
	// The pipeline will re-trigger a reset as necessary.
	if s.SafeHeadNotifs != nil {
		if err := s.SafeHeadNotifs.SafeHeadReset(x.CrossSafe); err != nil {
			s.Log.Error("Failed to warn safe-head notifier of safe-head reset", "safe", x.CrossSafe)
			return
		}
		if s.SafeHeadNotifs.Enabled() && x.CrossSafe.ID() == s.Config.Genesis.L2 {
			// The rollup genesis block is always safe by definition. So if the pipeline resets this far back we know
			// we will process all safe head updates and can record genesis as always safe from L1 genesis.
			// Note that it is not safe to use cfg.Genesis.L1 here as it is the block immediately before the L2 genesis
			// but the contracts may have been deployed earlier than that, allowing creating a dispute game
			// with a L1 head prior to cfg.Genesis.L1
			l1Genesis, err := s.L1.L1BlockRefByNumber(s.Ctx, 0)
			if err != nil {
				s.Log.Error("Failed to retrieve L1 genesis, cannot notify genesis as safe block", "err", err)
				return
			}
			if err := s.SafeHeadNotifs.SafeHeadUpdated(x.CrossSafe, l1Genesis.ID()); err != nil {
				s.Log.Error("Failed to notify safe-head listener of safe-head", "err", err)
				return
			}
		}
	}
	s.Log.Info("Confirming pipeline reset")
	s.Emitter.Emit(ctx, derive.ConfirmPipelineResetEvent{})
}

func (s *SyncDeriver) onResetEvent(ctx context.Context, x rollup.ResetEvent) {
	if s.ManagedBySupervisor {
		s.Log.Warn("Encountered reset when managed by op-supervisor, waiting for op-supervisor", "err", x.Err)
		// IndexingMode will pick up the ResetEvent
		return
	}
	// If the system corrupts, e.g. due to a reorg, simply reset it
	s.Log.Warn("Deriver system is resetting", "err", x.Err)
	s.Emitter.Emit(ctx, engine.ResetEngineRequestEvent{})
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) tryBackupUnsafeReorg() {
	// If we don't need to call FCU to restore unsafeHead using backupUnsafe, keep going b/c
	// this was a no-op(except correcting invalid state when backupUnsafe is empty but TryBackupUnsafeReorg called).
	fcuCalled, err := s.Engine.TryBackupUnsafeReorg(s.Ctx)
	// Dealing with legacy here: it used to skip over the error-handling if fcuCalled was false.
	// But that combination is not actually a code-path in TryBackupUnsafeReorg.
	// We should drop fcuCalled, and make the function emit events directly,
	// once there are no more synchronous callers.
	if !fcuCalled && err != nil {
		s.Log.Crit("unexpected TryBackupUnsafeReorg error after no FCU call", "err", err)
	}
	if err != nil {
		// If we needed to perform a network call, then we should yield even if we did not encounter an error.
		if errors.Is(err, derive.ErrReset) {
			s.Emitter.Emit(s.Ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			s.Emitter.Emit(s.Ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			s.Emitter.Emit(s.Ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected TryBackupUnsafeReorg error type: %w", err),
			})
		}
	}
}

// SyncStep performs the sequence of encapsulated syncing steps.
// Warning: this sequence will be broken apart as outlined in op-node derivers design doc.
func (s *SyncDeriver) SyncStep() {
	s.Log.Debug("Sync process step")

	s.tryBackupUnsafeReorg()

	s.Engine.TryUpdateEngine(s.Ctx)

	if s.Engine.IsEngineSyncing() {
		// The pipeline cannot move forwards if doing EL sync.
		s.Log.Debug("Rollup driver is backing off because execution engine is syncing.",
			"unsafe_head", s.Engine.UnsafeL2Head())
		s.StepDeriver.ResetStepBackoff(s.Ctx)
		return
	}

	// Any now processed forkchoice updates will trigger CL-sync payload processing, if any payload is queued up.

	// Since we don't force attributes to be processed at this point,
	// we cannot safely directly trigger the derivation, as that may generate new attributes that
	// conflict with what attributes have not been applied yet.
	// Instead, we request the engine to repeat where its pending-safe head is at.
	// Upon the pending-safe signal the attributes deriver can then ask the pipeline
	// to generate new attributes, if no attributes are known already.
	s.Engine.RequestPendingSafeUpdate(s.Ctx)

}
