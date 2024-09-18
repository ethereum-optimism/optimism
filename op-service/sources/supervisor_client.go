package sources

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SupervisorClient struct {
	client client.RPC
}

func NewSupervisorClient(client client.RPC) *SupervisorClient {
	return &SupervisorClient{
		client: client,
	}
}

func (cl *SupervisorClient) Stop(
	ctx context.Context,
) error {
	var result error
	err := cl.client.CallContext(
		ctx,
		&result,
		"admin_stop")
	if err != nil {
		return fmt.Errorf("failed to stop Supervisor: %w", err)
	}
	return result
}

func (cl *SupervisorClient) Start(
	ctx context.Context,
) error {
	var result error
	err := cl.client.CallContext(
		ctx,
		&result,
		"admin_start")
	if err != nil {
		return fmt.Errorf("failed to start Supervisor: %w", err)
	}
	return result
}

func (cl *SupervisorClient) AddL2RPC(
	ctx context.Context,
	rpc string,
) error {
	var result error
	err := cl.client.CallContext(
		ctx,
		&result,
		"admin_addL2RPC",
		rpc)
	if err != nil {
		return fmt.Errorf("failed to Add L2 to Supervisor (rpc: %s): %w", rpc, err)
	}
	return result
}

func (cl *SupervisorClient) CheckBlock(ctx context.Context,
	chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (types.SafetyLevel, error) {
	var result types.SafetyLevel
	err := cl.client.CallContext(
		ctx,
		&result,
		"supervisor_checkBlock",
		(*hexutil.U256)(&chainID), blockHash, hexutil.Uint64(blockNumber))
	if err != nil {
		return types.Unsafe, fmt.Errorf("failed to check Block %s:%d (chain %s): %w", blockHash, blockNumber, chainID, err)
	}
	return result, nil
}

func (cl *SupervisorClient) Close() {
	cl.client.Close()
}
