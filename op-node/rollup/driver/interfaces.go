package driver

import (
	"context"

	opnodemetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/metrics/metered"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/common"
)

type Metrics interface {
	RecordPipelineReset()
	RecordFollowSourceRequest(result string)
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

type SequencerStateListener interface {
	SequencerStarted() error
	SequencerStopped() error
}

type Drain interface {
	Drain() error
	Await() <-chan struct{}
}
