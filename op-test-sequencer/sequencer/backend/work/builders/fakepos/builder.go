package fakepos

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type Beacon interface {
	StoreBlobsBundle(slot uint64, bundle *engine.BlobsBundleV1) error
}

type Blockchain interface {
	CurrentBlock() *types.Header
	GetHeaderByNumber(number uint64) *types.Header
	GetHeaderByHash(hash common.Hash) *types.Header
	CurrentFinalBlock() *types.Header
	CurrentSafeBlock() *types.Header
	Genesis() *types.Block
	Config() *params.ChainConfig
}

type Builder struct {
	id  seqtypes.BuilderID
	log log.Logger

	engine     *catalyst.ConsensusAPI
	beacon     Beacon
	blockchain Blockchain

	registry work.Jobs

	envelopes map[common.Hash]*engine.ExecutionPayloadEnvelope

	withdrawalsIndex  uint64
	finalizedDistance uint64
	safeDistance      uint64
	blockTime         uint64
}

var _ work.Builder = (*Builder)(nil)

func NewBuilder(ctx context.Context, id seqtypes.BuilderID, opts *work.ServiceOpts, config *Config) (work.Builder, error) {
	return &Builder{
		id:                id,
		log:               opts.Log,
		registry:          opts.Jobs,
		engine:            catalyst.NewConsensusAPI(config.GethBackend),
		beacon:            config.Beacon,
		blockchain:        config.GethBackend.BlockChain(),
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
