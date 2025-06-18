package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2ELNodeConfig struct {
	ELNodeConfig
	RollupCfg *rollup.Config
	ID        stack.L2ELNodeID
}

type rpcL2ELNode struct {
	rpcELNode
	l2Client *sources.L2Client

	id stack.L2ELNodeID
}

var _ stack.L2ELNode = (*rpcL2ELNode)(nil)

func NewL2ELNode(cfg L2ELNodeConfig) stack.L2ELNode {
	require.Equal(cfg.T, cfg.ID.ChainID(), cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	require.NotNil(cfg.T, cfg.RollupCfg, "rollup config must be configured")
	l2Client, err := sources.NewL2Client(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.L2ClientSimpleConfig(cfg.RollupCfg, false, 10, 10))
	require.NoError(cfg.T, err)

	return &rpcL2ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
		l2Client:  l2Client,
		id:        cfg.ID,
	}
}

func (r *rpcL2ELNode) ID() stack.L2ELNodeID {
	return r.id
}

func (r *rpcL2ELNode) L2EthClient() apis.L2EthClient {
	return r.l2Client
}
