package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
)

type FollowClient struct {
	l2Client     *L2Client
	rollupClient *RollupClient
	followCL     bool
}

type FollowStatus struct {
	SafeL2      eth.L2BlockRef
	FinalizedL2 eth.L2BlockRef
	CurrentL1   eth.L1BlockRef
}

func NewFollowClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *L2ClientConfig) (*FollowClient, error) {
	l2Client, err := NewL2Client(client, log, metrics, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Eth client: %w", err)
	}
	rollupClient := NewRollupClient(client)
	// Check the RPC is from CL
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = rollupClient.SyncStatus(ctx)
	followCL := err == nil
	if followCL {
		log.Info("FollowClient: Following CL")
	} else {
		log.Info("FollowClient: Following EL")
	}
	return &FollowClient{l2Client: l2Client, rollupClient: rollupClient, followCL: followCL}, nil
}

func (s *FollowClient) GetFollowStatus(ctx context.Context) (*FollowStatus, error) {
	if s.followCL {
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
	eFinalized, err := s.l2Client.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch external finalizedRef: %w", err)
	}
	eSafe, err := s.l2Client.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch external safeRef: %w", err)
	}
	return &FollowStatus{FinalizedL2: eFinalized, SafeL2: eSafe}, nil
}
