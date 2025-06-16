package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type SupervisorConfig struct {
	CommonConfig
	ID     stack.SupervisorID
	Client client.RPC
}

type rpcSupervisor struct {
	commonImpl
	id stack.SupervisorID

	client client.RPC
	api    apis.SupervisorAPI
}

var _ stack.Supervisor = (*rpcSupervisor)(nil)

func NewSupervisor(cfg SupervisorConfig) stack.Supervisor {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcSupervisor{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewSupervisorClient(cfg.Client),
	}
}

func (r *rpcSupervisor) ID() stack.SupervisorID {
	return r.id
}

func (r *rpcSupervisor) AdminAPI() apis.SupervisorAdminAPI {
	return r.api
}

func (r *rpcSupervisor) QueryAPI() apis.SupervisorQueryAPI {
	return r.api
}
