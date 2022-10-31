package driver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Metrics interface {
	RecordPipelineReset()
	RecordSequencingError()
	RecordPublishingError()
	RecordDerivationError()

	RecordReceivedUnsafePayload(payload *eth.ExecutionPayload)

	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)

	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)

	SetDerivationIdle(idle bool)

	RecordL1ReorgDepth(d uint64)
	CountSequencedTxs(count int)
}

type L1Chain interface {
	derive.L1Fetcher
	L1BlockRefByLabel(context.Context, eth.BlockLabel) (eth.L1BlockRef, error)
}

type L2Chain interface {
	derive.Engine
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
}

type DerivationPipeline interface {
	Reset()
	Step(ctx context.Context) error
	SetUnsafeHead(head eth.L2BlockRef)
	AddUnsafePayload(payload *eth.ExecutionPayload)
	Finalize(ref eth.L1BlockRef)
	FinalizedL1() eth.L1BlockRef
	Finalized() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef
}

type L1StateIface interface {
	HandleNewL1HeadBlock(head eth.L1BlockRef)
	HandleNewL1SafeBlock(safe eth.L1BlockRef)
	HandleNewL1FinalizedBlock(finalized eth.L1BlockRef)

	L1Head() eth.L1BlockRef
	L1Safe() eth.L1BlockRef
	L1Finalized() eth.L1BlockRef
}

type L1OriginSelectorIface interface {
	FindL1Origin(ctx context.Context, l1Head eth.L1BlockRef, l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
}

type SequencerIface interface {
	StartBuildingBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) error
	CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error)

	// createNewBlock builds a new block based on the L2 Head, L1 Origin, and the current mempool.
	CreateNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error)
}

type Network interface {
	// PublishL2Payload is called by the driver whenever there is a new payload to publish, synchronously with the driver main loop.
	PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error
}

// NewDriver composes an events handler that tracks L1 state, triggers L2 derivation, and optionally sequences new L2 blocks.
func NewDriver(driverCfg *Config, cfg *rollup.Config, l2 L2Chain, l1 L1Chain, network Network, log log.Logger, snapshotLog log.Logger, metrics Metrics) *Driver {
	sequencer := NewSequencer(log, cfg, l1, l2)
	l1State := NewL1State(log, metrics)
	findL1Origin := NewL1OriginSelector(log, cfg, l1, driverCfg.SequencerConfDepth)
	verifConfDepth := NewConfDepth(driverCfg.VerifierConfDepth, l1State.L1Head, l1)
	derivationPipeline := derive.NewDerivationPipeline(log, cfg, verifConfDepth, l2, metrics)

	return &Driver{
		l1State:          l1State,
		derivation:       derivationPipeline,
		idleDerivation:   false,
		stateReq:         make(chan chan struct{}),
		forceReset:       make(chan chan struct{}, 10),
		config:           cfg,
		driverConfig:     driverCfg,
		done:             make(chan struct{}),
		log:              log,
		snapshotLog:      snapshotLog,
		l1:               l1,
		l2:               l2,
		l1OriginSelector: findL1Origin,
		sequencer:        sequencer,
		network:          network,
		metrics:          metrics,
		l1HeadSig:        make(chan eth.L1BlockRef, 10),
		l1SafeSig:        make(chan eth.L1BlockRef, 10),
		l1FinalizedSig:   make(chan eth.L1BlockRef, 10),
		unsafeL2Payloads: make(chan *eth.ExecutionPayload, 10),
	}
}
