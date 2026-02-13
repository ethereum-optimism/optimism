package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var NodeAdminRPCNamespace = "admin"

// NodeAdminProxyAPI implements a node admin rpc proxy with a leadership check to make sure only leader returns the result.
type NodeAdminProxyBackend struct {
	log    log.Logger
	con    conductor
	client *sources.RollupClient
}

var _ NodeAdminProxyAPI = (*NodeAdminProxyBackend)(nil)

// NewNodeAdminProxyBackend creates a new NodeAdminProxyBackend instance.
func NewNodeAdminProxyBackend(log log.Logger, con conductor, client *sources.RollupClient) NodeAdminProxyAPI {
	return &NodeAdminProxyBackend{
		log:    log,
		con:    con,
		client: client,
	}
}

func (api *NodeAdminProxyBackend) SequencerActive(ctx context.Context) (bool, error) {
	active, err := api.client.SequencerActive(ctx)
	if err != nil {
		return false, err
	}
	if !api.con.Leader(ctx) {
		return false, ErrNotLeader
	}
	return active, err
}
