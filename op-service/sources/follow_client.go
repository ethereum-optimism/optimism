package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
)

type FollowClient struct {
	rollupClient *RollupClient
}

type FollowStatus struct {
	SafeL2      eth.L2BlockRef
	FinalizedL2 eth.L2BlockRef
	CurrentL1   eth.L1BlockRef
}

func NewFollowClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *L2ClientConfig) (*FollowClient, error) {
	rollupClient := NewRollupClient(client)
	return &FollowClient{rollupClient: rollupClient}, nil
}

func (s *FollowClient) GetFollowStatus(ctx context.Context) (*FollowStatus, error) {
	status, err := s.rollupClient.SyncStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch external syncStatus: %w", err)
	}
	return &FollowStatus{
		FinalizedL2: status.FinalizedL2,
		SafeL2:      status.SafeL2,
		CurrentL1:   status.CurrentL1,
	}, nil
}
