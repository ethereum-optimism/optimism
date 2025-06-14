package engstate

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type Metrics interface {
	metrics.ChainRefMetricer
}

type State struct {
	// localUnsafe block is the block that the engine considers "latest".
	// This will never be optimistically assumed from the engine.
	// This is confirmed as "latestValidHash" in the FCU response.
	localUnsafe eth.BlockRef

	// crossSafe block is the block that the engine considers "safe".
	// This will never be optimistically assumed from the engine.
	crossSafe eth.BlockRef

	// finalized block is the block that the engine considers "finalized",
	// or should consider "finalized" after FCU.
	finalized eth.BlockRef

	// true if the last returned newPayload or FCU status was "syncing"
	isSyncing bool

	// syncTarget is the block we have last asked the engine to sync towards,
	// before getting a "SYNCING" status response.
	syncTarget eth.BlockID

	log     log.Logger
	metrics Metrics
}

func NewState(logger log.Logger, m Metrics) *State {
	return &State{
		isSyncing: false,
		log:       logger,
		metrics:   m,
	}
}

func (st *State) IsSyncing() bool {
	return st.isSyncing
}

func (st *State) SetIsSyncing(v bool) {
	// TODO metric?
	st.isSyncing = v
}

func (st *State) SyncTarget() eth.BlockID {
	return st.syncTarget
}

func (st *State) SetSyncTarget(v eth.BlockID) {
	st.log.Trace("Setting sync-target", "sync_target", v)
	st.syncTarget = v
}

func (st *State) LocalUnsafe() eth.BlockRef {
	return st.localUnsafe
}

func (st *State) SetLocalUnsafe(v eth.BlockRef) {
	st.log.Trace("Setting local-unsafe", "local_unsafe", v)
	st.metrics.RecordLocalUnsafe(types.BlockSealFromRef(v))
	st.localUnsafe = v
}

func (st *State) CrossSafe() eth.BlockRef {
	return st.crossSafe
}

func (st *State) SetCrossSafe(v eth.BlockRef) {
	st.log.Trace("Setting cross-safe", "cross_safe", v)
	st.metrics.RecordCrossSafe(types.BlockSealFromRef(v))
	st.crossSafe = v
}

func (st *State) Finalized() eth.BlockRef {
	return st.finalized
}

func (st *State) SetFinalized(v eth.BlockRef) {
	st.log.Trace("Setting finalized", "finalized", v)
	st.metrics.RecordFinalized(types.BlockSealFromRef(v))
	st.finalized = v
}
