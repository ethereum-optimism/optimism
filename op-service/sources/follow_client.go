package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FollowClient struct {
	rollupClient *RollupClient
}

type FollowStatus struct {
	SafeL2      eth.L2BlockRef
	LocalSafeL2 eth.L2BlockRef
	FinalizedL2 eth.L2BlockRef
	CurrentL1   eth.L1BlockRef
	Recovery    *FollowRecoveryStatus
}

// FollowRecoveryStatus identifies public deposit-only history after the last
// private checkpoint. Public hashes identify inputs, never private forkchoice heads.
type FollowRecoveryStatus struct {
	Anchor    eth.L2BlockRef        `json:"anchor"`
	Target    eth.L2BlockRef        `json:"target"`
	Safe      eth.L2BlockRef        `json:"safe"`
	Finalized eth.L2BlockRef        `json:"finalized"`
	Prefix    *FollowRecoveryPrefix `json:"prefix,omitempty"`
}

// FollowRecoveryPrefix authenticates a surviving private prefix by ancestry of
// the parent hash in an accepted terminal commitment. Last identifies the
// surviving projection parent. The backwards offset is Parent.Number - Last.Number;
// replacement starts at Last.Number + 1. The parent is attested by the operator
// under the same policy as the rest of the claim; a future verifier must bind it
// to the proven private execution.
type FollowRecoveryPrefix struct {
	Parent eth.BlockID    `json:"parent"`
	Last   eth.L2BlockRef `json:"last"`
}

// FollowSyncStatus extends the ordinary follow response without changing the
// sync-status wire format of nodes that do not serve private projections.
type FollowSyncStatus struct {
	eth.SyncStatus
	Recovery *FollowRecoveryStatus `json:"private_recovery,omitempty"`
}

func NewFollowClient(client client.RPC) (*FollowClient, error) {
	rollupClient := NewRollupClient(client)
	return &FollowClient{rollupClient: rollupClient}, nil
}

func (s *FollowClient) GetFollowStatus(ctx context.Context) (*FollowStatus, error) {
	var status FollowSyncStatus
	err := s.rollupClient.rpc.CallContext(ctx, &status, "optimism_syncStatus")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch external syncStatus: %w", err)
	}
	return &FollowStatus{
		FinalizedL2: status.FinalizedL2,
		SafeL2:      status.SafeL2,
		LocalSafeL2: status.LocalSafeL2,
		CurrentL1:   status.CurrentL1,
		Recovery:    status.Recovery,
	}, nil
}

// RecoveryBlock returns a canonical projection reference only if the source
// verifies that the block contains no sequencer transactions. The snapshot
// target fences requests against concurrent projection reorgs.
func (s *FollowClient) RecoveryBlock(ctx context.Context, number uint64, target eth.BlockID) (eth.L2BlockRef, error) {
	var ref eth.L2BlockRef
	err := s.rollupClient.rpc.CallContext(ctx, &ref, "optimism_recoveryBlock", number, target)
	return ref, err
}
