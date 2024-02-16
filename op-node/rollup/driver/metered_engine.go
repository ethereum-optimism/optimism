package driver

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type EngineMetrics interface {
	RecordSequencingError()
	CountSequencedTxs(count int)

	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(duration time.Duration)
}

// MeteredEngine wraps an EngineControl and adds metrics such as block building time diff and sealing time
type MeteredEngine struct {
	inner derive.EngineControl

	cfg     *rollup.Config
	metrics EngineMetrics
	log     log.Logger

	buildingStartTime time.Time
}

func NewMeteredEngine(cfg *rollup.Config, inner derive.EngineControl, metrics EngineMetrics, log log.Logger) *MeteredEngine {
	return &MeteredEngine{
		inner:   inner,
		cfg:     cfg,
		metrics: metrics,
		log:     log,
	}
}

func (m *MeteredEngine) Finalized() eth.L2BlockRef {
	return m.inner.Finalized()
}

func (m *MeteredEngine) UnsafeL2Head() eth.L2BlockRef {
	return m.inner.UnsafeL2Head()
}

func (m *MeteredEngine) SafeL2Head() eth.L2BlockRef {
	return m.inner.SafeL2Head()
}

func (m *MeteredEngine) StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *derive.AttributesWithParent, updateSafe bool) (errType derive.BlockInsertionErrType, err error) {
	m.buildingStartTime = time.Now()
	errType, err = m.inner.StartPayload(ctx, parent, attrs, updateSafe)
	if err != nil {
		m.metrics.RecordSequencingError()
	}
	return errType, err
}

func (m *MeteredEngine) ConfirmPayload(ctx context.Context, agossip async.AsyncGossiper, sequencerConductor conductor.SequencerConductor) (out *eth.ExecutionPayloadEnvelope, errTyp derive.BlockInsertionErrType, err error) {
	sealingStart := time.Now()
	// Actually execute the block and add it to the head of the chain.
	payload, errType, err := m.inner.ConfirmPayload(ctx, agossip, sequencerConductor)
	if err != nil {
		m.metrics.RecordSequencingError()
		return payload, errType, err
	}
	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(m.buildingStartTime)
	m.metrics.RecordSequencerSealingTime(sealTime)
	m.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(m.cfg.BlockTime)*time.Second)

	txnCount := len(payload.ExecutionPayload.Transactions)
	m.metrics.CountSequencedTxs(txnCount)

	ref := m.inner.UnsafeL2Head()

	m.log.Debug("Processed new L2 block", "l2_unsafe", ref, "l1_origin", ref.L1Origin,
		"txs", txnCount, "time", ref.Time, "seal_time", sealTime, "build_time", buildTime)

	return payload, errType, err
}

func (m *MeteredEngine) CancelPayload(ctx context.Context, force bool) error {
	return m.inner.CancelPayload(ctx, force)
}

func (m *MeteredEngine) BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool) {
	return m.inner.BuildingPayload()
}
