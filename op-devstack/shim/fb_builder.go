package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type FlashblocksBuilderNodeConfig struct {
	ELNodeConfig
	ID               stack.FlashblocksBuilderID
	Conductor        stack.Conductor
	FlashblocksWsUrl string
}

type flashblocksBuilderNode struct {
	rpcELNode
	l2Client *sources.L2Client

	id        stack.FlashblocksBuilderID
	conductor stack.Conductor

	flashblocksWsUrl string
}

var _ stack.FlashblocksBuilderNode = (*flashblocksBuilderNode)(nil)

func NewFlashblocksBuilderNode(cfg FlashblocksBuilderNodeConfig) stack.FlashblocksBuilderNode {
	require.Equal(cfg.T, cfg.ID.ChainID(), cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	l2Client, err := sources.NewL2Client(cfg.ELNodeConfig.Client, cfg.T.Logger(), nil, sources.L2ClientSimpleConfig(nil, false, 10, 10))
	require.NoError(cfg.T, err)

	return &flashblocksBuilderNode{
		rpcELNode:        newRpcELNode(cfg.ELNodeConfig),
		l2Client:         l2Client,
		id:               cfg.ID,
		conductor:        cfg.Conductor,
		flashblocksWsUrl: cfg.FlashblocksWsUrl,
	}
}

func (r *flashblocksBuilderNode) ID() stack.FlashblocksBuilderID {
	return r.id
}

func (r *flashblocksBuilderNode) Conductor() stack.Conductor {
	return r.conductor
}

func (r *flashblocksBuilderNode) L2EthClient() apis.L2EthClient {
	return r.l2Client
}

func (r *flashblocksBuilderNode) FlashblocksWsUrl() string {
	return r.flashblocksWsUrl
}
