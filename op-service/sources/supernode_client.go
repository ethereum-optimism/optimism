package sources

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type SuperNodeClient struct {
	rpc client.RPC
}

func NewSuperNodeClient(rpc client.RPC) *SuperNodeClient {
	return &SuperNodeClient{
		rpc: rpc,
	}
}

func (c *SuperNodeClient) SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	result, err := callNonNull[eth.SuperRootAtTimestampResponse](ctx, c.rpc, "superroot_atTimestamp", hexutil.Uint64(timestamp))
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}
	for chainID, optimistic := range result.OptimisticAtTimestamp {
		if optimistic.Output == nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("missing optimistic output for chain %s", chainID)
		}
	}
	return *result, nil
}

func (c *SuperNodeClient) SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	result, err := callNonNull[eth.SuperNodeSyncStatusResponse](ctx, c.rpc, "supernode_syncStatus")
	if err != nil {
		return eth.SuperNodeSyncStatusResponse{}, err
	}
	return *result, nil
}

func (cl *SuperNodeClient) Close() {
	cl.rpc.Close()
}
