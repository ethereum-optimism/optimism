package status

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	ErrStatusTrackerNotReady = errors.New("supervisor status tracker not ready")
)

type StatusTracker struct {
	statuses map[eth.ChainID]*NodeSyncStatus
	mu       sync.RWMutex
}

type NodeSyncStatus struct {
	LocalUnsafe eth.BlockRef
	LocalSafe   types.BlockSeal
	CrossUnsafe types.BlockSeal
	CrossSafe   types.BlockSeal
	Finalized   types.BlockSeal
}

func NewStatusTracker(chains []eth.ChainID) *StatusTracker {
	statuses := make(map[eth.ChainID]*NodeSyncStatus)
	for _, chain := range chains {
		statuses[chain] = new(NodeSyncStatus)
	}
	return &StatusTracker{
		statuses: statuses,
	}
}

func (su *StatusTracker) OnEvent(ev event.Event) bool {
	su.mu.Lock()
	defer su.mu.Unlock()

	loadStatusRef := func(chainID eth.ChainID) *NodeSyncStatus {
		v := su.statuses[chainID]
		if v == nil {
			v = &NodeSyncStatus{}
			su.statuses[chainID] = v
		}
		return v
	}
	switch x := ev.(type) {
	case superevents.LocalUnsafeUpdateEvent:
		status := loadStatusRef(x.ChainID)
		status.LocalUnsafe = x.NewLocalUnsafe
	case superevents.LocalSafeUpdateEvent:
		status := loadStatusRef(x.ChainID)
		status.LocalSafe = x.NewLocalSafe.Derived
	case superevents.CrossUnsafeUpdateEvent:
		status := loadStatusRef(x.ChainID)
		status.CrossUnsafe = x.NewCrossUnsafe
	case superevents.CrossSafeUpdateEvent:
		status := loadStatusRef(x.ChainID)
		status.CrossSafe = x.NewCrossSafe.Derived
	case superevents.FinalizedL2UpdateEvent:
		status := loadStatusRef(x.ChainID)
		status.Finalized = x.FinalizedL2
	case superevents.FinalizedL1UpdateEvent:
		log.Debug("Updated finalized L1", "finalizedL1", x.FinalizedL1)
	default:
		return false
	}
	return true
}

func (su *StatusTracker) HasInitializedStatuses() bool {
	su.mu.RLock()
	defer su.mu.RUnlock()

	for _, nodeStatus := range su.statuses {
		if nodeStatus != nil && *nodeStatus != (NodeSyncStatus{}) {
			return true
		}
	}
	return false
}

func (su *StatusTracker) SyncStatus() (eth.SupervisorSyncStatus, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	// after supervisor restarts, there is a timespan where all node's sync status is not fetched yet
	// error immediately until at least single node sync status is available, which is not empty
	if !su.HasInitializedStatuses() {
		return eth.SupervisorSyncStatus{}, ErrStatusTrackerNotReady
	}

	firstChain := true
	var supervisorStatus eth.SupervisorSyncStatus
	supervisorStatus.Chains = make(map[eth.ChainID]*eth.SupervisorChainSyncStatus)

	for chainID, nodeStatus := range su.statuses {
		if firstChain || supervisorStatus.SafeTimestamp >= nodeStatus.CrossSafe.Timestamp {
			supervisorStatus.SafeTimestamp = nodeStatus.CrossSafe.Timestamp
		}
		if firstChain || supervisorStatus.FinalizedTimestamp >= nodeStatus.Finalized.Timestamp {
			supervisorStatus.FinalizedTimestamp = nodeStatus.Finalized.Timestamp
		}

		supervisorStatus.Chains[chainID] = &eth.SupervisorChainSyncStatus{
			LocalUnsafe: nodeStatus.LocalUnsafe,
			LocalSafe:   nodeStatus.LocalSafe.ID(),
			CrossUnsafe: nodeStatus.CrossUnsafe.ID(),
			CrossSafe:   nodeStatus.CrossSafe.ID(),
			Finalized:   nodeStatus.Finalized.ID(),
		}
		firstChain = false
	}
	return supervisorStatus, nil
}
