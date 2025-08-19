package driver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	opnodemetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/metrics/metered"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/confdepth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// aliases to not disrupt op-conductor code
var (
	ErrSequencerAlreadyStarted = sequencing.ErrSequencerAlreadyStarted
	ErrSequencerAlreadyStopped = sequencing.ErrSequencerAlreadyStopped
)

type Metrics interface {
	RecordPipelineReset()
	RecordPublishingError()
	RecordDerivationError()

	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)
	RecordChannelInputBytes(inputCompressedBytes int)
	RecordHeadChannelOpened()
	RecordChannelTimedOut()
	RecordFrame()

	RecordDerivedBatches(batchType string)

	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)

	SetDerivationIdle(idle bool)
	SetSequencerState(active bool)

	RecordL1ReorgDepth(d uint64)

	opnodemetrics.Metricer
	metered.L1FetcherMetrics
	event.Metrics
	sequencing.Metrics
}

type L1Chain interface {
	derive.L1Fetcher
	L1BlockRefByLabel(context.Context, eth.BlockLabel) (eth.L1BlockRef, error)
}

type L2Chain interface {
	engine.Engine
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
}

type DerivationPipeline interface {
	Reset()
	Step(ctx context.Context, pendingSafeHead eth.L2BlockRef) (*derive.AttributesWithParent, error)
	Origin() eth.L1BlockRef
	DerivationReady() bool
	ConfirmEngineReset()
}

type EngineController interface {
	engine.RollupAPI
	engine.LocalEngineControl
	IsEngineSyncing() bool
	InsertUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error
	TryBackupUnsafeReorg(ctx context.Context) (bool, error)
}

type CLSync interface {
	LowestQueuedUnsafeBlock() eth.L2BlockRef
}

type AttributesHandler interface {
	// HasAttributes returns if there are any block attributes to process.
	// HasAttributes is for EngineQueue testing only, and can be removed when attribute processing is fully independent.
	HasAttributes() bool
	// SetAttributes overwrites the set of attributes. This may be nil, to clear what may be processed next.
	SetAttributes(attributes *derive.AttributesWithParent)
	// Proceed runs one attempt of processing attributes, if any.
	// Proceed returns io.EOF if there are no attributes to process.
	Proceed(ctx context.Context) error
}

type Finalizer interface {
	FinalizedL1() eth.L1BlockRef
	OnL1Finalized(x eth.L1BlockRef)
	event.Deriver
}

type AltDAIface interface {
	// Notify L1 finalized head so AltDA finality is always behind L1
	Finalize(ref eth.L1BlockRef)
	// Set the engine finalization signal callback
	OnFinalizedHeadSignal(f altda.HeadSignalFn)

	derive.AltDAInputFetcher
}

type SyncStatusTracker interface {
	event.Deriver
	SyncStatus() *eth.SyncStatus
	L1Head() eth.L1BlockRef
	OnL1Unsafe(x eth.L1BlockRef)
	OnL1Safe(x eth.L1BlockRef)
	OnL1Finalized(x eth.L1BlockRef)
}

type Network interface {
	// SignAndPublishL2Payload is called by the driver whenever there is a new payload to publish, synchronously with the driver main loop.
	SignAndPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
}

type AltSync interface {
	// RequestL2Range informs the sync source that the given range of L2 blocks is missing,
	// and should be retrieved from any available alternative syncing source.
	// The start and end of the range are exclusive:
	// the start is the head we already have, the end is the first thing we have queued up.
	// It's the task of the alt-sync mechanism to use this hint to fetch the right payloads.
	// Note that the end and start may not be consistent: in this case the sync method should fetch older history
	//
	// If the end value is zeroed, then the sync-method may determine the end free of choice,
	// e.g. sync till the chain head meets the wallclock time. This functionality is optional:
	// a fixed target to sync towards may be determined by picking up payloads through P2P gossip or other sources.
	//
	// The sync results should be returned back to the driver via the OnUnsafeL2Payload(ctx, payload) method.
	// The latest requested range should always take priority over previous requests.
	// There may be overlaps in requested ranges.
	// An error may be returned if the scheduling fails immediately, e.g. a context timeout.
	RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error
}

type SequencerStateListener interface {
	SequencerStarted() error
	SequencerStopped() error
}

type Drain interface {
	Drain() error
	Await() <-chan struct{}
}

// NewDriver composes an events handler that tracks L1 state, triggers L2 Derivation, and optionally sequences new L2 blocks.
func NewDriver(
	sys event.Registry,
	drain Drain,
	driverCfg *Config,
	cfg *rollup.Config,
	depSet derive.DependencySet,
	l2 L2Chain,
	l1 L1Chain,
	l1Blobs derive.L1BlobsFetcher,
	altSync AltSync,
	network Network,
	log log.Logger,
	metrics Metrics,
	sequencerStateListener sequencing.SequencerStateListener,
	safeHeadListener rollup.SafeHeadListener,
	syncCfg *sync.Config,
	sequencerConductor conductor.SequencerConductor,
	altDA AltDAIface,
	indexingMode bool,
) *Driver {
	driverCtx, driverCancel := context.WithCancel(context.Background())

	statusTracker := status.NewStatusTracker(log, metrics)
	sys.Register("status", statusTracker)

	l1Tracker := status.NewL1Tracker(l1)

	l1 = metered.NewMeteredL1Fetcher(l1Tracker, metrics)
	verifConfDepth := confdepth.NewConfDepth(driverCfg.VerifierConfDepth, statusTracker.L1Head, l1)

	ec := engine.NewEngineController(driverCtx, l2, log, metrics, cfg, syncCfg, sys.Register("engine-controller", nil))
	// TODO(#17115): Refactor dependency cycles
	ec.SetCrossUpdateHandler(statusTracker)

	engineReset := engine.NewEngineResetDeriver(driverCtx, log, cfg, l1, l2, syncCfg)
	engineReset.SetEngController(ec)
	sys.Register("engine-reset", engineReset)

	clSync := clsync.NewCLSync(log, cfg, metrics, ec) // alt-sync still uses cl-sync state to determine what to sync to
	sys.Register("cl-sync", clSync)

	var finalizer Finalizer
	if cfg.AltDAEnabled() {
		finalizer = finality.NewAltDAFinalizer(driverCtx, log, cfg, l1, altDA)
	} else {
		finalizer = finality.NewFinalizer(driverCtx, log, cfg, l1)
	}
	sys.Register("finalizer", finalizer)

	attrHandler := attributes.NewAttributesHandler(log, cfg, driverCtx, l2, ec)
	sys.Register("attributes-handler", attrHandler)

	derivationPipeline := derive.NewDerivationPipeline(log, cfg, depSet, verifConfDepth, l1Blobs, altDA, l2, metrics, indexingMode)

	pipelineDeriver := derive.NewPipelineDeriver(driverCtx, derivationPipeline)
	sys.Register("pipeline", pipelineDeriver)

	// Connect components that need force reset notifications to the engine controller
	ec.SetAttributesResetter(attrHandler)
	ec.SetPipelineResetter(pipelineDeriver)

	schedDeriv := NewStepSchedulingDeriver(log)
	sys.Register("step-scheduler", schedDeriv)

	syncDeriver := &SyncDeriver{
		Derivation:          derivationPipeline,
		SafeHeadNotifs:      safeHeadListener,
		CLSync:              clSync,
		Engine:              ec,
		SyncCfg:             syncCfg,
		Config:              cfg,
		L1:                  l1,
		L1Tracker:           l1Tracker,
		L2:                  l2,
		Log:                 log,
		Ctx:                 driverCtx,
		ManagedBySupervisor: indexingMode,
		StepDeriver:         schedDeriv,
	}
	// TODO(#16917) Remove Event System Refactor Comments
	//  Couple SyncDeriver and EngineController for event refactoring
	//  Couple EngDeriver and NewAttributesHandler for event refactoring
	ec.SyncDeriver = syncDeriver
	sys.Register("sync", syncDeriver)
	sys.Register("engine", ec)

	var sequencer sequencing.SequencerIface
	if driverCfg.SequencerEnabled {
		asyncGossiper := async.NewAsyncGossiper(driverCtx, network, log, metrics)
		attrBuilder := derive.NewFetchingAttributesBuilder(cfg, depSet, l1, l2)
		sequencerConfDepth := confdepth.NewConfDepth(driverCfg.SequencerConfDepth, statusTracker.L1Head, l1)
		findL1Origin := sequencing.NewL1OriginSelector(driverCtx, log, cfg, sequencerConfDepth)
		sys.Register("origin-selector", findL1Origin)

		// Connect origin selector to the engine controller for force reset notifications
		ec.SetOriginSelectorResetter(findL1Origin)

		sequencer = sequencing.NewSequencer(driverCtx, log, cfg, attrBuilder, findL1Origin,
			sequencerStateListener, sequencerConductor, asyncGossiper, metrics, ec)
		sys.Register("sequencer", sequencer)
	} else {
		sequencer = sequencing.DisabledSequencer{}
	}

	driverEmitter := sys.Register("driver", nil)
	driver := &Driver{
		StatusTracker: statusTracker,
		Finalizer:     finalizer,
		SyncDeriver:   syncDeriver,
		sched:         schedDeriv,
		emitter:       driverEmitter,
		drain:         drain,
		stateReq:      make(chan chan struct{}),
		forceReset:    make(chan chan struct{}, 10),
		driverConfig:  driverCfg,
		driverCtx:     driverCtx,
		driverCancel:  driverCancel,
		log:           log,
		sequencer:     sequencer,
		metrics:       metrics,
		altSync:       altSync,
	}

	return driver
}
