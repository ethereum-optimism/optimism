package sources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
)

var ErrFollowSourceCurrentL1NotSupported = errors.New("follow source does not support CurrentL1")

type FollowClient struct {
	l2Client     *L2Client
	rollupClient *RollupClient
	followCL     bool
}

var _ apis.L2FollowClient = (*FollowClient)(nil)

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

func (s *FollowClient) SafeL2(ctx context.Context) (eth.L2BlockRef, error) {
	if s.followCL {
		status, err := s.rollupClient.SyncStatus(ctx)
		if err != nil {
			return eth.L2BlockRef{}, err
		}
		return status.SafeL2, nil
	}
	return s.l2Client.L2BlockRefByLabel(ctx, eth.Safe)
}

func (s *FollowClient) FinalizedL2(ctx context.Context) (eth.L2BlockRef, error) {
	if s.followCL {
		status, err := s.rollupClient.SyncStatus(ctx)
		if err != nil {
			return eth.L2BlockRef{}, err
		}
		return status.FinalizedL2, nil
	}
	return s.l2Client.L2BlockRefByLabel(ctx, eth.Finalized)
}

func (s *FollowClient) CurrentL1(ctx context.Context) (eth.L1BlockRef, error) {
	if s.followCL {
		status, err := s.rollupClient.SyncStatus(ctx)
		if err != nil {
			return eth.L1BlockRef{}, err
		}
		return status.CurrentL1, nil
	}
	return eth.L1BlockRef{}, ErrFollowSourceCurrentL1NotSupported
}
