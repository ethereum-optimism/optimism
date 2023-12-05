package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type ProposerDriver interface {
	StartL2OutputSubmitting() error
	StopL2OutputSubmitting() error
}

type adminAPI struct {
	*rpc.CommonAdminAPI
	b ProposerDriver
}

func NewAdminAPI(dr ProposerDriver, m metrics.RPCMetricer, log log.Logger) *adminAPI {
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

func (a *adminAPI) StartProposer(_ context.Context) error {
	return a.b.StartL2OutputSubmitting()
}

func (a *adminAPI) StopProposer(ctx context.Context) error {
	return a.b.StopL2OutputSubmitting()
}
