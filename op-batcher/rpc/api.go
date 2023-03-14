package rpc

import (
	"context"
)

type batcherClient interface {
	Start() error
	Stop() error
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

func (a *adminAPI) StopBatcher(_ context.Context) error {
	return a.b.Stop()
}
