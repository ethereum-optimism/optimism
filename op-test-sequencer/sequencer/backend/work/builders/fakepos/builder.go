package fakepos

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Beacon interface {
	StoreBlobsBundle(slot uint64, bundle *engine.BlobsBundleV1) error
}

type Blockchain interface {
	// All methods are assumed to have identical behavior to the corresponding methods on
	// go-ethereum/ethclient.Client.

	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
}

type Builder struct {
	id  seqtypes.BuilderID
	log log.Logger

	engine     geth.EngineAPI
	beacon     Beacon
	blockchain Blockchain
	genesis    *types.Header
	config     types.BlockType

	registry work.Jobs

	envelopes map[common.Hash]*engine.ExecutionPayloadEnvelope

	withdrawalsIndex  uint64
	finalizedDistance uint64
	safeDistance      uint64
	blockTime         uint64
}

var _ work.Builder = (*Builder)(nil)

func NewBuilder(ctx context.Context, id seqtypes.BuilderID, opts *work.ServiceOpts, config *Config) (work.Builder, error) {
	genesis, err := config.Backend.HeaderByNumber(context.Background(), new(big.Int))
	if err != nil {
		return nil, fmt.Errorf("get genesis header: %w", err)
	}
	return &Builder{
		id:                id,
		log:               opts.Log,
		genesis:           genesis,
		config:            config.ChainConfig,
		registry:          opts.Jobs,
		engine:            config.EngineAPI,
		beacon:            config.Beacon,
		blockchain:        config.Backend,
		withdrawalsIndex:  1001,
		envelopes:         make(map[common.Hash]*engine.ExecutionPayloadEnvelope),
		finalizedDistance: config.FinalizedDistance,
		safeDistance:      config.SafeDistance,
		blockTime:         config.BlockTime,
	}, nil
}

func (b *Builder) Close() error {
	return nil
}

func (b *Builder) ID() seqtypes.BuilderID {
	return b.id
}

func (b *Builder) Register(jobs work.Jobs) {
	b.registry = jobs
}

func (b *Builder) NewJob(ctx context.Context, opts seqtypes.BuildOpts) (work.BuildJob, error) {
	b.log.Debug("FakePoS Builder NewJob request", "opts", opts)

	job := &Job{
		logger: b.log,
		id:     seqtypes.RandomJobID(),
		b:      b,
		parent: opts.Parent,
	}
	if err := b.registry.RegisterJob(job); err != nil {
		return nil, err
	}

	b.log.Info("FakePoS Builder NewJob has registered job", "job_id", job.ID())
	return job, nil
}

func (b *Builder) String() string {
	return "fakepos-builder-" + b.id.String()
}
