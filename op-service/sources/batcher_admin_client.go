package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

type BatcherAdminClient struct {
	client client.RPC
}

var _ apis.BatcherActivity = (*BatcherAdminClient)(nil)

func NewBatcherAdminClient(client client.RPC) *BatcherAdminClient {
	return &BatcherAdminClient{
		client: client,
	}
}

func (cl *BatcherAdminClient) StartBatcher(ctx context.Context) error {
	return cl.client.CallContext(ctx, nil, "admin_startBatcher")
}

func (cl *BatcherAdminClient) StopBatcher(ctx context.Context) error {
	return cl.client.CallContext(ctx, nil, "admin_stopBatcher")
}

func (cl *BatcherAdminClient) FlushBatcher(ctx context.Context) error {
	return cl.client.CallContext(ctx, nil, "admin_flushBatcher")
}
