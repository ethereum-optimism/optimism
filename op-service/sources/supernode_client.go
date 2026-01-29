package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SuperNodeClient struct {
	rpc client.RPC
}

func NewSuperNodeClient(rpc client.RPC) *SuperNodeClient {
	return &SuperNodeClient{
		rpc: rpc,
	}
}

func (c *SuperNodeClient) SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (result eth.SuperRootAtTimestampResponse, err error) {
	err = c.rpc.CallContext(ctx, &result, "superroot_atTimestamp", timestamp)
	return
}

func (cl *SuperNodeClient) Close() {
	cl.rpc.Close()
}
