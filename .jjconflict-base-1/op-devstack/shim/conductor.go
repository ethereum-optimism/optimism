package shim

import (
	"github.com/ethereum/go-ethereum/rpc"

	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type ConductorConfig struct {
	CommonConfig
	ID     stack.ConductorID
	Client *rpc.Client
}

type rpcConductor struct {
	commonImpl
	id stack.ConductorID

	client *rpc.Client
	api    conductorRpc.API
}

var _ stack.Conductor = (*rpcConductor)(nil)

func NewConductor(cfg ConductorConfig) stack.Conductor {
	return &rpcConductor{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        conductorRpc.NewAPIClient(cfg.Client),
	}
}

func (r *rpcConductor) ID() stack.ConductorID {
	return r.id
}

func (r *rpcConductor) RpcAPI() conductorRpc.API {
	return r.api
}
