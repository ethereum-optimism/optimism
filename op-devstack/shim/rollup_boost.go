package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/stretchr/testify/require"
)

type RollupBoostNodeConfig struct {
	ELNodeConfig
	RollupCfg           *rollup.Config
	ID                  stack.RollupBoostNodeID
	FlashblocksWsClient stack.FlashblocksWSClient
}

type RollupBoostNode struct {
	rpcELNode
	engineClient *sources.EngineClient

	id stack.RollupBoostNodeID

	flashblocksWsClient stack.FlashblocksWSClient
}

var _ stack.RollupBoostNode = (*RollupBoostNode)(nil)

func NewRollupBoostNode(cfg RollupBoostNodeConfig) *RollupBoostNode {
	require.Equal(cfg.T, cfg.ID.ChainID(), cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	l2EngineClient, err := sources.NewEngineClient(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.EngineClientDefaultConfig(cfg.RollupCfg))

	require.NoError(cfg.T, err)

	return &RollupBoostNode{
		rpcELNode:           newRpcELNode(cfg.ELNodeConfig),
		engineClient:        l2EngineClient,
		id:                  cfg.ID,
		flashblocksWsClient: cfg.FlashblocksWsClient,
	}
}

func (r *RollupBoostNode) ID() stack.RollupBoostNodeID {
	return r.id
}

func (r *RollupBoostNode) L2EthClient() apis.L2EthClient {
	return r.engineClient.L2Client
}

func (r *RollupBoostNode) FlashblocksClient() stack.FlashblocksWSClient {
	return r.flashblocksWsClient
}

func (r *RollupBoostNode) L2EngineClient() apis.EngineClient {
	return r.engineClient.EngineAPIClient
}

func (r *RollupBoostNode) ELNode() stack.ELNode {
	return &r.rpcELNode
}
