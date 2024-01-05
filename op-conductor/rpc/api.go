package rpc

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ServerID is the ID of the server, e.g. "SequencerA"
type ServerID string

// ServerAddr is the address of the server, e.g. "8.8.8.8:50050", "dns:50050"
type ServerAddr string

// API defines the interface for the op-conductor API.
type API interface {
	// Pause pauses op-conductor.
	Pause(ctx context.Context) error
	// Resume resumes op-conductor.
	Resume(ctx context.Context) error
	// Stop stops op-conductor.
	Stop(ctx context.Context) error

	// Consensus related APIs
	// IsLeader returns true if the server is the leader.
	IsLeader(ctx context.Context) bool
	// LeaderWithID returns the current leader address and ID of the cluster.
	LeaderWithID(ctx context.Context) (ServerAddr, ServerID)
	// AddServerAsVoter adds a server as a voter to the cluster.
	AddServerAsVoter(ctx context.Context, id ServerID, addr ServerAddr) error
	// AddServerAsNonvoter adds a server as a non-voter to the cluster. non-voter will not participate in leader election.
	AddServerAsNonvoter(ctx context.Context, id ServerID, addr ServerAddr) error
	// RemoveServer removes a server from the cluster.
	RemoveServer(ctx context.Context, id ServerID) error
	// TransferLeader transfers leadership to another server.
	TransferLeader(ctx context.Context) error
	// TransferLeaderToServer transfers leadership to a specific server.
	TransferLeaderToServer(ctx context.Context, id ServerID, addr ServerAddr) error

	// APIs called by op-node
	// Active returns true if op-conductor is active (not paused).
	Active(ctx context.Context) (bool, error)
	// CommitUnsafePayload commits a unsafe payload (lastest head) to the consensus layer.
	CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayload) error
}
