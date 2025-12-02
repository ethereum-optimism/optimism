package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type OPRBuilderNodeConfig struct {
	ELNodeConfig
	RollupCfg         *rollup.Config
	ID                stack.OPRBuilderNodeID
	FlashblocksClient *opclient.WSClient
}

type OPRBuilderNode struct {
	rpcELNode
	id                stack.OPRBuilderNodeID
	engineClient      *sources.EngineClient
	flashblocksClient *opclient.WSClient
}

var _ stack.OPRBuilderNode = (*OPRBuilderNode)(nil)

func NewOPRBuilderNode(cfg OPRBuilderNodeConfig) *OPRBuilderNode {
	require.Equal(cfg.T, cfg.ID.ChainID(), cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	l2EngineClient, err := sources.NewEngineClient(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.EngineClientDefaultConfig(cfg.RollupCfg))

	require.NoError(cfg.T, err)

	return &OPRBuilderNode{
		rpcELNode:         newRpcELNode(cfg.ELNodeConfig),
		engineClient:      l2EngineClient,
		id:                cfg.ID,
		flashblocksClient: cfg.FlashblocksClient,
	}
}

func (r *OPRBuilderNode) ID() stack.OPRBuilderNodeID {
	return r.id
}

func (r *OPRBuilderNode) L2EthClient() apis.L2EthClient {
	return r.engineClient.L2Client
}

func (r *OPRBuilderNode) FlashblocksClient() *opclient.WSClient {
	return r.flashblocksClient
}

func (r *OPRBuilderNode) L2EngineClient() apis.EngineClient {
	return r.engineClient.EngineAPIClient
}
