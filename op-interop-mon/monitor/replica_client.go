package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// ReplicaClient is the narrow view the divergence collector needs of a single
// supernode replica: its current finalized frontier and its super root at a
// given timestamp.
type ReplicaClient interface {
	// SyncStatus returns the replica's current sync status, including the
	// finalized L2 timestamp used to pick a comparison point.
	SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error)
	// SuperRootAtTimestamp returns the replica's super root at ts.
	SuperRootAtTimestamp(ctx context.Context, ts uint64) (eth.SuperRootAtTimestampResponse, error)
	// Endpoint identifies this replica in logs and errors.
	Endpoint() string
	Close()
}

// supernodeReplicaClient adds endpoint identification to the shared
// sources.SuperNodeClient RPC wrapper, so the divergence monitor reuses the
// same superroot_atTimestamp / supernode_syncStatus wire format (and its tests)
// rather than re-issuing CallContext.
type supernodeReplicaClient struct {
	*sources.SuperNodeClient
	endpoint string
}

var _ ReplicaClient = (*supernodeReplicaClient)(nil)

// NewSupernodeReplicaClient dials a supernode replica's RPC endpoint.
func NewSupernodeReplicaClient(endpoint string, log log.Logger) (*supernodeReplicaClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("supernode replica endpoint not configured")
	}
	rpc, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial supernode replica %s: %w", endpoint, err)
	}
	return &supernodeReplicaClient{SuperNodeClient: sources.NewSuperNodeClient(rpc), endpoint: endpoint}, nil
}

func (c *supernodeReplicaClient) Endpoint() string { return c.endpoint }
