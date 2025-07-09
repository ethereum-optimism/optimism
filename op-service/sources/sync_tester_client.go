package sources

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
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
