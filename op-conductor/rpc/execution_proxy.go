package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var ExecutionRPCNamespace = "eth"

// ExecutionProxyBackend implements an execution rpc proxy with a leadership check before each call.
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
	if err != nil {
		return nil, err
	}
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return result, nil
}
