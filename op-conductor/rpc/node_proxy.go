package rpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var NodeRPCNamespace = "optimism"

// NodeProxyBackend implements a node rpc proxy that only returns results while this conductor is the leader.
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
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return status, err
}

func (api *NodeProxyBackend) OutputAtBlock(ctx context.Context, blockNumString string) (*eth.OutputResponse, error) {
	blockNum, err := hexutil.DecodeUint64(blockNumString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode block number: %w", err)
	}
	output, err := api.client.OutputAtBlock(ctx, blockNum)
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return output, err
}

func (api *NodeProxyBackend) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	config, err := api.client.RollupConfig(ctx)
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return config, err
}
