package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var NodeRPCNamespace = "optimism"

// NodeProxyBackend implements a node rpc proxy with a leadership check before each call.
type NodeProxyBackend struct {
	log    log.Logger
	con    conductor
	client *sources.RollupClient
}

var _ NodeProxyAPI = (*NodeProxyBackend)(nil)

func NewNodeProxyBackend(log log.Logger, con conductor, client *sources.RollupClient) *NodeProxyBackend {
	return &NodeProxyBackend{
		log:    log,
		con:    con,
		client: client,
	}
}

func (api *NodeProxyBackend) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	status, err := api.client.SyncStatus(ctx)
	if err != nil {
		return nil, err
	}
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return status, err
}

func (api *NodeProxyBackend) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	output, err := api.client.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return nil, err
	}
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return output, nil
}

func (api *NodeProxyBackend) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	config, err := api.client.RollupConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return config, nil
}
