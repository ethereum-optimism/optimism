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
	var result eth.SuperRootAtTimestampResponse
	if err := c.rpc.CallContext(ctx, &result, "superroot_atTimestamp", hexutil.Uint64(timestamp)); err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}
	for chainID, optimistic := range result.OptimisticAtTimestamp {
		if optimistic.Output == nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("missing optimistic output for chain %s", chainID)
		}
	}
	return result, nil
}

func (c *SuperNodeClient) SyncStatus(ctx context.Context) (result eth.SuperNodeSyncStatusResponse, err error) {
	err = c.rpc.CallContext(ctx, &result, "supernode_syncStatus")
	return
}

func (cl *SuperNodeClient) Close() {
	cl.rpc.Close()
}
