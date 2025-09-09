package sequencing

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type SequencerIface interface {
	event.Deriver
	// NextAction returns when the sequencer needs to do the next change, and iff it should do so.
	NextAction() (t time.Time, ok bool)
	Active() bool
	Init(ctx context.Context, active bool) error
	Start(ctx context.Context, head common.Hash) error
	Stop(ctx context.Context) (hash common.Hash, err error)
	SetMaxSafeLag(ctx context.Context, v uint64) error
	OverrideLeader(ctx context.Context) error
	ConductorEnabled(ctx context.Context) bool
	SetRecoverMode(mode bool)
	Close()
}

type L1OriginSelectorIface interface {
	FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
	SetRecoverMode(bool)
}

type Metrics interface {
	SetSequencerState(active bool)
	RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID)
	RecordSequencerReset()
	RecordSequencingError()
	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(time time.Duration)
	CountSequencedTxsInBlock(txns int, deposits int)
}

type SequencerStateListener interface {
	SequencerStarted() error
	SequencerStopped() error
}

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Get() *eth.ExecutionPayloadEnvelope
	Clear()
	Stop()
	Start()
}
