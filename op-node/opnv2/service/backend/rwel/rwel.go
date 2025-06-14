package rwel

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/metrics"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel/engstate"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type ExecEngine interface {
	GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error)
	BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockRef, error)
	BlockRefByHash(ctx context.Context, hash common.Hash) (eth.BlockRef, error)
	L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error)
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
}

type Metrics interface {
	engstate.Metrics

	metrics.ChainSequencingMetrics
}

var _ Metrics = (*metrics.ChainMetrics)(nil)

type RWEL struct {
	// - the last propagated forkchoice state
	// - the health (last successful contact)
	// - the syncing situation as reported by past engine API responses

	engine ExecEngine

	chainID eth.ChainID
	cfg     *rollup.Config

	metrics Metrics
	log     log.Logger
	emitter event.Emitter

	state *engstate.State

	mu sync.RWMutex

	ctx context.Context

	id ID
}

func NewRWEL(rootCtx context.Context, id ID, logger log.Logger, eng ExecEngine, m Metrics, cfg *rollup.Config) *RWEL {
	return &RWEL{
		chainID: eth.ChainIDFromBig(cfg.L2ChainID),
		engine:  eng,
		metrics: m,
		cfg:     cfg,
		log:     logger,
		state:   engstate.NewState(logger, m),
		ctx:     WithID(rootCtx, id),
		id:      id,
	}
}

func (r *RWEL) ChainID() eth.ChainID {
	return r.chainID
}

// IsSyncedTo answers if the RWEL can serve things before the given block number
func (r *RWEL) IsSyncedTo(num uint64) bool {
	return false
}

func (r *RWEL) ID() ID {
	return r.id
}

var _ event.AttachEmitter = (*RWEL)(nil)
var _ event.Deriver = (*RWEL)(nil)

func (d *RWEL) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

func (d *RWEL) OnEvent(ctx context.Context, ev event.Event) bool {
	if IDFromContext(ctx) != d.id { // If the event is not for us, ignore it
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	switch x := ev.(type) {
	case rollup.ForceResetEvent:
		panic("not used anymore")
	case PromoteLocalUnsafeEvent:
		d.onPromoteLocalUnsafe(ctx, x)
	case PromoteCrossSafeEvent:
		d.onPromoteCrossSafe(ctx, x)
	case PromoteFinalizedEvent:
		d.onPromoteFinalized(ctx, x)
	case ForkchoiceUpdateRequestEvent:
		d.onForkchoiceUpdateRequest(ctx, x)
	case BuildStartEvent:
		d.onBuildStart(ctx, x)
	case BuildStartedEvent:
		d.onBuildStarted(ctx, x)
	case BuildSealEvent:
		d.onBuildSeal(ctx, x)
	case BuildSealedEvent:
		d.onBuildSealed(ctx, x)
	case BuildInvalidEvent:
		d.onBuildInvalid(ctx, x)
	case BuildCancelEvent:
		d.onBuildCancel(ctx, x)
	case PayloadProcessEvent:
		d.onPayloadProcess(ctx, x)
	case PayloadSuccessEvent:
		d.onPayloadSuccess(ctx, x)
	case PayloadInvalidEvent:
		d.onPayloadInvalid(ctx, x)
	case PollLocalUnsafeRequestEvent:
		d.onPollLocalUnsafeRequest(ctx, x)
	case PollCrossSafeRequestEvent:
		d.onPollCrossSafeRequest(ctx, x)
	case PollFinalizedRequestEvent:
		d.onPollFinalizedRequest(ctx, x)
	case InvalidateBlockRequestEvent:
		d.onInvalidateBlockRequest(ctx, x)
	case TriggerSyncEvent:
		d.onTriggerSync(ctx, x)
	case TryConsolidateAttributesEvent:
		d.onTryConsolidateAttributes(ctx, x)
	default:
		return false
	}
	return true
}
