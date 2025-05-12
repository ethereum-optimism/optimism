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

func (b *Builder) NewJob(ctx context.Context, opts seqtypes.BuildOpts) (work.BuildJob, error) {
	b.log.Debug("Builder NewJob request", "opts", opts)

	parentRef, err := b.l2.L2BlockRefByHash(ctx, opts.Parent)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve parent-block: %w", err)
	}
	b.log.Debug("Builder NewJob fetched parentRef", "ref", parentRef)

	var l1OriginRefBlkID eth.BlockID

	if opts.L1Origin != nil {
		b.log.Debug("Builder NewJob about to get BlockRefByHash for L1 origin", "opts", opts)

		l1OriginRef, err := b.l1.L1BlockRefByHash(ctx, *opts.L1Origin)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve L1 origin: %w", err)
		}
		l1OriginRefBlkID = l1OriginRef.ID()

		b.log.Debug("Builder NewJob fetched l1OriginRef", "block_id", l1OriginRefBlkID)
	} else {
		l1OriginRefBlkID = parentRef.L1Origin

		b.log.Debug("Builder NewJob using L1Origin value from parentRef, as opts.L1Origin is nil", "block_id", l1OriginRefBlkID)
	}

	attrs, err := b.attrPrep.PreparePayloadAttributes(ctx, parentRef, l1OriginRefBlkID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare payload attributes: %w", err)
	}
	b.log.Debug("Builder NewJob prepared payload attrs", "attrs", attrs)

	id := seqtypes.RandomJobID()
	job := &Job{
		logger:    b.log,
		id:        id,
		eng:       b.cl,
		attrs:     attrs,
		parentRef: parentRef,
		unregister: func() {
			b.registry.UnregisterJob(id)
		},
	}
	if err := b.registry.RegisterJob(job); err != nil {
		return nil, err
	}
	b.log.Info("Builder NewJob has registered job", "job_id", id)
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
