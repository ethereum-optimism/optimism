package rpc

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var ErrNotLeader = errors.New("refusing to proxy request to non-leader sequencer")

// API defines the interface for the op-conductor API.
type API interface {
	// OverrideLeader is used to override the leader status, this is only used to return true for Leader() & LeaderWithID() calls.
	// It does not impact the actual raft consensus leadership status. It is supposed to be used when the cluster is unhealthy
	// and the node is the only one up, to allow batcher to be able to connect to the node, so that it could download blocks from the manually started sequencer.
	OverrideLeader(ctx context.Context) error
	// Pause pauses op-conductor.
	Pause(ctx context.Context) error
	// Resume resumes op-conductor.
	Resume(ctx context.Context) error
	// Stop stops op-conductor.
	Stop(ctx context.Context) error
	// Paused returns true if op-conductor is paused.
	Paused(ctx context.Context) (bool, error)
	// Stopped returns true if op-conductor is stopped.
	Stopped(ctx context.Context) (bool, error)
	// SequencerHealthy returns true if the sequencer is healthy.
	SequencerHealthy(ctx context.Context) (bool, error)

	// Consensus related APIs
	// Leader returns true if the server is the leader.
	Leader(ctx context.Context) (bool, error)
	// LeaderWithID returns the current leader's server info.
	LeaderWithID(ctx context.Context) (*consensus.ServerInfo, error)
	// AddServerAsVoter adds a server as a voter to the cluster.
	AddServerAsVoter(ctx context.Context, id string, addr string, version uint64) error
	// AddServerAsNonvoter adds a server as a non-voter to the cluster. non-voter will not participate in leader election.
	AddServerAsNonvoter(ctx context.Context, id string, addr string, version uint64) error
	// RemoveServer removes a server from the cluster.
	RemoveServer(ctx context.Context, id string, version uint64) error
	// TransferLeader transfers leadership to another server.
	TransferLeader(ctx context.Context) error
	// TransferLeaderToServer transfers leadership to a specific server.
	TransferLeaderToServer(ctx context.Context, id string, addr string) error
	// ClusterMembership returns the current cluster membership configuration.
	ClusterMembership(ctx context.Context) (*consensus.ClusterMembership, error)

	// APIs called by op-node
	// Active returns true if op-conductor is active (not paused or stopped).
	Active(ctx context.Context) (bool, error)
	// CommitUnsafePayload commits an unsafe payload (latest head) to the consensus layer.
	CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error
}

// ExecutionProxyAPI defines the methods proxied to the execution rpc backend
// This should include all methods that are called by op-batcher or op-proposer
type ExecutionProxyAPI interface {
	GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error)
}

// NodeProxyAPI defines the methods proxied to the node rpc backend
// This should include all methods that are called by op-batcher or op-proposer
type NodeProxyAPI interface {
	OutputAtBlock(ctx context.Context, blockNumString string) (*eth.OutputResponse, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	RollupConfig(ctx context.Context) (*rollup.Config, error)
}

// NodeProxyAPI defines the methods proxied to the node rpc backend
// This should include all methods that are called by op-batcher or op-proposer
type NodeAdminProxyAPI interface {
	SequencerActive(ctx context.Context) (bool, error)
}
