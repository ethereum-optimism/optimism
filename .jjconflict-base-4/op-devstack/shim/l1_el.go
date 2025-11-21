package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L1ELNodeConfig struct {
	ELNodeConfig
}

type rpcL1ELNode struct {
	rpcELNode
}

var _ stack.L1ELNode = (*rpcL1ELNode)(nil)

func NewL1ELNode(cfg L1ELNodeConfig) stack.L1ELNode {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	return &rpcL1ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
	}
}
