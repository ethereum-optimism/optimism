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
	ID                stack.ComponentID
	FlashblocksClient *opclient.WSClient
}

type OPRBuilderNode struct {
	rpcELNode
	id                stack.ComponentID
	engineClient      *sources.EngineClient
	flashblocksClient *opclient.WSClient
}

var _ stack.OPRBuilderNode = (*OPRBuilderNode)(nil)

func NewOPRBuilderNode(cfg OPRBuilderNodeConfig) *OPRBuilderNode {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	l2EngineClient, err := sources.NewEngineClient(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.EngineClientDefaultConfig(cfg.RollupCfg))

	require.NoError(cfg.T, err)

	return &OPRBuilderNode{
		rpcELNode:         newRpcELNode(cfg.ELNodeConfig),
		engineClient:      l2EngineClient,
		id:                cfg.ID,
		flashblocksClient: cfg.FlashblocksClient,
	}
}

func (r *OPRBuilderNode) ID() stack.ComponentID {
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
