package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type batcherClient interface {
	Start() error
	Stop(ctx context.Context) error
}

type adminAPI struct {
	*rpc.CommonAdminAPI
	b batcherClient
}

func NewAdminAPI(dr batcherClient, m metrics.RPCMetricer, log log.Logger) *adminAPI {
	return &adminAPI{
		CommonAdminAPI: rpc.NewCommonAdminAPI(m, log),
		b:              dr,
	}
}

func GetAdminAPI(api *adminAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "admin",
		Service:   api,
	}
}

func (a *adminAPI) StartBatcher(_ context.Context) error {
	return a.b.Start()
}

func (a *adminAPI) StopBatcher(ctx context.Context) error {
	return a.b.Stop(ctx)
}
