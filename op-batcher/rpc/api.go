package rpc

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type batcherClient interface {
	Start() error
	Stop() error
	CloseChannel(ctx context.Context, id derive.ChannelID, frameNumber uint16) error
}

type adminAPI struct {
	b batcherClient
}

func NewAdminAPI(dr batcherClient) *adminAPI {
	return &adminAPI{
		b: dr,
	}
}

func (a *adminAPI) CloseChannel(ctx context.Context, id derive.ChannelID, frameNumber uint16) error {
	return a.b.CloseChannel(ctx, id, frameNumber)
}

func (a *adminAPI) StartBatcher(_ context.Context) error {
	return a.b.Start()
}

func (a *adminAPI) StopBatcher(_ context.Context) error {
	return a.b.Stop()
}
