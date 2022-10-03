package driver

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Driver struct {
	s *state
}

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

type Downloader interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	Fetch(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type L1Chain interface {
	derive.L1Fetcher
	L1BlockRefByLabel(context.Context, eth.BlockLabel) (eth.L1BlockRef, error)
}

type L2Chain interface {
	derive.Engine
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

type DerivationPipeline interface {
	Reset()
	Step(ctx context.Context) error
	SetUnsafeHead(head eth.L2BlockRef)
	AddUnsafePayload(payload *eth.ExecutionPayload)
	Finalize(ref eth.BlockID)
	Finalized() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef
}

type outputInterface interface {
	// createNewBlock builds a new block based on the L2 Head, L1 Origin, and the current mempool.
	createNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error)
}

type Network interface {
	// PublishL2Payload is called by the driver whenever there is a new payload to publish, synchronously with the driver main loop.
	PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error
}

func NewDriver(driverCfg *Config, cfg *rollup.Config, l2 L2Chain, l1 L1Chain, network Network, log log.Logger, snapshotLog log.Logger, metrics Metrics) *Driver {
	output := &outputImpl{
		Config: cfg,
		dl:     l1,
		l2:     l2,
		log:    log,
	}

	var state *state
	verifConfDepth := NewConfDepth(driverCfg.VerifierConfDepth, func() eth.L1BlockRef { return state.l1Head }, l1)
	derivationPipeline := derive.NewDerivationPipeline(log, cfg, verifConfDepth, l2, metrics)
	state = NewState(driverCfg, log, snapshotLog, cfg, l1, l2, output, derivationPipeline, network, metrics)
	return &Driver{s: state}
}

func (d *Driver) OnL1Head(ctx context.Context, head eth.L1BlockRef) error {
	return d.s.OnL1Head(ctx, head)
}

func (d *Driver) OnL1Safe(ctx context.Context, safe eth.L1BlockRef) error {
	return d.s.OnL1Safe(ctx, safe)
}

func (d *Driver) OnL1Finalized(ctx context.Context, finalized eth.L1BlockRef) error {
	return d.s.OnL1Finalized(ctx, finalized)
}

func (d *Driver) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error {
	return d.s.OnUnsafeL2Payload(ctx, payload)
}

func (d *Driver) ResetDerivationPipeline(ctx context.Context) error {
	return d.s.ResetDerivationPipeline(ctx)
}

func (d *Driver) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return d.s.SyncStatus(ctx)
}

func (d *Driver) Start(ctx context.Context) error {
	return d.s.Start(ctx)
}
func (d *Driver) Close() error {
	return d.s.Close()
}
