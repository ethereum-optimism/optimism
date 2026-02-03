package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type SuperNodeConfig struct {
	CommonConfig
	ID     stack.SupernodeID
	Client client.RPC
}

type rpcSuperNode struct {
	commonImpl
	id stack.SupernodeID

	client client.RPC
	api    apis.SupernodeQueryAPI
}

var _ stack.Supernode = (*rpcSuperNode)(nil)

func NewSuperNode(cfg SuperNodeConfig) stack.Supernode {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &rpcSuperNode{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		client:     cfg.Client,
		api:        sources.NewSuperNodeClient(cfg.Client),
	}
}

func (r *rpcSuperNode) ID() stack.SupernodeID {
	return r.id
}

func (r *rpcSuperNode) QueryAPI() apis.SupernodeQueryAPI {
	return r.api
}
