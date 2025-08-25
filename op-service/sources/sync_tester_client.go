package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncTesterClient struct {
	client client.RPC
}

var _ apis.SyncTester = (*SyncTesterClient)(nil)

func NewSyncTesterClient(client client.RPC) *SyncTesterClient {
	return &SyncTesterClient{
		client: client,
	}
}

func (cl *SyncTesterClient) ChainID(ctx context.Context) (eth.ChainID, error) {
	var result eth.ChainID
	err := cl.client.CallContext(ctx, &result, "eth_chainId")
	return result, err
}
