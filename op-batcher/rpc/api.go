package rpc

import (
	"context"
)

type batcherClient interface {
	Start() error
	Stop(ctx context.Context) error
}

type adminAPI struct {
	b batcherClient
}

func NewAdminAPI(dr batcherClient) *adminAPI {
	return &adminAPI{
		b: dr,
	}
}

func (a *adminAPI) StartBatcher(_ context.Context) error {
	return a.b.Start()
}

func (a *adminAPI) StopBatcher(ctx context.Context) error {
	return a.b.Stop(ctx)
}
