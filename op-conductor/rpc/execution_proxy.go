package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
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

func (api *ExecutionProxyBackend) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	block, err := api.client.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if !api.con.Leader(ctx) {
		return nil, ErrNotLeader
	}
	return block, nil
}
