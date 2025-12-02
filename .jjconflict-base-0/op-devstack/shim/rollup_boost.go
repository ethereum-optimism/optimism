package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type RollupBoostNodeConfig struct {
	ELNodeConfig
	RollupCfg         *rollup.Config
	ID                stack.RollupBoostNodeID
	FlashblocksClient *opclient.WSClient
}

type RollupBoostNode struct {
	rpcELNode
	engineClient *sources.EngineClient

	id stack.RollupBoostNodeID

	flashblocksClient *opclient.WSClient
}

var _ stack.RollupBoostNode = (*RollupBoostNode)(nil)

func NewRollupBoostNode(cfg RollupBoostNodeConfig) *RollupBoostNode {
	require.Equal(cfg.T, cfg.ID.ChainID(), cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	l2EngineClient, err := sources.NewEngineClient(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.EngineClientDefaultConfig(cfg.RollupCfg))

	require.NoError(cfg.T, err)

	return &RollupBoostNode{
		rpcELNode:         newRpcELNode(cfg.ELNodeConfig),
		engineClient:      l2EngineClient,
		id:                cfg.ID,
		flashblocksClient: cfg.FlashblocksClient,
	}
}

func (r *RollupBoostNode) ID() stack.RollupBoostNodeID {
	return r.id
}

func (r *RollupBoostNode) L2EthClient() apis.L2EthClient {
	return r.engineClient.L2Client
}

func (r *RollupBoostNode) FlashblocksClient() *opclient.WSClient {
	return r.flashblocksClient
}

func (r *RollupBoostNode) L2EngineClient() apis.EngineClient {
	return r.engineClient.EngineAPIClient
}

func (r *RollupBoostNode) ELNode() stack.ELNode {
	return &r.rpcELNode
}
