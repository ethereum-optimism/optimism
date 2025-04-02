package shim

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/stretchr/testify/require"
)

type L1ELNodeConfig struct {
	ELNodeConfig
	ID stack.L1ELNodeID
}

type rpcL1ELNode struct {
	rpcELNode
	id stack.L1ELNodeID
}

var _ stack.L1ELNode = (*rpcL1ELNode)(nil)

func NewL1ELNode(cfg L1ELNodeConfig) stack.L1ELNode {
	require.Equal(cfg.T, cfg.ID.ChainID, cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL1ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
		id:        cfg.ID,
	}
}

func (r *rpcL1ELNode) ID() stack.L1ELNodeID {
	return r.id
}
