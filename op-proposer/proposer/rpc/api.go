package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

// ProposerDriver is the interface for starting and stopping the proposer.
type ProposerDriver interface {
	StartL2OutputSubmitting() error
	StopL2OutputSubmitting() error
}

// adminAPI implements the `ProposerAdminServer` interface.
type adminAPI struct {
	*rpc.CommonAdminAPI
	b ProposerDriver
}

var _ apis.ProposerAdminServer = (*adminAPI)(nil)

// NewAdminAPI constructs a new `adminAPI` instance.
func NewAdminAPI(dr ProposerDriver, log log.Logger) *adminAPI {
	return &adminAPI{
		CommonAdminAPI: rpc.NewCommonAdminAPI(log),
		b:              dr,
	}
}

// GetAdminAPI returns the `admin` API.
func GetAdminAPI(api *adminAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "admin",
		Service:   api,
	}
}

// StartProposer starts the proposer.
func (a *adminAPI) StartProposer(_ context.Context) error {
	return a.b.StartL2OutputSubmitting()
}

// StopProposer stops the proposer.
func (a *adminAPI) StopProposer(ctx context.Context) error {
	return a.b.StopL2OutputSubmitting()
}
