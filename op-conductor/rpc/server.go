package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// APIBackend is the backend implementation of the API.
type APIBackend struct {
	log log.Logger
	con *conductor.OpConductor

	// TODO (https://github.com/ethereum-optimism/protocol-quest/issues/45) Add metrics tracer here
}

// NewAPIBackend creates a new APIBackend instance.
func NewAPIBackend(log log.Logger, con *conductor.OpConductor) *APIBackend {
	return &APIBackend{
		log: log,
		con: con,
	}
}

var _ API = (*APIBackend)(nil)

// Active implements API.
func (api *APIBackend) Active(_ context.Context) bool {
	return !api.con.Stopped() && !api.con.Paused()
}

// AddServerAsNonvoter implements API.
func (api *APIBackend) AddServerAsNonvoter(ctx context.Context, id string, addr string) error {
	return api.con.AddServerAsNonvoter(ctx, id, addr)
}

// AddServerAsVoter implements API.
func (api *APIBackend) AddServerAsVoter(ctx context.Context, id string, addr string) error {
	return api.con.AddServerAsVoter(ctx, id, addr)
}

// CommitUnsafePayload implements API.
func (api *APIBackend) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayload) error {
	return api.con.CommitUnsafePayload(ctx, payload)
}

// Leader implements API, returns true if current conductor is leader of the cluster.
func (api *APIBackend) Leader(ctx context.Context) bool {
	return api.con.Leader(ctx)
}

// LeaderWithID implements API, returns the leader's server ID and address (not necessarily the current conductor).
func (api *APIBackend) LeaderWithID(ctx context.Context) (string, string) {
	return api.con.LeaderWithID(ctx)
}

// Pause implements API.
func (api *APIBackend) Pause(ctx context.Context) error {
	return api.con.Pause(ctx)
}

// RemoveServer implements API.
func (api *APIBackend) RemoveServer(ctx context.Context, id string) error {
	return api.con.RemoveServer(ctx, id)
}

// Resume implements API.
func (api *APIBackend) Resume(ctx context.Context) error {
	return api.con.Resume(ctx)
}

// Stop implements API.
func (api *APIBackend) Stop(ctx context.Context) error {
	return api.con.Stop(ctx)
}

// TransferLeader implements API.
func (api *APIBackend) TransferLeader(ctx context.Context) error {
	return api.con.TransferLeader(ctx)
}

// TransferLeaderToServer implements API.
func (api *APIBackend) TransferLeaderToServer(ctx context.Context, id string, addr string) error {
	return api.con.TransferLeaderToServer(ctx, id, addr)
}
