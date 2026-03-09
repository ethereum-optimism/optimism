package supernode

import (
	"context"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
)

var _ activity.RPCActivity = (*Activity)(nil)

type Activity struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
}

func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Activity {
	return &Activity{
		log:    log,
		chains: chains,
	}
}

func (a *Activity) Name() string { return "supernode" }

func (a *Activity) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
	// No-op: sync status queries chain containers directly.
}

func (a *Activity) RPCNamespace() string    { return "supernode" }
func (a *Activity) RPCService() interface{} { return &api{a: a} }

type api struct{ a *Activity }

// SyncStatus returns all the per-node SyncStatus responses and computes the current localsafe/safe/finalized timestamps.
func (api *api) SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	return api.a.syncStatus(ctx)
}

func (a *Activity) syncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	var (
		statuses              map[eth.ChainID]eth.SyncStatus
		minCurrentL1          eth.BlockID
		minLocalSafeTimestamp uint64
		minSafeTimestamp      uint64
		minFinalizedTimestamp uint64
		safeInitialized       bool
		localSafeInitialized  bool
		finalizedInitialized  bool
	)
	statuses = make(map[eth.ChainID]eth.SyncStatus, len(a.chains))

	for chainID, chain := range a.chains {
		status, err := chain.SyncStatus(ctx)
		if err != nil {
			a.log.Warn("failed to get sync status", "chain_id", chainID.String(), "err", err)
			return eth.SuperNodeSyncStatusResponse{}, err
		}
		if status == nil {
			status = &eth.SyncStatus{}
		}
		statuses[chainID] = *status

		currentL1 := status.CurrentL1.ID()
		if currentL1.Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1
		}

		if !localSafeInitialized {
			minLocalSafeTimestamp = status.LocalSafeL2.Time
			localSafeInitialized = true
		} else if minLocalSafeTimestamp == 0 || status.LocalSafeL2.Time == 0 {
			minLocalSafeTimestamp = 0
		} else if status.LocalSafeL2.Time < minLocalSafeTimestamp {
			minLocalSafeTimestamp = status.LocalSafeL2.Time
		}

		if !safeInitialized {
			minSafeTimestamp = status.SafeL2.Time
			safeInitialized = true
		} else if minSafeTimestamp == 0 || status.SafeL2.Time == 0 {
			minSafeTimestamp = 0
		} else if status.SafeL2.Time < minSafeTimestamp {
			minSafeTimestamp = status.SafeL2.Time
		}

		if !finalizedInitialized {
			minFinalizedTimestamp = status.FinalizedL2.Time
			finalizedInitialized = true
		} else if minFinalizedTimestamp == 0 || status.FinalizedL2.Time == 0 {
			minFinalizedTimestamp = 0
		} else if status.FinalizedL2.Time < minFinalizedTimestamp {
			minFinalizedTimestamp = status.FinalizedL2.Time
		}
	}

	chainIDs := make([]eth.ChainID, 0, len(statuses))
	for chainID := range statuses {
		chainIDs = append(chainIDs, chainID)
	}
	slices.SortFunc(chainIDs, func(a, b eth.ChainID) int { return a.Cmp(b) })

	return eth.SuperNodeSyncStatusResponse{
		Chains:             statuses,
		ChainIDs:           chainIDs,
		CurrentL1:          minCurrentL1,
		SafeTimestamp:      minSafeTimestamp,
		LocalSafeTimestamp: minLocalSafeTimestamp,
		FinalizedTimestamp: minFinalizedTimestamp,
	}, nil
}
