package standardbuilder

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type L1BlockRefByHash interface {
	L1BlockRefByHash(ctx context.Context, h common.Hash) (eth.L1BlockRef, error)
}

type L2BlockRefByHash interface {
	L2BlockRefByHash(ctx context.Context, h common.Hash) (eth.L2BlockRef, error)
}

type Builder struct {
	id  seqtypes.BuilderID
	log log.Logger
	m   metrics.Metricer

	l1       L1BlockRefByHash
	l2       L2BlockRefByHash
	attrPrep derive.AttributesBuilder
	cl       apis.BuildAPI

	onClose func() // always non-nil

	registry work.Jobs
}

var _ work.Builder = (*Builder)(nil)

func NewBuilder(id seqtypes.BuilderID,
	log log.Logger,
	m metrics.Metricer,
	l1 L1BlockRefByHash,
	l2 L2BlockRefByHash,
	attrPrep derive.AttributesBuilder,
	cl apis.BuildAPI,
	registry work.Jobs) *Builder {

	return &Builder{
		id:       id,
		log:      log,
		m:        m,
		l1:       l1,
		l2:       l2,
		attrPrep: attrPrep,
		cl:       cl,
		onClose:  func() {},
		registry: registry,
	}
}

func (b *Builder) NewJob(ctx context.Context, opts *seqtypes.BuildOpts) (work.BuildJob, error) {
	parentRef, err := b.l2.L2BlockRefByHash(ctx, opts.Parent)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve parent-block: %w", err)
	}
	l1OriginRef, err := b.l1.L1BlockRefByHash(ctx, *opts.L1Origin)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L1 origin: %w", err)
	}
	attrs, err := b.attrPrep.PreparePayloadAttributes(ctx, parentRef, l1OriginRef.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare payload attributes: %w", err)
	}
	id := seqtypes.RandomJobID()
	info, err := b.cl.OpenBlock(ctx, parentRef.ID(), attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to open block: %w", err)
	}
	job := &Job{
		id:          id,
		eng:         b.cl,
		payloadInfo: info,
		unregister: func() {
			b.registry.UnregisterJob(id)
		},
	}
	if err := b.registry.RegisterJob(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (b *Builder) Close() error {
	b.onClose()
	return nil
}

func (b *Builder) String() string {
	return "standard-builder-" + b.id.String()
}

func (b *Builder) ID() seqtypes.BuilderID {
	return b.id
}
