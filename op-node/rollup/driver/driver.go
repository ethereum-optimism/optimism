package driver

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Metrics interface {
	RecordPipelineReset()
	RecordPublishingError()
	RecordDerivationError()

	RecordReceivedUnsafePayload(payload *eth.ExecutionPayloadEnvelope)

	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)
	RecordChannelInputBytes(inputCompressedBytes int)
	RecordHeadChannelOpened()
	RecordChannelTimedOut()
	RecordFrame()

	RecordDerivedBatches(batchType string)

	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)

	SetDerivationIdle(idle bool)

	RecordL1ReorgDepth(d uint64)

	EngineMetrics
	L1FetcherMetrics
	SequencerMetrics
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
	engine.LocalEngineControl
	IsEngineSyncing() bool
	InsertUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error
	TryUpdateEngine(ctx context.Context) error
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
	rollup.Deriver
}

type PlasmaIface interface {
	// Notify L1 finalized head so plasma finality is always behind L1
	Finalize(ref eth.L1BlockRef)
	// Set the engine finalization signal callback
	OnFinalizedHeadSignal(f plasma.HeadSignalFn)

	derive.PlasmaInputFetcher
}

type L1StateIface interface {
	HandleNewL1HeadBlock(head eth.L1BlockRef)
	HandleNewL1SafeBlock(safe eth.L1BlockRef)
	HandleNewL1FinalizedBlock(finalized eth.L1BlockRef)

	L1Head() eth.L1BlockRef
	L1Safe() eth.L1BlockRef
	L1Finalized() eth.L1BlockRef
}

type SequencerIface interface {
	StartBuildingBlock(ctx context.Context) error
	CompleteBuildingBlock(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (*eth.ExecutionPayloadEnvelope, error)
	PlanNextSequencerAction() time.Duration
	RunNextSequencerAction(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (*eth.ExecutionPayloadEnvelope, error)
	BuildingOnto() eth.L2BlockRef
	CancelBuildingBlock(ctx context.Context)
}

type Network interface {
	// PublishL2Payload is called by the driver whenever there is a new payload to publish, synchronously with the driver main loop.
	PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
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

// NewDriver composes an events handler that tracks L1 state, triggers L2 Derivation, and optionally sequences new L2 blocks.
func NewDriver(
	driverCfg *Config,
	cfg *rollup.Config,
	l2 L2Chain,
	l1 L1Chain,
	l1Blobs derive.L1BlobsFetcher,
	altSync AltSync,
	network Network,
	log log.Logger,
	snapshotLog log.Logger,
	metrics Metrics,
	sequencerStateListener SequencerStateListener,
	safeHeadListener rollup.SafeHeadListener,
	syncCfg *sync.Config,
	sequencerConductor conductor.SequencerConductor,
	plasma PlasmaIface,
) *Driver {
	driverCtx, driverCancel := context.WithCancel(context.Background())
	rootDeriver := &rollup.SynchronousDerivers{}
	synchronousEvents := rollup.NewSynchronousEvents(log, driverCtx, rootDeriver)

	l1 = NewMeteredL1Fetcher(l1, metrics)
	l1State := NewL1State(log, metrics)
	sequencerConfDepth := NewConfDepth(driverCfg.SequencerConfDepth, l1State.L1Head, l1)
	findL1Origin := NewL1OriginSelector(log, cfg, sequencerConfDepth)
	verifConfDepth := NewConfDepth(driverCfg.VerifierConfDepth, l1State.L1Head, l1)
	ec := engine.NewEngineController(l2, log, metrics, cfg, syncCfg.SyncMode, synchronousEvents)
	engineResetDeriver := engine.NewEngineResetDeriver(driverCtx, log, cfg, l1, l2, syncCfg, synchronousEvents)
	clSync := clsync.NewCLSync(log, cfg, metrics, synchronousEvents)

	var finalizer Finalizer
	if cfg.PlasmaEnabled() {
		finalizer = finality.NewPlasmaFinalizer(driverCtx, log, cfg, l1, synchronousEvents, plasma)
	} else {
		finalizer = finality.NewFinalizer(driverCtx, log, cfg, l1, synchronousEvents)
	}

	attributesHandler := attributes.NewAttributesHandler(log, cfg, driverCtx, l2, synchronousEvents)
	derivationPipeline := derive.NewDerivationPipeline(log, cfg, verifConfDepth, l1Blobs, plasma, l2, metrics)
	pipelineDeriver := derive.NewPipelineDeriver(driverCtx, derivationPipeline, synchronousEvents)
	attrBuilder := derive.NewFetchingAttributesBuilder(cfg, l1, l2)
	meteredEngine := NewMeteredEngine(cfg, ec, metrics, log) // Only use the metered engine in the sequencer b/c it records sequencing metrics.
	sequencer := NewSequencer(log, cfg, meteredEngine, attrBuilder, findL1Origin, metrics)
	asyncGossiper := async.NewAsyncGossiper(driverCtx, network, log, metrics)

	syncDeriver := &SyncDeriver{
		Derivation:     derivationPipeline,
		Finalizer:      finalizer,
		SafeHeadNotifs: safeHeadListener,
		CLSync:         clSync,
		Engine:         ec,
		SyncCfg:        syncCfg,
		Config:         cfg,
		L1:             l1,
		L2:             l2,
		Emitter:        synchronousEvents,
		Log:            log,
		Ctx:            driverCtx,
		Drain:          synchronousEvents.Drain,
	}
	engDeriv := engine.NewEngDeriver(log, driverCtx, cfg, ec, synchronousEvents)
	schedDeriv := NewStepSchedulingDeriver(log, synchronousEvents)

	driver := &Driver{
		l1State:            l1State,
		SyncDeriver:        syncDeriver,
		sched:              schedDeriv,
		synchronousEvents:  synchronousEvents,
		stateReq:           make(chan chan struct{}),
		forceReset:         make(chan chan struct{}, 10),
		startSequencer:     make(chan hashAndErrorChannel, 10),
		stopSequencer:      make(chan chan hashAndError, 10),
		sequencerActive:    make(chan chan bool, 10),
		sequencerNotifs:    sequencerStateListener,
		driverConfig:       driverCfg,
		driverCtx:          driverCtx,
		driverCancel:       driverCancel,
		log:                log,
		snapshotLog:        snapshotLog,
		sequencer:          sequencer,
		network:            network,
		metrics:            metrics,
		l1HeadSig:          make(chan eth.L1BlockRef, 10),
		l1SafeSig:          make(chan eth.L1BlockRef, 10),
		l1FinalizedSig:     make(chan eth.L1BlockRef, 10),
		unsafeL2Payloads:   make(chan *eth.ExecutionPayloadEnvelope, 10),
		altSync:            altSync,
		asyncGossiper:      asyncGossiper,
		sequencerConductor: sequencerConductor,
	}

	*rootDeriver = []rollup.Deriver{
		syncDeriver,
		engineResetDeriver,
		engDeriv,
		schedDeriv,
		driver,
		clSync,
		pipelineDeriver,
		attributesHandler,
		finalizer,
	}

	return driver
}
