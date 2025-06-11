package rpc

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var ExecutionMinerRPCNamespace = "miner"

// ExecutionMinerProxyBackend implements an execution rpc proxy.
type ExecutionMinerProxyBackend struct {
	log    log.Logger
	con    conductor
	client *ethclient.Client
}

var _ ExecutionMinerProxyAPI = (*ExecutionMinerProxyBackend)(nil)

func NewExecutionMinerProxyBackend(log log.Logger, con conductor, client *ethclient.Client) *ExecutionMinerProxyBackend {
	return &ExecutionMinerProxyBackend{
		log:    log,
		con:    con,
		client: client,
	}
}

func (api *ExecutionMinerProxyBackend) SetMaxDASize(ctx context.Context, maxTxSize hexutil.Big, maxBlockSize hexutil.Big) (bool, error) {
	var result bool
	err := api.client.Client().Call(&result, "miner_setMaxDASize", maxTxSize, maxBlockSize)
	if err != nil {
		var rpcErr rpc.Error
		switch {
		case errors.As(err, &rpcErr):
			api.log.Debug("proxy miner_setMaxDASize call returned an RPC error",
				"err", err,
				"maxTxSize", maxTxSize,
				"maxBlockSize", maxBlockSize,
				"method", "miner_setMaxDASize")
		default:
			api.log.Warn("proxy miner_setMaxDASize call failed",
				"err", err,
				"maxTxSize", maxTxSize,
				"maxBlockSize", maxBlockSize,
				"method", "miner_setMaxDASize")
		}
		return false, err
	}
	api.log.Debug("successfully proxied miner_setMaxDASize call",
		"maxTxSize", maxTxSize,
		"maxBlockSize", maxBlockSize,
		"result", result)
	return result, nil
}
