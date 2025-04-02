package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

type L2ELNodeConfig struct {
	ELNodeConfig
	ID stack.L2ELNodeID
}

type rpcL2ELNode struct {
	rpcELNode

	id stack.L2ELNodeID
}

var _ stack.L2ELNode = (*rpcL2ELNode)(nil)

func NewL2ELNode(cfg L2ELNodeConfig) stack.L2ELNode {
	require.Equal(cfg.T, cfg.ID.ChainID, cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
		id:        cfg.ID,
	}
}

func (r *rpcL2ELNode) ID() stack.L2ELNodeID {
	return r.id
}
