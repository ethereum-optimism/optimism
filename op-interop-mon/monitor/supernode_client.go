package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SupernodeObserverClient is the read-only op-supernode surface the observer needs.
type SupernodeObserverClient interface {
	// SyncStatus returns the supernode's aggregate per-chain sync status.
	SyncStatus(ctx context.Context) (*eth.SuperNodeSyncStatusResponse, error)
	// Heartbeat returns a nil error when the supernode answers the liveness probe.
	Heartbeat(ctx context.Context) error
	Close()
}

// SupernodeClient calls the op-supernode CL RPC (read-only).
type SupernodeClient struct {
	client client.RPC
	log    log.Logger
}

var _ SupernodeObserverClient = (*SupernodeClient)(nil)

func NewSupernodeClient(endpoint string, log log.Logger) (*SupernodeClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("supernode endpoint not configured")
	}
	c, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create supernode client: %w", err)
	}
	return &SupernodeClient{client: c, log: log}, nil
}

func (sc *SupernodeClient) SyncStatus(ctx context.Context) (*eth.SuperNodeSyncStatusResponse, error) {
	var out eth.SuperNodeSyncStatusResponse
	err := sc.client.CallContext(ctx, &out, "supernode_syncStatus")
	return &out, err
}

func (sc *SupernodeClient) Heartbeat(ctx context.Context) error {
	var out hexutil.Bytes
	return sc.client.CallContext(ctx, &out, "heartbeat_check")
}

func (sc *SupernodeClient) Close() { sc.client.Close() }
