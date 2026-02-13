package syncstatus

import (
	"context"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
)

func Aggregate(ctx context.Context, log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) (eth.SuperNodeSyncStatusResponse, error) {
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
	statuses = make(map[eth.ChainID]eth.SyncStatus, len(chains))
	chainIDs := make([]eth.ChainID, 0, len(chains))

	for chainID, chain := range chains {
		chainIDs = append(chainIDs, chainID)
		status, err := chain.SyncStatus(ctx)
		if err != nil {
			log.Warn("failed to get sync status", "chain_id", chainID.String(), "err", err)
			return eth.SuperNodeSyncStatusResponse{}, err
		}
		if status == nil {
			status = &eth.SyncStatus{}
		}
		statuses[chainID] = *status

		// Get current L1s — the minimum L1 block that all derivation pipelines and verifiers have processed.
		// This informs callers that the chains' local views have considered at least up to this L1 block.
		currentL1 := status.CurrentL1.ID()
		if currentL1.Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1
		}
		// Also consider the L1 progress of any registered verifiers.
		for _, verifierL1 := range chain.VerifierCurrentL1s() {
			if verifierL1.Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
				minCurrentL1 = verifierL1
			}
		}

		// Conservative aggregation across chains: take the minimum timestamps.
		// If any chain has a zero timestamp (not initialized), the aggregate is zero.
		if !localSafeInitialized {
			minLocalSafeTimestamp = status.LocalSafeL2.Time
			localSafeInitialized = true
		} else if status.LocalSafeL2.Time < minLocalSafeTimestamp {
			minLocalSafeTimestamp = status.LocalSafeL2.Time
		}

		if !safeInitialized {
			minSafeTimestamp = status.SafeL2.Time
			safeInitialized = true
		} else if status.SafeL2.Time < minSafeTimestamp {
			minSafeTimestamp = status.SafeL2.Time
		}

		if !finalizedInitialized {
			minFinalizedTimestamp = status.FinalizedL2.Time
			finalizedInitialized = true
		} else if status.FinalizedL2.Time < minFinalizedTimestamp {
			minFinalizedTimestamp = status.FinalizedL2.Time
		}
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
