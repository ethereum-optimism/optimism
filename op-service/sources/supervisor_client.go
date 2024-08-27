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

func (cl *SupervisorClient) CheckBlock(ctx context.Context,
	chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (types.SafetyLevel, error) {
	var result types.SafetyLevel
	err := cl.client.CallContext(ctx, &result, "interop_checkBlock",
		(*hexutil.U256)(&chainID), blockHash, hexutil.Uint64(blockNumber))
	if err != nil {
		return types.Unsafe, fmt.Errorf("failed to check Block %s:%d (chain %s): %w", blockHash, blockNumber, chainID, err)
	}
	return result, nil
}

func (cl *SupervisorClient) Close() {
	cl.client.Close()
}
