package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

var ExecutionRPCNamespace = "eth"

// ExecutionProxyBackend implements an execution rpc proxy that only returns results while this conductor is the leader.
type ExecutionProxyBackend struct {
	log    log.Logger
	con    conductor
	client *ethclient.Client
}

var _ ExecutionProxyAPI = (*ExecutionProxyBackend)(nil)

func NewExecutionProxyBackend(log log.Logger, con conductor, client *ethclient.Client) *ExecutionProxyBackend {
	return &ExecutionProxyBackend{
		log:    log,
		con:    con,
		client: client,
	}
}

func (api *ExecutionProxyBackend) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := api.client.Client().Call(&result, "eth_getBlockByNumber", number, fullTx)
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return result, err
}
